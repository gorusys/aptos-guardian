package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	CheckSuccess = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "aptos_guardian_check_success",
			Help: "1 if last check succeeded, 0 otherwise",
		},
		[]string{"entity_type", "name"},
	)
	LatencyMs = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "aptos_guardian_latency_ms",
			Help: "Last check latency in milliseconds",
		},
		[]string{"entity_type", "name"},
	)
	IncidentsOpen = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "aptos_guardian_incidents_open",
			Help: "Number of open incidents",
		},
	)
	ReportsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "aptos_guardian_reports_total",
			Help: "Total number of user reports submitted",
		},
	)
	BuildInfo = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "aptos_guardian_build_info",
			Help: "Build and version info",
		},
		[]string{"version", "commit", "date"},
	)
)

func RecordCheck(entityType, name string, success bool, latencyMs int64) {
	if success {
		CheckSuccess.WithLabelValues(entityType, name).Set(1)
	} else {
		CheckSuccess.WithLabelValues(entityType, name).Set(0)
	}
	LatencyMs.WithLabelValues(entityType, name).Set(float64(latencyMs))
}

func SetIncidentsOpen(n float64) {
	IncidentsOpen.Set(n)
}

func IncReportsTotal() {
	ReportsTotal.Inc()
}

func SetBuildInfo(version, commit, date string) {
	BuildInfo.WithLabelValues(version, commit, date).Set(1)
}
