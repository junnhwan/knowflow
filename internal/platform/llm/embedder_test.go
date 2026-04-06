package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenAICompatibleEmbedder_EmbedCallsRemoteEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/embeddings" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer embedding-key" {
			t.Fatalf("unexpected authorization header: %s", got)
		}

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}

		if payload["model"] != "text-embedding-v4" {
			t.Fatalf("unexpected model: %#v", payload["model"])
		}
		if payload["dimensions"] != float64(64) {
			t.Fatalf("unexpected dimensions: %#v", payload["dimensions"])
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"index": 0, "embedding": []float64{0.1, 0.2}},
				{"index": 1, "embedding": []float64{0.3, 0.4}},
			},
		})
	}))
	defer server.Close()

	embedder := OpenAICompatibleEmbedder{
		BaseURL:    server.URL,
		APIKey:     "embedding-key",
		Model:      "text-embedding-v4",
		Dimensions: 64,
		Client:     server.Client(),
	}

	vectors, err := embedder.Embed(context.Background(), []string{"first", "second"})
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}

	if len(vectors) != 2 {
		t.Fatalf("expected 2 vectors, got %d", len(vectors))
	}
	if len(vectors[0]) != 2 || vectors[0][0] != 0.1 {
		t.Fatalf("unexpected first vector: %#v", vectors[0])
	}
}

func TestFallbackEmbedder_UsesLocalWhenRemoteFails(t *testing.T) {
	embedder := FallbackEmbedder{
		Primary: failingEmbedder{},
		Fallback: LocalHasherEmbedder{
			Dimension: 64,
		},
	}

	vectors, err := embedder.Embed(context.Background(), []string{"go redis rerank"})
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}
	if len(vectors) != 1 {
		t.Fatalf("expected 1 vector, got %d", len(vectors))
	}
	if len(vectors[0]) != 64 {
		t.Fatalf("expected fallback dimension 64, got %d", len(vectors[0]))
	}
}

func TestFallbackEmbedder_UsesLocalWhenRemoteReturnsEmptyVectors(t *testing.T) {
	embedder := FallbackEmbedder{
		Primary: emptyRemoteEmbedder{},
		Fallback: LocalHasherEmbedder{
			Dimension: 64,
		},
	}

	vectors, err := embedder.Embed(context.Background(), []string{"go redis rerank"})
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}
	if len(vectors) != 1 || len(vectors[0]) != 64 {
		t.Fatalf("expected fallback vectors, got %#v", vectors)
	}
}

type failingEmbedder struct{}

func (failingEmbedder) Embed(context.Context, []string) ([][]float32, error) {
	return nil, context.DeadlineExceeded
}

type emptyRemoteEmbedder struct{}

func (emptyRemoteEmbedder) Embed(context.Context, []string) ([][]float32, error) {
	return [][]float32{{}}, nil
}
