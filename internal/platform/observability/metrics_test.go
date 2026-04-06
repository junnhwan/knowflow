package observability

import (
	"strings"
	"testing"
)

func TestMetricsRegistry_RegistersRAGCounters(t *testing.T) {
	reg := NewMetrics()
	reg.RecordRAGHit("demo-user", "s-1")
	reg.RecordGuardrailReject("/api/chat/query", "prompt_injection")
	reg.RecordReindexTask("document", "success")
	reg.RecordKnowledgeExtraction("success")
	reg.RecordKnowledgeDedupe("candidate")
	reg.RecordKnowledgeMerge("success")

	body, err := reg.Expose()
	if err != nil {
		t.Fatalf("Expose() error = %v", err)
	}

	if !strings.Contains(body, "knowflow_rag_hit_total") {
		t.Fatalf("expected rag hit metric to be exposed")
	}
	if !strings.Contains(body, "knowflow_guardrail_reject_total") {
		t.Fatalf("expected guardrail metric to be exposed")
	}
	if !strings.Contains(body, "knowflow_reindex_task_total") {
		t.Fatalf("expected reindex task metric to be exposed")
	}
	if !strings.Contains(body, "knowflow_knowledge_extract_total") {
		t.Fatalf("expected knowledge extract metric to be exposed")
	}
	if !strings.Contains(body, "knowflow_knowledge_dedupe_total") {
		t.Fatalf("expected knowledge dedupe metric to be exposed")
	}
	if !strings.Contains(body, "knowflow_knowledge_merge_total") {
		t.Fatalf("expected knowledge merge metric to be exposed")
	}
}
