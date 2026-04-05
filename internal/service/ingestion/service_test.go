package ingestion

import (
	"context"
	"testing"
	"time"

	"knowflow/internal/domain/document"
)

func TestIngestionService_IngestMarkdownDocument(t *testing.T) {
	store := &fakeDocumentStore{}
	svc := NewService(store, fakeEmbedder{}, ServiceConfig{
		ChunkSize:    64,
		ChunkOverlap: 8,
		Now:          func() time.Time { return time.Unix(1700000000, 0) },
		NewID: func() string {
			return "doc-1"
		},
		NewChunkID: func(index int) string {
			return "chunk-" + string(rune('0'+index))
		},
	})

	result, err := svc.Ingest(context.Background(), IngestRequest{
		UserID:     "demo-user",
		SourceName: "rag.md",
		Content:    "# Title\n\nKnowFlow keeps citations.",
	})
	if err != nil {
		t.Fatalf("Ingest() error = %v", err)
	}

	if result.DocumentID == "" {
		t.Fatalf("expected document id")
	}

	if result.ChunkCount == 0 {
		t.Fatalf("expected chunks to be created")
	}
}

type fakeDocumentStore struct {
	document document.Document
	chunks   []document.Chunk
}

func (f *fakeDocumentStore) UpsertDocument(_ context.Context, doc document.Document) (document.Document, error) {
	f.document = doc
	return doc, nil
}

func (f *fakeDocumentStore) ReplaceChunks(_ context.Context, _ string, chunks []document.Chunk) error {
	f.chunks = chunks
	return nil
}

type fakeEmbedder struct{}

func (fakeEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, 0, len(texts))
	for range texts {
		out = append(out, []float32{1, 0, 0, 1})
	}
	return out, nil
}
