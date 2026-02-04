package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorusys/aptos-guardian/internal/incidents"
	"github.com/gorusys/aptos-guardian/internal/metrics"
	"github.com/gorusys/aptos-guardian/internal/store"
)

type Handlers struct {
	Store     *store.Store
	Engine    *incidents.Engine
	RPCNames  []string
	DappNames []string
	RPCURLs   map[string]string
	DappURLs  map[string]string
}

func (h *Handlers) Healthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

type StatusResponse struct {
	RecommendedProvider string            `json:"recommended_provider"`
	RPCProviders        []ProviderStatus  `json:"rpc_providers"`
	Dapps               []DappStatus      `json:"dapps"`
	OpenIncidents       []IncidentSummary `json:"open_incidents"`
}

type ProviderStatus struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	Healthy   bool   `json:"healthy"`
	LatencyMs *int64 `json:"latency_ms,omitempty"`
	LastError string `json:"last_error,omitempty"`
}

type DappStatus struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	Healthy   bool   `json:"healthy"`
	LatencyMs *int64 `json:"latency_ms,omitempty"`
}

type IncidentSummary struct {
	ID         int64  `json:"id"`
	EntityType string `json:"entity_type"`
	EntityName string `json:"entity_name"`
	Severity   string `json:"severity"`
	Summary    string `json:"summary"`
	StartedAt  string `json:"started_at"`
}

