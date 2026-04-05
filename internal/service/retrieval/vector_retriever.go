package retrieval

import (
	"context"

	"knowflow/internal/platform/llm"
)

type SearchStore interface {
	SearchVector(ctx context.Context, userID string, embedding []float32, limit int) ([]Candidate, error)
	SearchKeyword(ctx context.Context, userID string, query string, tokens []string, limit int) ([]Candidate, error)
}

type VectorRetriever struct {
	embedder llm.Embedder
	store    SearchStore
}

func NewVectorRetriever(embedder llm.Embedder, store SearchStore) VectorRetriever {
	return VectorRetriever{embedder: embedder, store: store}
}

func (r VectorRetriever) Retrieve(ctx context.Context, userID string, query ProcessedQuery, limit int) ([]Candidate, error) {
	embeddings, err := r.embedder.Embed(ctx, []string{query.Normalized})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, nil
	}
	return r.store.SearchVector(ctx, userID, embeddings[0], limit)
}
