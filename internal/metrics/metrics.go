package metrics

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP Metrics
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "scoring_http_requests_total",
			Help: "Total count of processed HTTP requests partitioned by method, path pattern, and status code.",
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "scoring_http_request_duration_seconds",
			Help:    "Latency of processed HTTP requests in seconds.",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
		},
		[]string{"method", "path", "status"},
	)

	// WebSocket Metrics
	WSActiveConnections = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "scoring_ws_active_connections",
			Help: "Number of active WebSocket connections per arena and role.",
		},
		[]string{"arena_id", "role"},
	)

	// Consensus Metrics
	ConsensusDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "scoring_consensus_duration_seconds",
			Help:    "Processing duration of judge consensus evaluation in seconds.",
			Buckets: []float64{0.00001, 0.00005, 0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1},
		},
		[]string{"result"},
	)

	ConsensusTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "scoring_consensus_total",
			Help: "Total count of judge consensus submissions by result outcome (accepted, pending, duplicate).",
		},
		[]string{"result"},
	)
)

// RecordHTTPRequest records latency and counts for an HTTP request.
func RecordHTTPRequest(method, path string, statusCode int, duration time.Duration) {
	status := strconv.Itoa(statusCode)
	HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
	HTTPRequestDuration.WithLabelValues(method, path, status).Observe(duration.Seconds())
}

// IncWSConnections increments the active WebSocket connection gauge for an arena and role.
func IncWSConnections(arenaID, role string) {
	if arenaID == "" {
		arenaID = "unknown"
	}
	if role == "" {
		role = "unknown"
	}
	WSActiveConnections.WithLabelValues(arenaID, role).Inc()
}

// DecWSConnections decrements the active WebSocket connection gauge for an arena and role.
func DecWSConnections(arenaID, role string) {
	if arenaID == "" {
		arenaID = "unknown"
	}
	if role == "" {
		role = "unknown"
	}
	WSActiveConnections.WithLabelValues(arenaID, role).Dec()
}

// RecordConsensusDuration records the duration and result of judge consensus evaluation.
func RecordConsensusDuration(result string, duration time.Duration) {
	ConsensusTotal.WithLabelValues(result).Inc()
	ConsensusDuration.WithLabelValues(result).Observe(duration.Seconds())
}
