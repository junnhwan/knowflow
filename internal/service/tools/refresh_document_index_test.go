package tools

import (
	"context"
	"testing"
)

func TestRefreshDocumentIndexTool_ExecuteReindexesDocument(t *testing.T) {
	documentRefresher := &fakeDocumentRefresher{
		result: map[string]any{
			"target_type": "document",
			"document_id": "doc-1",
			"chunk_count": 3,
			"status":      "indexed",
		},
	}
	knowledgeRefresher := &fakeKnowledgeRefresher{}
	tool := NewRefreshDocumentIndexTool(documentRefresher, knowledgeRefresher)

	output, err := tool.Execute(context.Background(), map[string]any{
		"document_id": "doc-1",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if output.Status != "success" {
		t.Fatalf("expected success, got %s", output.Status)
	}
	if documentRefresher.documentID != "doc-1" {
		t.Fatalf("expected document refresh for doc-1, got %s", documentRefresher.documentID)
	}
}

func TestRefreshDocumentIndexTool_ExecuteReindexesKnowledgeEntry(t *testing.T) {
	documentRefresher := &fakeDocumentRefresher{}
	knowledgeRefresher := &fakeKnowledgeRefresher{
		result: map[string]any{
			"target_type":        "knowledge_entry",
			"knowledge_entry_id": "knowledge-1",
			"chunk_count":        2,
			"status":             "indexed",
		},
	}
	tool := NewRefreshDocumentIndexTool(documentRefresher, knowledgeRefresher)

	output, err := tool.Execute(context.Background(), map[string]any{
		"knowledge_entry_id": "knowledge-1",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if output.Status != "success" {
		t.Fatalf("expected success, got %s", output.Status)
	}
	if knowledgeRefresher.entryID != "knowledge-1" {
		t.Fatalf("expected knowledge refresh for knowledge-1, got %s", knowledgeRefresher.entryID)
	}
}

func TestRefreshDocumentIndexTool_ExecuteRejectsMissingTargets(t *testing.T) {
	tool := NewRefreshDocumentIndexTool(&fakeDocumentRefresher{}, &fakeKnowledgeRefresher{})

	output, err := tool.Execute(context.Background(), map[string]any{})
	if err == nil {
		t.Fatalf("expected missing target error")
	}
	if output.Status != "error" {
		t.Fatalf("expected error output, got %s", output.Status)
	}
}

type fakeDocumentRefresher struct {
	documentID string
	result     map[string]any
	err        error
}

func (f *fakeDocumentRefresher) ReindexDocument(_ context.Context, documentID string) (map[string]any, error) {
	f.documentID = documentID
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}

type fakeKnowledgeRefresher struct {
	entryID string
	result  map[string]any
	err     error
}

func (f *fakeKnowledgeRefresher) ReindexKnowledgeEntry(_ context.Context, entryID string) (map[string]any, error) {
	f.entryID = entryID
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}
