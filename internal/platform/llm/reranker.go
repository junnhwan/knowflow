package llm

import (
	"context"
	"sort"
	"strings"
)

type Reranker interface {
	Rerank(ctx context.Context, query string, documents []string, topN int) ([]int, error)
}

type LocalOverlapReranker struct{}

func (LocalOverlapReranker) Rerank(_ context.Context, query string, documents []string, topN int) ([]int, error) {
	queryTerms := strings.Fields(strings.ToLower(query))
	type scoredIndex struct {
		index int
		score int
	}

	scored := make([]scoredIndex, 0, len(documents))
	for index, document := range documents {
		score := 0
		lower := strings.ToLower(document)
		for _, term := range queryTerms {
			if term != "" && strings.Contains(lower, term) {
				score++
			}
		}
		scored = append(scored, scoredIndex{index: index, score: score})
	}

	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score == scored[j].score {
			return scored[i].index < scored[j].index
		}
		return scored[i].score > scored[j].score
	})

	limit := topN
	if limit <= 0 || limit > len(scored) {
		limit = len(scored)
	}

	result := make([]int, 0, limit)
	for _, item := range scored[:limit] {
		result = append(result, item.index)
	}
	return result, nil
}
