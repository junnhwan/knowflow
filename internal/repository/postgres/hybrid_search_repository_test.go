package postgres

import (
	"context"
	"testing"

	"knowflow/internal/service/retrieval"
)

func TestHybridSearchRepository_SearchVectorMergesDocumentAndKnowledgeCandidates(t *testing.T) {
	repo := NewHybridSearchRepository(
		fakeSearchRepository{
			vector: []retrieval.Candidate{
				{ChunkID: "doc-chunk-1", DocumentID: "doc-1", SourceName: "doc.md", SourceKind: "document", VectorScore: 0.81},
			},
		},
		fakeSearchRepository{
			vector: []retrieval.Candidate{
				{ChunkID: "knowledge-chunk-1", KnowledgeEntryID: "knowledge-1", SourceName: "knowledge:knowledge-1", SourceKind: "knowledge", VectorScore: 0.92},
			},
		},
	)

	candidates, err := repo.SearchVector(context.Background(), "demo-user", []float32{0.1, 0.2}, 5)
	if err != nil {
		t.Fatalf("SearchVector() error = %v", err)
	}

	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(candidates))
	}
	if candidates[0].SourceKind != "knowledge" {
		t.Fatalf("expected knowledge candidate to rank first, got %s", candidates[0].SourceKind)
	}
}

func TestHybridSearchRepository_SearchKeywordMergesDocumentAndKnowledgeCandidates(t *testing.T) {
	repo := NewHybridSearchRepository(
		fakeSearchRepository{
			keyword: []retrieval.Candidate{
				{ChunkID: "doc-chunk-1", DocumentID: "doc-1", SourceName: "doc.md", SourceKind: "document", KeywordScore: 0.76},
			},
		},
		fakeSearchRepository{
			keyword: []retrieval.Candidate{
				{ChunkID: "knowledge-chunk-1", KnowledgeEntryID: "knowledge-1", SourceName: "knowledge:knowledge-1", SourceKind: "knowledge", KeywordScore: 0.88},
			},
		},
	)

	candidates, err := repo.SearchKeyword(context.Background(), "demo-user", "GMP", []string{"GMP"}, 5)
	if err != nil {
		t.Fatalf("SearchKeyword() error = %v", err)
	}

	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(candidates))
	}
	if candidates[0].SourceKind != "knowledge" {
		t.Fatalf("expected knowledge candidate to rank first, got %s", candidates[0].SourceKind)
	}
}

type fakeSearchRepository struct {
	vector  []retrieval.Candidate
	keyword []retrieval.Candidate
	err     error
}

func (f fakeSearchRepository) SearchVector(_ context.Context, _ string, _ []float32, _ int) ([]retrieval.Candidate, error) {
	if f.err != nil {
		return nil, f.err
	}
	return append([]retrieval.Candidate(nil), f.vector...), nil
}

func (f fakeSearchRepository) SearchKeyword(_ context.Context, _ string, _ string, _ []string, _ int) ([]retrieval.Candidate, error) {
	if f.err != nil {
		return nil, f.err
	}
	return append([]retrieval.Candidate(nil), f.keyword...), nil
}
