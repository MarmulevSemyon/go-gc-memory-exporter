package metrics

import (
	"strings"
	"testing"
)

func TestPrometheusContainsRequiredMetrics(t *testing.T) {
	collector := NewCollector(100)
	body := collector.Prometheus()

	required := []string{
		"go_mem_total_alloc_bytes",
		"go_gc_cycles_total",
		"go_heap_alloc_bytes",
		"go_gc_last_timestamp_seconds",
		"go_gc_percent 100",
		"go_goroutines",
	}

	for _, metric := range required {
		if !strings.Contains(body, metric) {
			t.Fatalf("expected metric %q in body:\n%s", metric, body)
		}
	}
}
