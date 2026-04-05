package retrieval

import (
	"context"

	"knowflow/internal/platform/llm"
)

func ApplyRerank(ctx context.Context, reranker llm.Reranker, query string, candidates []Candidate, topN int) ([]Candidate, bool, error) {
	if reranker == nil || len(candidates) == 0 {
		return limitCandidates(candidates, topN), false, nil
	}

	documents := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		documents = append(documents, candidate.Content)
	}

	indices, err := reranker.Rerank(ctx, query, documents, topN)
	if err != nil {
		return limitCandidates(candidates, topN), true, err
	}
	if len(indices) == 0 {
		return limitCandidates(candidates, topN), true, nil
	}

	reranked := make([]Candidate, 0, len(indices))
	for _, index := range indices {
		if index >= 0 && index < len(candidates) {
			candidate := candidates[index]
			candidate.FinalScore = float64(len(indices) - len(reranked))
			reranked = append(reranked, candidate)
		}
	}
	if len(reranked) == 0 {
		return limitCandidates(candidates, topN), true, nil
	}
	return reranked, false, nil
}

func limitCandidates(candidates []Candidate, topN int) []Candidate {
	if topN <= 0 || topN >= len(candidates) {
		return append([]Candidate(nil), candidates...)
	}
	return append([]Candidate(nil), candidates[:topN]...)
}
