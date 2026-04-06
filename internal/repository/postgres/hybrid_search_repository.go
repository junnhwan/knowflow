package postgres

import (
	"context"
	"sort"

	"knowflow/internal/service/retrieval"
)

type chunkSearchRepository interface {
	SearchVector(ctx context.Context, userID string, embedding []float32, limit int) ([]retrieval.Candidate, error)
	SearchKeyword(ctx context.Context, userID string, query string, tokens []string, limit int) ([]retrieval.Candidate, error)
}

type HybridSearchRepository struct {
	documents chunkSearchRepository
	knowledge chunkSearchRepository
}

func NewHybridSearchRepository(documents, knowledge chunkSearchRepository) *HybridSearchRepository {
	return &HybridSearchRepository{
		documents: documents,
		knowledge: knowledge,
	}
}

func (r *HybridSearchRepository) SearchVector(ctx context.Context, userID string, embedding []float32, limit int) ([]retrieval.Candidate, error) {
	documentCandidates, err := r.documents.SearchVector(ctx, userID, embedding, limit)
	if err != nil {
		return nil, err
	}
	knowledgeCandidates, err := r.knowledge.SearchVector(ctx, userID, embedding, limit)
	if err != nil {
		return nil, err
	}

	merged := append(documentCandidates, knowledgeCandidates...)
	sort.SliceStable(merged, func(i, j int) bool {
		if merged[i].VectorScore == merged[j].VectorScore {
			return merged[i].ChunkID < merged[j].ChunkID
		}
		return merged[i].VectorScore > merged[j].VectorScore
	})
	return merged, nil
}

func (r *HybridSearchRepository) SearchKeyword(ctx context.Context, userID string, query string, tokens []string, limit int) ([]retrieval.Candidate, error) {
	documentCandidates, err := r.documents.SearchKeyword(ctx, userID, query, tokens, limit)
	if err != nil {
		return nil, err
	}
	knowledgeCandidates, err := r.knowledge.SearchKeyword(ctx, userID, query, tokens, limit)
	if err != nil {
		return nil, err
	}

	merged := append(documentCandidates, knowledgeCandidates...)
	sort.SliceStable(merged, func(i, j int) bool {
		if merged[i].KeywordScore == merged[j].KeywordScore {
			return merged[i].ChunkID < merged[j].ChunkID
		}
		return merged[i].KeywordScore > merged[j].KeywordScore
	})
	return merged, nil
}
