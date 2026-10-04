package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func TestMetrics_EndpointHTTP(t *testing.T) {
	// Initialize sample observations so MetricVecs appear in Prometheus output
	RecordHTTPRequest("GET", "/api/v1/init", http.StatusOK, 10*time.Millisecond)
	IncWSConnections("arena-init", "referee")
	RecordConsensusDuration("accepted", 5*time.Millisecond)

	req := httptest.NewRequest("GET", "/metrics", nil)
	rec := httptest.NewRecorder()

	handler := promhttp.Handler()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from /metrics, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "scoring_http_requests_total") {
		t.Errorf("expected /metrics output to contain scoring_http_requests_total")
	}
	if !strings.Contains(body, "scoring_ws_active_connections") {
		t.Errorf("expected /metrics output to contain scoring_ws_active_connections")
	}
	if !strings.Contains(body, "scoring_consensus_duration_seconds") {
		t.Errorf("expected /metrics output to contain scoring_consensus_duration_seconds")
	}
}

func TestMetrics_WebSocketTracking(t *testing.T) {
	arenaID := "arena-metric-test"
	role := "juri"

	IncWSConnections(arenaID, role)

	req := httptest.NewRequest("GET", "/metrics", nil)
	rec := httptest.NewRecorder()
	promhttp.Handler().ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, `scoring_ws_active_connections{arena_id="arena-metric-test",role="juri"} 1`) {
		t.Errorf("expected active connections to be 1, got metrics: %s", body)
	}

	DecWSConnections(arenaID, role)

	rec2 := httptest.NewRecorder()
	promhttp.Handler().ServeHTTP(rec2, req)

	body2 := rec2.Body.String()
	if !strings.Contains(body2, `scoring_ws_active_connections{arena_id="arena-metric-test",role="juri"} 0`) {
		t.Errorf("expected active connections to be 0 after decrement, got metrics: %s", body2)
	}
}

func TestMetrics_ConsensusDurationTracking(t *testing.T) {
	RecordConsensusDuration("accepted", 5*time.Millisecond)
	RecordConsensusDuration("pending", 2*time.Millisecond)
	RecordConsensusDuration("duplicate", 1*time.Millisecond)

	req := httptest.NewRequest("GET", "/metrics", nil)
	rec := httptest.NewRecorder()
	promhttp.Handler().ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, `scoring_consensus_total{result="accepted"}`) {
		t.Errorf("expected scoring_consensus_total with result=accepted")
	}
	if !strings.Contains(body, `scoring_consensus_total{result="pending"}`) {
		t.Errorf("expected scoring_consensus_total with result=pending")
	}
	if !strings.Contains(body, `scoring_consensus_total{result="duplicate"}`) {
		t.Errorf("expected scoring_consensus_total with result=duplicate")
	}
}
