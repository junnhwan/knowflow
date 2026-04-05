package retrieval

import (
	"context"
	"errors"
	"testing"
)

func TestRetrievalService_HybridRetrieveReturnsCitations(t *testing.T) {
	store := fakeChunkSearchStore{
		vector: []Candidate{
			{
				ChunkID:    "chunk-1",
				DocumentID: "doc-1",
				SourceName: "intro.md",
				Content:    "KnowFlow 通过混合检索和引用式回答降低幻觉。",
			},
		},
		keyword: []Candidate{
			{
				ChunkID:    "chunk-1",
				DocumentID: "doc-1",
				SourceName: "intro.md",
				Content:    "KnowFlow 通过混合检索和引用式回答降低幻觉。",
			},
		},
	}

	svc := NewService(fakeRetrievalEmbedder{}, store, fakeReranker{}, Config{
		VectorTopK:  5,
		KeywordTopK: 5,
		FinalTopK:   5,
		RRFK:        60,
	})

	result, err := svc.Retrieve(context.Background(), RetrieveRequest{
		UserID: "demo-user",
		Query:  "KnowFlow 如何降低幻觉？",
		TopK:   5,
	})
	if err != nil {
		t.Fatalf("Retrieve() error = %v", err)
	}

	if len(result.Citations) == 0 {
		t.Fatalf("expected citations")
	}

	if result.Meta.Fallback {
		t.Fatalf("expected rerank to succeed")
	}
}

func TestRetrievalService_FallsBackWhenRerankFails(t *testing.T) {
	store := fakeChunkSearchStore{
		vector: []Candidate{{
			ChunkID:    "chunk-1",
			DocumentID: "doc-1",
			SourceName: "intro.md",
			Content:    "hybrid retrieval",
		}},
	}

	svc := NewService(fakeRetrievalEmbedder{}, store, fakeFailingReranker{}, Config{
		VectorTopK:  5,
		KeywordTopK: 5,
		FinalTopK:   5,
		RRFK:        60,
	})

	result, err := svc.Retrieve(context.Background(), RetrieveRequest{
		UserID: "demo-user",
		Query:  "hybrid retrieval",
		TopK:   5,
	})
	if err != nil {
		t.Fatalf("Retrieve() error = %v", err)
	}

	if !result.Meta.Fallback {
		t.Fatalf("expected fallback when rerank fails")
	}
}

type fakeChunkSearchStore struct {
	vector  []Candidate
	keyword []Candidate
}

func (f fakeChunkSearchStore) SearchVector(_ context.Context, _ string, _ []float32, _ int) ([]Candidate, error) {
	return f.vector, nil
}

func (f fakeChunkSearchStore) SearchKeyword(_ context.Context, _ string, _ string, _ []string, _ int) ([]Candidate, error) {
	return f.keyword, nil
}

type fakeRetrievalEmbedder struct{}

func (fakeRetrievalEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, 0, len(texts))
	for range texts {
		out = append(out, []float32{1, 2, 3, 4})
	}
	return out, nil
}

type fakeReranker struct{}

func (fakeReranker) Rerank(_ context.Context, _ string, documents []string, topN int) ([]int, error) {
	limit := topN
	if limit <= 0 || limit > len(documents) {
		limit = len(documents)
	}
	indices := make([]int, 0, limit)
	for i := 0; i < limit; i++ {
		indices = append(indices, i)
	}
	return indices, nil
}

type fakeFailingReranker struct{}

func (fakeFailingReranker) Rerank(_ context.Context, _ string, _ []string, _ int) ([]int, error) {
	return nil, errors.New("boom")
}
