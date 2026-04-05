package tools

import (
	"context"
	"fmt"
)

type DocumentIndexRefresher interface {
	Reindex(ctx context.Context, documentID string) (map[string]any, error)
}

type RefreshDocumentIndexTool struct {
	refresher DocumentIndexRefresher
}

func NewRefreshDocumentIndexTool(refresher DocumentIndexRefresher) *RefreshDocumentIndexTool {
	return &RefreshDocumentIndexTool{refresher: refresher}
}

func (t *RefreshDocumentIndexTool) Execute(ctx context.Context, input map[string]any) (Output, error) {
	documentID, _ := input["document_id"].(string)
	if documentID == "" {
		return Output{Status: "error", Error: "document_id is required"}, fmt.Errorf("document_id is required")
	}

	result, err := t.refresher.Reindex(ctx, documentID)
	if err != nil {
		return Output{Status: "error", Error: err.Error()}, err
	}
	return Output{Status: "success", Data: result}, nil
}
