package tools

import (
	"context"
	"fmt"
)

type DocumentIndexRefresher interface {
	ReindexDocument(ctx context.Context, documentID string) (map[string]any, error)
}

type KnowledgeIndexRefresher interface {
	ReindexKnowledgeEntry(ctx context.Context, entryID string) (map[string]any, error)
}

type RefreshDocumentIndexTool struct {
	documents DocumentIndexRefresher
	knowledge KnowledgeIndexRefresher
}

func NewRefreshDocumentIndexTool(documents DocumentIndexRefresher, knowledge KnowledgeIndexRefresher) *RefreshDocumentIndexTool {
	return &RefreshDocumentIndexTool{
		documents: documents,
		knowledge: knowledge,
	}
}

func (t *RefreshDocumentIndexTool) Execute(ctx context.Context, input map[string]any) (Output, error) {
	documentID, _ := input["document_id"].(string)
	knowledgeEntryID, _ := input["knowledge_entry_id"].(string)
	switch {
	case documentID != "":
		result, err := t.documents.ReindexDocument(ctx, documentID)
		if err != nil {
			return Output{Status: "error", Error: err.Error()}, err
		}
		return Output{Status: "success", Data: result}, nil
	case knowledgeEntryID != "":
		result, err := t.knowledge.ReindexKnowledgeEntry(ctx, knowledgeEntryID)
		if err != nil {
			return Output{Status: "error", Error: err.Error()}, err
		}
		return Output{Status: "success", Data: result}, nil
	default:
		return Output{Status: "error", Error: "document_id or knowledge_entry_id is required"}, fmt.Errorf("document_id or knowledge_entry_id is required")
	}
}
