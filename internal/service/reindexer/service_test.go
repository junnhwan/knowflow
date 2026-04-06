package reindexer

import (
	"context"
	"errors"
	"testing"
	"time"

	documentdomain "knowflow/internal/domain/document"
	knowledgedomain "knowflow/internal/domain/knowledge"
)

func TestService_ProcessPendingReindexesDocumentsAndKnowledgeEntries(t *testing.T) {
	documentSource := &fakeDocumentSource{
		documents: []documentdomain.Document{
			{ID: "doc-1", Status: "pending_index"},
		},
	}
	knowledgeSource := &fakeKnowledgeSource{
		entries: []knowledgedomain.Entry{
			{ID: "knowledge-1", Status: "pending_index"},
		},
	}
	documentReindexer := &fakeDocumentReindexer{}
	knowledgeReindexer := &fakeKnowledgeReindexer{}

	svc := NewService(documentSource, knowledgeSource, documentReindexer, knowledgeReindexer, Config{
		RetryInterval: 30 * time.Second,
		BatchSize:     10,
		Now:           func() time.Time { return time.Unix(1700000100, 0) },
	})

	if err := svc.ProcessPending(context.Background()); err != nil {
		t.Fatalf("ProcessPending() error = %v", err)
	}

	if len(documentReindexer.documentIDs) != 1 || documentReindexer.documentIDs[0] != "doc-1" {
		t.Fatalf("expected doc-1 to be reindexed, got %#v", documentReindexer.documentIDs)
	}
	if len(knowledgeReindexer.entryIDs) != 1 || knowledgeReindexer.entryIDs[0] != "knowledge-1" {
		t.Fatalf("expected knowledge-1 to be reindexed, got %#v", knowledgeReindexer.entryIDs)
	}
}

func TestService_ProcessPendingMarksFailedTargetsAsIndexFailed(t *testing.T) {
	documentSource := &fakeDocumentSource{
		documents: []documentdomain.Document{
			{ID: "doc-2", Status: "pending_index"},
		},
	}
	knowledgeSource := &fakeKnowledgeSource{
		entries: []knowledgedomain.Entry{
			{ID: "knowledge-2", Status: "pending_index"},
		},
	}
	documentReindexer := &fakeDocumentReindexer{err: errors.New("embed failed")}
	knowledgeReindexer := &fakeKnowledgeReindexer{err: errors.New("rerank failed")}
	now := time.Unix(1700000200, 0)

	svc := NewService(documentSource, knowledgeSource, documentReindexer, knowledgeReindexer, Config{
		RetryInterval: 30 * time.Second,
		BatchSize:     10,
		Now:           func() time.Time { return now },
	})

	if err := svc.ProcessPending(context.Background()); err != nil {
		t.Fatalf("ProcessPending() error = %v", err)
	}

	if len(documentSource.statusUpdates) != 1 {
		t.Fatalf("expected one document status update, got %#v", documentSource.statusUpdates)
	}
	if documentSource.statusUpdates[0].status != "index_failed" {
		t.Fatalf("expected document status index_failed, got %s", documentSource.statusUpdates[0].status)
	}
	if len(knowledgeSource.statusUpdates) != 1 {
		t.Fatalf("expected one knowledge status update, got %#v", knowledgeSource.statusUpdates)
	}
	if knowledgeSource.statusUpdates[0].status != "index_failed" {
		t.Fatalf("expected knowledge status index_failed, got %s", knowledgeSource.statusUpdates[0].status)
	}
}

type fakeDocumentSource struct {
	documents      []documentdomain.Document
	statusUpdates  []statusUpdate
	listBeforeTime time.Time
	listLimit      int
}

func (f *fakeDocumentSource) ListPendingForReindex(_ context.Context, before time.Time, limit int) ([]documentdomain.Document, error) {
	f.listBeforeTime = before
	f.listLimit = limit
	return append([]documentdomain.Document(nil), f.documents...), nil
}

func (f *fakeDocumentSource) UpdateStatus(_ context.Context, documentID, status string, updatedAt time.Time) error {
	f.statusUpdates = append(f.statusUpdates, statusUpdate{id: documentID, status: status, updatedAt: updatedAt})
	return nil
}

type fakeKnowledgeSource struct {
	entries        []knowledgedomain.Entry
	statusUpdates  []statusUpdate
	listBeforeTime time.Time
	listLimit      int
}

func (f *fakeKnowledgeSource) ListPendingForReindex(_ context.Context, before time.Time, limit int) ([]knowledgedomain.Entry, error) {
	f.listBeforeTime = before
	f.listLimit = limit
	return append([]knowledgedomain.Entry(nil), f.entries...), nil
}

func (f *fakeKnowledgeSource) UpdateStatus(_ context.Context, entryID, status string, updatedAt time.Time) error {
	f.statusUpdates = append(f.statusUpdates, statusUpdate{id: entryID, status: status, updatedAt: updatedAt})
	return nil
}

type fakeDocumentReindexer struct {
	documentIDs []string
	err         error
}

func (f *fakeDocumentReindexer) ReindexDocument(_ context.Context, documentID string) (map[string]any, error) {
	f.documentIDs = append(f.documentIDs, documentID)
	if f.err != nil {
		return nil, f.err
	}
	return map[string]any{"document_id": documentID}, nil
}

type fakeKnowledgeReindexer struct {
	entryIDs []string
	err      error
}

func (f *fakeKnowledgeReindexer) ReindexKnowledgeEntry(_ context.Context, entryID string) (map[string]any, error) {
	f.entryIDs = append(f.entryIDs, entryID)
	if f.err != nil {
		return nil, f.err
	}
	return map[string]any{"knowledge_entry_id": entryID}, nil
}

type statusUpdate struct {
	id        string
	status    string
	updatedAt time.Time
}
