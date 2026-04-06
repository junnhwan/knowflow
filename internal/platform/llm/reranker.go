package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
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

type DashScopeReranker struct {
	URL         string
	APIKey      string
	Model       string
	Instruction string
	Client      *http.Client
}

func (r DashScopeReranker) Rerank(ctx context.Context, query string, documents []string, topN int) ([]int, error) {
	if len(documents) == 0 {
		return nil, nil
	}
	if strings.TrimSpace(r.URL) == "" {
		return nil, fmt.Errorf("rerank url is required")
	}
	if strings.TrimSpace(r.APIKey) == "" {
		return nil, fmt.Errorf("rerank api key is required")
	}

	body := map[string]any{
		"model":     r.Model,
		"query":     query,
		"documents": documents,
	}
	if topN > 0 {
		body["top_n"] = topN
	}
	if strings.TrimSpace(r.Instruction) != "" {
		body["instruction"] = r.Instruction
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.URL, strings.NewReader(string(payload)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+r.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := r.Client
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("rerank request failed with status %d", resp.StatusCode)
	}

	var result struct {
		Output struct {
			Results []struct {
				Index          int     `json:"index"`
				RelevanceScore float64 `json:"relevance_score"`
			} `json:"results"`
		} `json:"output"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	indices := make([]int, 0, len(result.Output.Results))
	for _, item := range result.Output.Results {
		indices = append(indices, item.Index)
	}
	return indices, nil
}

type FallbackReranker struct {
	Primary  Reranker
	Fallback Reranker
}

func (r FallbackReranker) Rerank(ctx context.Context, query string, documents []string, topN int) ([]int, error) {
	if r.Primary != nil {
		indices, err := r.Primary.Rerank(ctx, query, documents, topN)
		if err == nil && len(indices) > 0 {
			return indices, nil
		}
	}
	if r.Fallback == nil {
		return nil, fmt.Errorf("fallback reranker is required")
	}
	return r.Fallback.Rerank(ctx, query, documents, topN)
}
