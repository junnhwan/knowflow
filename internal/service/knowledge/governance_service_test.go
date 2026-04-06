package knowledge

import (
	"context"
	"testing"
	"time"

	knowledgedomain "knowflow/internal/domain/knowledge"
	"knowflow/internal/service/retrieval"
)

func TestGovernanceService_UpdateEntryReindexesAndRefreshesDerivedFields(t *testing.T) {
	repo := &fakeGovernanceRepository{
		entries: map[string]knowledgedomain.Entry{
			"knowledge-1": {
				ID:           "knowledge-1",
				UserID:       "demo-user",
				Title:        "旧标题",
				Summary:      "旧摘要",
				Content:      "旧内容",
				ReviewStatus: "draft",
				Status:       "indexed",
			},
		},
	}
	indexer := &fakeGovernanceIndexer{}
	service := NewGovernanceService(repo, repo, repo, indexer, fakeGovernanceEmbedder{}, GovernanceConfig{
		Now: func() time.Time { return time.Unix(1700000000, 0) },
	})

	result, err := service.UpdateEntry(context.Background(), UpdateEntryRequest{
		UserID:       "demo-user",
		KnowledgeID:  "knowledge-1",
		Title:        "Redis 双层记忆",
		Summary:      "最近窗口与历史摘要一起工作。",
		Content:      "Redis 双层记忆会保留最近多轮上下文，并在阈值触发后压缩更早历史。",
		Keywords:     []string{"redis", "memory", "summary"},
		ReviewStatus: "active",
	})
	if err != nil {
		t.Fatalf("UpdateEntry() error = %v", err)
	}

	if result.Entry.Title != "Redis 双层记忆" {
		t.Fatalf("unexpected title: %s", result.Entry.Title)
	}
	if result.Entry.ReviewStatus != "active" {
		t.Fatalf("unexpected review status: %s", result.Entry.ReviewStatus)
	}
	if result.Entry.DedupeHash == "" {
		t.Fatal("expected dedupe hash to be refreshed")
	}
	if result.Entry.QualityScore <= 0 {
		t.Fatalf("expected quality score to be recalculated, got %v", result.Entry.QualityScore)
	}
	if !indexer.called {
		t.Fatal("expected update to trigger reindex")
	}
}

func TestGovernanceService_DisableEntryRemovesChunks(t *testing.T) {
	repo := &fakeGovernanceRepository{
		entries: map[string]knowledgedomain.Entry{
			"knowledge-1": {
				ID:           "knowledge-1",
				UserID:       "demo-user",
				Title:        "可被禁用的知识",
				ReviewStatus: "active",
				Status:       "indexed",
			},
		},
	}
	service := NewGovernanceService(repo, repo, repo, &fakeGovernanceIndexer{}, fakeGovernanceEmbedder{}, GovernanceConfig{
		Now: func() time.Time { return time.Unix(1700000000, 0) },
	})

	result, err := service.DisableEntry(context.Background(), "demo-user", "knowledge-1")
	if err != nil {
		t.Fatalf("DisableEntry() error = %v", err)
	}

	if result.ReviewStatus != "disabled" {
		t.Fatalf("expected disabled review status, got %s", result.ReviewStatus)
	}
	if result.DisabledAt == nil {
		t.Fatal("expected disabled_at to be set")
	}
	if len(repo.deletedChunkIDs) != 1 || repo.deletedChunkIDs[0] != "knowledge-1" {
		t.Fatalf("expected chunks to be deleted, got %#v", repo.deletedChunkIDs)
	}
}

func TestGovernanceService_MergeEntriesKeepsTargetSearchableAndMarksSourceMerged(t *testing.T) {
	repo := &fakeGovernanceRepository{
		entries: map[string]knowledgedomain.Entry{
			"knowledge-source": {
				ID:           "knowledge-source",
				UserID:       "demo-user",
				Title:        "知识反写",
				Summary:      "把高价值问答沉淀成知识条目。",
				Content:      "知识反写会把高价值问答沉淀为结构化知识。",
				Keywords:     []string{"writeback", "knowledge"},
				ReviewStatus: "draft",
				Status:       "indexed",
			},
			"knowledge-target": {
				ID:           "knowledge-target",
				UserID:       "demo-user",
				Title:        "知识运营闭环",
				Summary:      "检索与沉淀形成闭环。",
				Content:      "知识运营闭环包括检索、回答、沉淀和热更新。",
				Keywords:     []string{"rag", "closure"},
				ReviewStatus: "active",
				Status:       "indexed",
			},
		},
	}
	indexer := &fakeGovernanceIndexer{}
	service := NewGovernanceService(repo, repo, repo, indexer, fakeGovernanceEmbedder{}, GovernanceConfig{
		Now: func() time.Time { return time.Unix(1700000000, 0) },
	})

	result, err := service.MergeEntries(context.Background(), MergeEntriesRequest{
		UserID:        "demo-user",
		SourceEntryID: "knowledge-source",
		TargetEntryID: "knowledge-target",
	})
	if err != nil {
		t.Fatalf("MergeEntries() error = %v", err)
	}

	if result.TargetEntry.ID != "knowledge-target" {
		t.Fatalf("unexpected target entry: %s", result.TargetEntry.ID)
	}
	if result.SourceEntry.ReviewStatus != "merged" {
		t.Fatalf("expected source to be marked merged, got %s", result.SourceEntry.ReviewStatus)
	}
	if result.SourceEntry.MergedIntoID != "knowledge-target" {
		t.Fatalf("unexpected merged_into_id: %s", result.SourceEntry.MergedIntoID)
	}
	if len(repo.deletedChunkIDs) == 0 || repo.deletedChunkIDs[0] != "knowledge-source" {
		t.Fatalf("expected source chunks to be deleted, got %#v", repo.deletedChunkIDs)
	}
	if !indexer.called {
		t.Fatal("expected target entry to be reindexed after merge")
	}
	if len(result.TargetEntry.Keywords) < 3 {
		t.Fatalf("expected merged keywords, got %#v", result.TargetEntry.Keywords)
	}
}

