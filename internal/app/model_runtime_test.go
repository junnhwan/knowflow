package app

import (
	"testing"

	"knowflow/internal/config"
	"knowflow/internal/platform/llm"
	chatservice "knowflow/internal/service/chat"
)

func TestBuildEmbedderUsesLocalModeByDefault(t *testing.T) {
	embedder := buildEmbedder(config.Config{
		Model: config.ModelConfig{
			Provider:           "local",
			EmbeddingDimension: 64,
		},
	})

	if _, ok := embedder.(llm.LocalHasherEmbedder); !ok {
		t.Fatalf("expected LocalHasherEmbedder, got %T", embedder)
	}
}

func TestBuildEmbedderUsesRemoteOnlyWhenConfigured(t *testing.T) {
	embedder := buildEmbedder(config.Config{
		Model: config.ModelConfig{
			Provider:           "dashscope",
			EmbeddingBaseURL:   "https://example.com/compatible-mode/v1",
			EmbeddingAPIKey:    "embedding-key",
			EmbeddingModel:     "text-embedding-v4",
			EmbeddingDimension: 64,
		},
	})

	if _, ok := embedder.(llm.OpenAICompatibleEmbedder); !ok {
		t.Fatalf("expected OpenAICompatibleEmbedder, got %T", embedder)
	}
}

func TestBuildRerankerUsesLocalModeByDefault(t *testing.T) {
	reranker := buildReranker(config.Config{
		Model: config.ModelConfig{
			Provider: "local",
		},
	})

	if _, ok := reranker.(llm.LocalOverlapReranker); !ok {
		t.Fatalf("expected LocalOverlapReranker, got %T", reranker)
	}
}

func TestBuildRerankerUsesRemoteWithFallbackWhenConfigured(t *testing.T) {
	reranker := buildReranker(config.Config{
		Model: config.ModelConfig{
			Provider:          "dashscope",
			RerankURL:         "https://example.com/reranks",
			RerankAPIKey:      "rerank-key",
			RerankModel:       "qwen3-rerank",
			RerankInstruction: "优先按后端面试相关性排序",
		},
	})

	if _, ok := reranker.(llm.FallbackReranker); !ok {
		t.Fatalf("expected FallbackReranker, got %T", reranker)
	}
}

func TestBuildKnowledgeExtractorUsesRuleFallbackInLocalMode(t *testing.T) {
	extractor := buildKnowledgeExtractor(config.Config{
		Model: config.ModelConfig{
			Provider: "local",
		},
	})

	fallback, ok := extractor.(chatservice.FallbackKnowledgeExtractor)
	if !ok {
		t.Fatalf("expected wrapped extractor, got %T", extractor)
	}
	if fallback.Primary != nil {
		t.Fatalf("expected no remote primary in local mode")
	}
}

func TestBuildKnowledgeExtractorUsesRemotePrimaryWhenConfigured(t *testing.T) {
	extractor := buildKnowledgeExtractor(config.Config{
		Model: config.ModelConfig{
			Provider:  "dashscope",
			BaseURL:   "https://example.com/compatible-mode/v1",
			APIKey:    "chat-key",
			ChatModel: "qwen-plus",
		},
	})

	fallback, ok := extractor.(chatservice.FallbackKnowledgeExtractor)
	if !ok {
		t.Fatalf("expected wrapped extractor, got %T", extractor)
	}
	if fallback.Primary == nil {
		t.Fatalf("expected remote primary to be configured")
	}
}