func (h *Handlers) Status(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()
	resp := StatusResponse{}
	if h.Engine != nil && len(h.RPCNames) > 0 {
		resp.RecommendedProvider = h.Engine.RecommendedRPCProvider(ctx, h.RPCNames, 50)
	}
	for _, name := range h.RPCNames {
		checks, _ := h.Store.RecentChecks(ctx, "rpc", name, 1)
		ps := ProviderStatus{Name: name}
		if h.RPCURLs != nil {
			ps.URL = h.RPCURLs[name]
		}
		if len(checks) > 0 {
			c := checks[0]
			ps.Healthy = c.Success
			if c.LatencyMs.Valid {
				ps.LatencyMs = &c.LatencyMs.Int64
			}
			if c.ErrorCategory.Valid {
				ps.LastError = c.ErrorCategory.String
			}
		}
		resp.RPCProviders = append(resp.RPCProviders, ps)
	}
	for _, name := range h.DappNames {
		checks, _ := h.Store.RecentChecks(ctx, "dapp", name, 1)
		ds := DappStatus{Name: name}
		if h.DappURLs != nil {
			ds.URL = h.DappURLs[name]
		}
		if len(checks) > 0 {
			c := checks[0]
			ds.Healthy = c.Success
			if c.LatencyMs.Valid {
				ds.LatencyMs = &c.LatencyMs.Int64
			}
		}
		resp.Dapps = append(resp.Dapps, ds)
	}
	openList, _ := h.Store.ListIncidents(ctx, store.IncidentStateOpen, 20)
	for _, i := range openList {
		resp.OpenIncidents = append(resp.OpenIncidents, IncidentSummary{
			ID:         i.ID,
			EntityType: i.EntityType,
			EntityName: i.EntityName,
			Severity:   i.Severity,
			Summary:    i.Summary,
			StartedAt:  i.StartedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handlers) ListIncidents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	state := r.URL.Query().Get("state")
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	list, err := h.Store.ListIncidents(r.Context(), state, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	type incidentRow struct {
		ID         int64   `json:"id"`
		EntityType string  `json:"entity_type"`
		EntityName string  `json:"entity_name"`
		EntityURL  string  `json:"entity_url"`
		State      string  `json:"state"`
		Severity   string  `json:"severity"`
		Summary    string  `json:"summary"`
		StartedAt  string  `json:"started_at"`
		EndedAt    *string `json:"ended_at,omitempty"`
	}
	out := make([]incidentRow, 0, len(list))
	for _, i := range list {
		row := incidentRow{
			ID: i.ID, EntityType: i.EntityType, EntityName: i.EntityName, EntityURL: i.EntityURL,
			State: i.State, Severity: i.Severity, Summary: i.Summary,
			StartedAt: i.StartedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
		if i.EndedAt != nil {
			s := i.EndedAt.Format("2006-01-02T15:04:05Z07:00")
			row.EndedAt = &s
		}
		out = append(out, row)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func (h *Handlers) GetIncident(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	idStr := strings.TrimPrefix(r.URL.Path, "/v1/incidents/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "invalid incident id", http.StatusBadRequest)
		return
	}
	inc, err := h.Store.GetIncident(r.Context(), id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	updates, _ := h.Store.IncidentUpdates(r.Context(), id)
	type incidentDetail struct {
		ID         int64  `json:"id"`
		EntityType string `json:"entity_type"`
		EntityName string `json:"entity_name"`
		EntityURL  string `json:"entity_url"`
		State      string `json:"state"`
		Severity   string `json:"severity"`
		Summary    string `json:"summary"`
		StartedAt  string `json:"started_at"`
		EndedAt    string `json:"ended_at,omitempty"`
		Updates    []struct {
			Message   string `json:"message"`
			CreatedAt string `json:"created_at"`
		} `json:"updates"`
	}
	detail := incidentDetail{
		ID: inc.ID, EntityType: inc.EntityType, EntityName: inc.EntityName, EntityURL: inc.EntityURL,
		State: inc.State, Severity: inc.Severity, Summary: inc.Summary,
		StartedAt: inc.StartedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if inc.EndedAt != nil {
		detail.EndedAt = inc.EndedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	for _, u := range updates {
		detail.Updates = append(detail.Updates, struct {
			Message   string `json:"message"`
			CreatedAt string `json:"created_at"`
		}{u.Message, u.CreatedAt.Format("2006-01-02T15:04:05Z07:00")})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(detail)
}

type ReportRequest struct {
	IssueType   string `json:"issue_type"`
	Wallet      string `json:"wallet"`
	Device      string `json:"device"`
	Region      string `json:"region"`
	Description string `json:"description"`
	URL         string `json:"url"`
	TxHash      string `json:"tx_hash"`
	UserAgent   string `json:"user_agent"`
}

func (h *Handlers) Report(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req ReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	req.IssueType = trunc(req.IssueType, store.MaxReportIssueType)
	req.Wallet = trunc(req.Wallet, store.MaxReportWallet)
	req.Device = trunc(req.Device, store.MaxReportDevice)
	req.Region = trunc(req.Region, store.MaxReportRegion)
	req.Description = trunc(req.Description, store.MaxReportDescription)
	req.URL = trunc(req.URL, store.MaxReportURL)
	req.TxHash = trunc(req.TxHash, store.MaxReportTxHash)
	req.UserAgent = trunc(req.UserAgent, store.MaxReportUserAgent)
	if req.IssueType == "" {
		http.Error(w, "issue_type required", http.StatusBadRequest)
		return
	}
	rep := &store.Report{
		IssueType:   req.IssueType,
		Wallet:      req.Wallet,
		Device:      req.Device,
		Region:      req.Region,
		Description: req.Description,
		URL:         req.URL,
		TxHash:      req.TxHash,
		UserAgent:   req.UserAgent,
	}
	if r.UserAgent() != "" {
		rep.UserAgent = r.UserAgent()
	}
	id, err := h.Store.InsertReport(r.Context(), rep)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	metrics.IncReportsTotal()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]int64{"id": id})
}

func (h *Handlers) ListReports(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	list, err := h.Store.ListReports(r.Context(), limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	type redactedReport struct {
		ID          int64  `json:"id"`
		IssueType   string `json:"issue_type"`
		Device      string `json:"device"`
		Region      string `json:"region"`
		Description string `json:"description"`
		CreatedAt   string `json:"created_at"`
	}
	out := make([]redactedReport, 0, len(list))
	for _, r := range list {
		out = append(out, redactedReport{
			ID: r.ID, IssueType: r.IssueType, Device: r.Device, Region: r.Region,
			Description: r.Description, CreatedAt: r.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func trunc(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