func TestGovernanceService_GetEntryReturnsDedupeCandidates(t *testing.T) {
	repo := &fakeGovernanceRepository{
		entries: map[string]knowledgedomain.Entry{
			"knowledge-1": {
				ID:           "knowledge-1",
				UserID:       "demo-user",
				Title:        "Redis 双层记忆",
				Content:      "Redis 双层记忆会保留最近窗口并压缩历史摘要。",
				Keywords:     []string{"redis", "memory", "summary"},
				DedupeHash:   "hash-1",
				ReviewStatus: "active",
				Status:       "indexed",
			},
			"knowledge-2": {
				ID:           "knowledge-2",
				UserID:       "demo-user",
				Title:        "Redis 记忆压缩",
				Content:      "记忆压缩会在消息过长时总结更早历史。",
				Keywords:     []string{"redis", "summary"},
				DedupeHash:   "hash-1",
				ReviewStatus: "draft",
				Status:       "indexed",
			},
		},
		vectorCandidates: []retrieval.Candidate{
			{KnowledgeEntryID: "knowledge-2", VectorScore: 0.91},
		},
	}
	service := NewGovernanceService(repo, repo, repo, &fakeGovernanceIndexer{}, fakeGovernanceEmbedder{}, GovernanceConfig{
		Now: func() time.Time { return time.Unix(1700000000, 0) },
	})

	result, err := service.GetEntry(context.Background(), "demo-user", "knowledge-1")
	if err != nil {
		t.Fatalf("GetEntry() error = %v", err)
	}

	if len(result.DedupeCandidates) != 1 {
		t.Fatalf("expected one dedupe candidate, got %#v", result.DedupeCandidates)
	}
	if result.DedupeCandidates[0].KnowledgeEntryID != "knowledge-2" {
		t.Fatalf("unexpected candidate id: %s", result.DedupeCandidates[0].KnowledgeEntryID)
	}
}

type fakeGovernanceRepository struct {
	entries          map[string]knowledgedomain.Entry
	updatedEntries   []knowledgedomain.Entry
	deletedChunkIDs  []string
	vectorCandidates []retrieval.Candidate
}

func (f *fakeGovernanceRepository) GetByID(_ context.Context, entryID string) (knowledgedomain.Entry, error) {
	entry, ok := f.entries[entryID]
	if !ok {
		return knowledgedomain.Entry{}, ErrEntryNotFound
	}
	return entry, nil
}

func (f *fakeGovernanceRepository) ListByUser(_ context.Context, userID string, _ ListFilter) ([]knowledgedomain.Entry, error) {
	out := make([]knowledgedomain.Entry, 0, len(f.entries))
	for _, entry := range f.entries {
		if entry.UserID == userID {
			out = append(out, entry)
		}
	}
	return out, nil
}

func (f *fakeGovernanceRepository) Update(_ context.Context, entry knowledgedomain.Entry) error {
	f.updatedEntries = append(f.updatedEntries, entry)
	f.entries[entry.ID] = entry
	return nil
}

func (f *fakeGovernanceRepository) DeleteChunks(_ context.Context, entryID string) error {
	f.deletedChunkIDs = append(f.deletedChunkIDs, entryID)
	return nil
}

func (f *fakeGovernanceRepository) SearchVector(_ context.Context, _ string, _ []float32, _ int) ([]retrieval.Candidate, error) {
	return f.vectorCandidates, nil
}

type fakeGovernanceIndexer struct {
	called bool
}

func (f *fakeGovernanceIndexer) ReindexEntry(_ context.Context, _ string) (IndexResult, error) {
	f.called = true
	return IndexResult{Status: "indexed"}, nil
}

type fakeGovernanceEmbedder struct{}

func (fakeGovernanceEmbedder) Embed(_ context.Context, _ []string) ([][]float32, error) {
	return [][]float32{{0.1, 0.2, 0.3}}, nil
}
