package tools

import (
	"context"
	"fmt"
	"time"

	"knowflow/internal/domain/knowledge"
	knowledgeservice "knowflow/internal/service/knowledge"
)

type KnowledgeEntryWriter interface {
	Create(ctx context.Context, entry knowledge.Entry) error
}

type KnowledgeIndexer interface {
	IndexEntry(ctx context.Context, entry knowledge.Entry) (knowledgeservice.IndexResult, error)
}

type UpsertKnowledgeTool struct {
	writer  KnowledgeEntryWriter
	indexer KnowledgeIndexer
	now     func() time.Time
	newID   func() string
}

func NewUpsertKnowledgeTool(writer KnowledgeEntryWriter, indexer KnowledgeIndexer, now func() time.Time, newID func() string) *UpsertKnowledgeTool {
	if now == nil {
		now = time.Now
	}
	if newID == nil {
		newID = func() string {
			return fmt.Sprintf("knowledge-%d", time.Now().UnixNano())
		}
	}
	return &UpsertKnowledgeTool{
		writer:  writer,
		indexer: indexer,
		now:     now,
		newID:   newID,
	}
}

func (t *UpsertKnowledgeTool) Execute(ctx context.Context, input map[string]any) (Output, error) {
	userID, _ := input["user_id"].(string)
	sessionID, _ := input["session_id"].(string)
	sourceMessageID, _ := input["source_message_id"].(string)
	documentID, _ := input["document_id"].(string)
	content, _ := input["content"].(string)
	sourceType, _ := input["source_type"].(string)
	if sourceType == "" {
		sourceType = "manual"
	}
	if content == "" {
		return Output{Status: "error", Error: "content is required"}, fmt.Errorf("content is required")
	}

	entry := knowledge.Entry{
		ID:              t.newID(),
		UserID:          userID,
		SessionID:       sessionID,
		SourceMessageID: sourceMessageID,
		DocumentID:      documentID,
		SourceType:      sourceType,
		Content:         content,
		Status:          "pending_index",
		CreatedAt:       t.now(),
		UpdatedAt:       t.now(),
	}
	if err := t.writer.Create(ctx, entry); err != nil {
		return Output{Status: "error", Error: err.Error()}, err
	}
	indexResult, err := t.indexer.IndexEntry(ctx, entry)
	if err != nil {
		return Output{Status: "error", Error: err.Error()}, err
	}
	return Output{
		Status: "success",
		Data: map[string]any{
			"id":          entry.ID,
			"status":      indexResult.Status,
			"chunk_count": indexResult.ChunkCount,
		},
	}, nil
}
