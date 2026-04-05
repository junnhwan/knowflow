package observability

import (
	"strings"
	"testing"
)

func TestMetricsRegistry_RegistersRAGCounters(t *testing.T) {
	reg := NewMetrics()
	reg.RecordRAGHit("demo-user", "s-1")

	body, err := reg.Expose()
	if err != nil {
		t.Fatalf("Expose() error = %v", err)
	}

	if !strings.Contains(body, "knowflow_rag_hit_total") {
		t.Fatalf("expected rag hit metric to be exposed")
	}
}
