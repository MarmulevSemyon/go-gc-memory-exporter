package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MarmulevSemyon/go-gc-memory-exporter/internal/metrics"
)

func TestHealth(t *testing.T) {
	server := NewServer(metrics.NewCollector(100))
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if !strings.Contains(rec.Body.String(), `"status":"ok"`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestMetrics(t *testing.T) {
	server := NewServer(metrics.NewCollector(100))
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "go_gc_cycles_total") {
		t.Fatalf("expected Prometheus metrics, got: %s", rec.Body.String())
	}
}

func TestSetGCPercent(t *testing.T) {
	server := NewServer(metrics.NewCollector(100))
	req := httptest.NewRequest(http.MethodPost, "/gc-percent?value=50", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if server.collector.GCPercent() != 50 {
		t.Fatalf("expected gc percent 50, got %d", server.collector.GCPercent())
	}
}
