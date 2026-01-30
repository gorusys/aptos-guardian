package api

import (
	"net/http"
	"strings"
)

func Router(h *Handlers, metricsPath string, metricsHandler http.Handler, webRoot string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", h.Healthz)
	mux.HandleFunc("/v1/status", h.Status)
	mux.HandleFunc("/v1/incidents", h.ListIncidents)
	mux.HandleFunc("/v1/incidents/", h.incidentIDRoute)
	mux.HandleFunc("/v1/report", h.Report)
	mux.HandleFunc("/v1/reports", h.ListReports)
	if metricsPath != "" && metricsHandler != nil {
		mux.Handle(metricsPath, metricsHandler)
	}
	if webRoot != "" {
		mux.Handle("/", StaticHandler(webRoot))
	}
	return mux
}

func (h *Handlers) incidentIDRoute(w http.ResponseWriter, r *http.Request) {
	if strings.TrimPrefix(r.URL.Path, "/v1/incidents/") == "" {
		h.ListIncidents(w, r)
		return
	}
	h.GetIncident(w, r)
}
