package tools

import (
	"context"
	"fmt"
	"strings"
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
	title, _ := input["title"].(string)
	summary, _ := input["summary"].(string)
	reviewStatus, _ := input["review_status"].(string)
	dedupeHash, _ := input["dedupe_hash"].(string)
	if sourceType == "" {
		sourceType = "manual"
	}
	if reviewStatus == "" {
		reviewStatus = "draft"
	}
	if content == "" {
		return Output{Status: "error", Error: "content is required"}, fmt.Errorf("content is required")
	}

	keywords := knowledgeservice.NormalizeKeywords(parseKeywords(input["keywords"]))
	qualityScore := parseFloat(input["quality_score"])
	if qualityScore <= 0 {
		qualityScore = knowledgeservice.BuildQualityScore(summary, content, keywords)
	}
	if dedupeHash == "" {
		dedupeHash = knowledgeservice.BuildDedupeHash(title, summary, content)
	}
	entry := knowledge.Entry{
		ID:              t.newID(),
		UserID:          userID,
		SessionID:       sessionID,
		SourceMessageID: sourceMessageID,
		DocumentID:      documentID,
		SourceType:      sourceType,
		Title:           strings.TrimSpace(title),
		Summary:         strings.TrimSpace(summary),
		Content:         content,
		Keywords:        keywords,
		Status:          "pending_index",
		ReviewStatus:    reviewStatus,
		QualityScore:    qualityScore,
		DedupeHash:      strings.TrimSpace(dedupeHash),
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

func parseKeywords(raw any) []string {
	switch value := raw.(type) {
	case nil:
		return nil
	case []string:
		return compactKeywords(value)
	case []any:
		out := make([]string, 0, len(value))
		for _, item := range value {
			text, _ := item.(string)
			if strings.TrimSpace(text) == "" {
				continue
			}
			out = append(out, text)
		}
		return compactKeywords(out)
	default:
		return nil
	}
}

func compactKeywords(keywords []string) []string {
	if len(keywords) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(keywords))
	out := make([]string, 0, len(keywords))
	for _, keyword := range keywords {
		trimmed := strings.TrimSpace(keyword)
		if trimmed == "" {
			continue
		}
		lower := strings.ToLower(trimmed)
		if _, exists := seen[lower]; exists {
			continue
		}
		seen[lower] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func parseFloat(raw any) float64 {
	switch value := raw.(type) {
	case float64:
		return value
	case float32:
		return float64(value)
	case int:
		return float64(value)
	case int64:
		return float64(value)
	default:
		return 0
	}
}
