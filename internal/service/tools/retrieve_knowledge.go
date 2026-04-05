package tools

import (
	"context"
	"fmt"

	"knowflow/internal/service/retrieval"
)

type RetrievalService interface {
	Retrieve(ctx context.Context, req retrieval.RetrieveRequest) (retrieval.Result, error)
}

type RetrieveKnowledgeTool struct {
	service RetrievalService
}

func NewRetrieveKnowledgeTool(service RetrievalService) *RetrieveKnowledgeTool {
	return &RetrieveKnowledgeTool{service: service}
}

func (t *RetrieveKnowledgeTool) Execute(ctx context.Context, input map[string]any) (Output, error) {
	userID, _ := input["user_id"].(string)
	query, _ := input["query"].(string)
	topK, _ := input["top_k"].(int)
	if topK == 0 {
		topK = 5
	}
	if query == "" {
		return Output{Status: "error", Error: "query is required"}, fmt.Errorf("query is required")
	}

	result, err := t.service.Retrieve(ctx, retrieval.RetrieveRequest{
		UserID: userID,
		Query:  query,
		TopK:   topK,
	})
	if err != nil {
		return Output{Status: "error", Error: err.Error()}, err
	}
	return Output{Status: "success", Data: result, Meta: input}, nil
}
