package tools

import (
	"context"
	"testing"
	"time"

	knowledgedomain "knowflow/internal/domain/knowledge"
	knowledgeservice "knowflow/internal/service/knowledge"
)

func TestUpsertKnowledgeTool_ExecuteCreatesEntryAndReturnsIndexMeta(t *testing.T) {
	writer := &fakeKnowledgeEntryWriter{}
	indexer := &fakeKnowledgeIndexer{
		result: knowledgeservice.IndexResult{
			EntryID:    "knowledge-1",
			ChunkCount: 2,
			Status:     "indexed",
		},
	}
	tool := NewUpsertKnowledgeTool(writer, indexer, func() time.Time {
		return time.Unix(1700000000, 0)
	}, func() string {
		return "knowledge-1"
	})

	output, err := tool.Execute(context.Background(), map[string]any{
		"user_id":           "demo-user",
		"session_id":        "session-1",
		"source_message_id": "msg-1",
		"document_id":       "doc-1",
		"source_type":       "qa",
		"title":             "GMP 调度核心结论",
		"summary":           "P 负责承载可运行的 G，M 绑定 P 才能执行 Go 代码。",
		"keywords":          []string{"gmp", "调度", "goroutine"},
		"content":           "GMP 调度模型里，P 负责承载可运行的 G。",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(writer.entries) != 1 {
		t.Fatalf("expected one created entry, got %d", len(writer.entries))
	}

	created := writer.entries[0]
	if created.Status != "pending_index" {
		t.Fatalf("expected pending_index before indexing, got %s", created.Status)
	}
	if created.SourceMessageID != "msg-1" {
		t.Fatalf("expected source message id msg-1, got %s", created.SourceMessageID)
	}
	if created.Title != "GMP 调度核心结论" {
		t.Fatalf("expected structured title to be persisted, got %s", created.Title)
	}
	if created.Summary == "" {
		t.Fatal("expected structured summary to be persisted")
	}
	if created.ReviewStatus != "draft" {
		t.Fatalf("expected default review status draft, got %s", created.ReviewStatus)
	}
	if len(created.Keywords) != 3 {
		t.Fatalf("expected keywords to be persisted, got %#v", created.Keywords)
	}

	data, ok := output.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected output data map")
	}
	if data["chunk_count"] != 2 {
		t.Fatalf("expected chunk_count 2, got %#v", data["chunk_count"])
	}
	if data["status"] != "indexed" {
		t.Fatalf("expected indexed status, got %#v", data["status"])
	}
}

func TestUpsertKnowledgeTool_ExecuteReturnsErrorWhenIndexingFails(t *testing.T) {
	writer := &fakeKnowledgeEntryWriter{}
	indexer := &fakeKnowledgeIndexer{err: context.DeadlineExceeded}
	tool := NewUpsertKnowledgeTool(writer, indexer, func() time.Time {
		return time.Unix(1700000000, 0)
	}, func() string {
		return "knowledge-2"
	})

	output, err := tool.Execute(context.Background(), map[string]any{
		"user_id": "demo-user",
		"content": "channel 关闭后仍可继续读取零值。",
	})
	if err == nil {
		t.Fatalf("expected indexing error")
	}
	if output.Status != "error" {
		t.Fatalf("expected error output, got %s", output.Status)
	}
}

func TestUpsertKnowledgeTool_ExecuteRespectsExplicitReviewStatusAndQualityScore(t *testing.T) {
	writer := &fakeKnowledgeEntryWriter{}
	indexer := &fakeKnowledgeIndexer{
		result: knowledgeservice.IndexResult{
			EntryID:    "knowledge-3",
			ChunkCount: 1,
			Status:     "indexed",
		},
	}
	tool := NewUpsertKnowledgeTool(writer, indexer, func() time.Time {
		return time.Unix(1700000000, 0)
	}, func() string {
		return "knowledge-3"
	})

	_, err := tool.Execute(context.Background(), map[string]any{
		"user_id":        "demo-user",
		"content":        "Redis 双层记忆会保留最近窗口，并把更早历史压缩成摘要。",
		"review_status":  "active",
		"quality_score":  0.91,
		"dedupe_hash":    "hash-redis-memory",
		"source_type":    "manual",
		"keywords":       []any{"redis", "memory"},
		"merged_into_id": "",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(writer.entries) != 1 {
		t.Fatalf("expected one created entry, got %d", len(writer.entries))
	}

	created := writer.entries[0]
	if created.ReviewStatus != "active" {
		t.Fatalf("expected review status active, got %s", created.ReviewStatus)
	}
	if created.QualityScore != 0.91 {
		t.Fatalf("expected quality score 0.91, got %v", created.QualityScore)
	}
	if created.DedupeHash != "hash-redis-memory" {
		t.Fatalf("expected dedupe hash to be persisted, got %s", created.DedupeHash)
	}
	if len(created.Keywords) != 2 || created.Keywords[0] != "redis" {
		t.Fatalf("unexpected keywords: %#v", created.Keywords)
	}
}

type fakeKnowledgeEntryWriter struct {
	entries []knowledgedomain.Entry
}

func (f *fakeKnowledgeEntryWriter) Create(_ context.Context, entry knowledgedomain.Entry) error {
	f.entries = append(f.entries, entry)
	return nil
}

type fakeKnowledgeIndexer struct {
	result knowledgeservice.IndexResult
	err    error
	entry  knowledgedomain.Entry
}

func (f *fakeKnowledgeIndexer) IndexEntry(_ context.Context, entry knowledgedomain.Entry) (knowledgeservice.IndexResult, error) {
	f.entry = entry
	if f.err != nil {
		return knowledgeservice.IndexResult{}, f.err
	}
	return f.result, nil
}
