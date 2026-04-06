package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDashScopeReranker_RerankCallsRemoteEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer rerank-key" {
			t.Fatalf("unexpected authorization header: %s", got)
		}

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}

		if payload["model"] != "qwen3-rerank" {
			t.Fatalf("unexpected model: %#v", payload["model"])
		}
		if payload["query"] != "什么是 GMP" {
			t.Fatalf("unexpected query: %#v", payload["query"])
		}
		if payload["top_n"] != float64(2) {
			t.Fatalf("unexpected top_n: %#v", payload["top_n"])
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"output": map[string]any{
				"results": []map[string]any{
					{"index": 1, "relevance_score": 0.98},
					{"index": 0, "relevance_score": 0.76},
				},
			},
		})
	}))
	defer server.Close()

	reranker := DashScopeReranker{
		URL:    server.URL,
		APIKey: "rerank-key",
		Model:  "qwen3-rerank",
		Client: server.Client(),
	}

	indices, err := reranker.Rerank(context.Background(), "什么是 GMP", []string{"doc-1", "doc-2"}, 2)
	if err != nil {
		t.Fatalf("Rerank() error = %v", err)
	}

	if len(indices) != 2 {
		t.Fatalf("expected 2 indices, got %d", len(indices))
	}
	if indices[0] != 1 || indices[1] != 0 {
		t.Fatalf("unexpected rerank indices: %#v", indices)
	}
}

func TestFallbackReranker_UsesLocalWhenRemoteFails(t *testing.T) {
	reranker := FallbackReranker{
		Primary:  failingReranker{},
		Fallback: LocalOverlapReranker{},
	}

	indices, err := reranker.Rerank(context.Background(), "redis 记忆", []string{"redis 记忆依赖 recent 和 summary", "无关文本"}, 1)
	if err != nil {
		t.Fatalf("Rerank() error = %v", err)
	}
	if len(indices) != 1 || indices[0] != 0 {
		t.Fatalf("expected fallback rerank to return first document, got %#v", indices)
	}
}

type failingReranker struct{}

func (failingReranker) Rerank(context.Context, string, []string, int) ([]int, error) {
	return nil, context.DeadlineExceeded
}
