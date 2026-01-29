package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/gorusys/aptos-guardian/internal/config"
	"github.com/gorusys/aptos-guardian/internal/incidents"
	"github.com/gorusys/aptos-guardian/internal/store"
)

func setupHandlers(t *testing.T) *Handlers {
	t.Helper()
	cfg, err := config.Load(filepath.Join("..", "..", "configs", "example.yaml"))
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	st, err := store.New(context.Background(), filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	engine := incidents.NewEngine(st, cfg, nil)
	rpcNames := make([]string, 0, len(cfg.RPCProviders))
	rpcURLs := make(map[string]string)
	for _, p := range cfg.RPCProviders {
		rpcNames = append(rpcNames, p.Name)
		rpcURLs[p.Name] = p.URL
	}
	dappNames := make([]string, 0, len(cfg.Dapps))
	dappURLs := make(map[string]string)
	for _, d := range cfg.Dapps {
		dappNames = append(dappNames, d.Name)
		dappURLs[d.Name] = d.URL
	}
	return &Handlers{Store: st, Engine: engine, RPCNames: rpcNames, DappNames: dappNames, RPCURLs: rpcURLs, DappURLs: dappURLs}
}

func TestHealthz(t *testing.T) {
	h := setupHandlers(t)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	h.Healthz(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d", rec.Code)
	}
	if body := rec.Body.String(); body != "ok" {
		t.Errorf("body = %q", body)
	}
}

func TestStatus(t *testing.T) {
	h := setupHandlers(t)
	req := httptest.NewRequest(http.MethodGet, "/v1/status", nil)
	rec := httptest.NewRecorder()
	h.Status(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d", rec.Code)
	}
	var resp StatusResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.RPCProviders) == 0 {
		t.Error("expected rpc_providers")
	}
	if len(resp.Dapps) == 0 {
		t.Error("expected dapps")
	}
}

func TestListIncidents(t *testing.T) {
	h := setupHandlers(t)
	req := httptest.NewRequest(http.MethodGet, "/v1/incidents?state=open&limit=10", nil)
	rec := httptest.NewRecorder()
	h.ListIncidents(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d", rec.Code)
	}
	var list []interface{}
	if err := json.NewDecoder(rec.Body).Decode(&list); err != nil {
		t.Fatalf("decode: %v", err)
	}
}

func TestGetIncident(t *testing.T) {
	h := setupHandlers(t)
	ctx := context.Background()
	id, err := h.Store.OpenIncident(ctx, "rpc", "test", "https://x.com", store.SeverityCrit, "test summary")
	if err != nil {
		t.Fatalf("open incident: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/v1/incidents/"+strconv.FormatInt(id, 10), nil)
	rec := httptest.NewRecorder()
	h.GetIncident(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestReport(t *testing.T) {
	h := setupHandlers(t)
	body := bytes.NewBufferString(`{"issue_type":"rpc_down","description":"cannot connect"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/report", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Report(rec, req)
	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var out map[string]int64
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out["id"] <= 0 {
		t.Errorf("id = %d", out["id"])
	}
}

func TestReport_BadRequest(t *testing.T) {
	h := setupHandlers(t)
	body := bytes.NewBufferString(`{}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/report", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Report(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d", rec.Code)
	}
}

func TestListReports(t *testing.T) {
	h := setupHandlers(t)
	req := httptest.NewRequest(http.MethodGet, "/v1/reports?limit=5", nil)
	rec := httptest.NewRecorder()
	h.ListReports(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d", rec.Code)
	}
}
