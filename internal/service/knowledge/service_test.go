package knowledge

import (
	"context"
	"fmt"
	"testing"
	"time"

	knowledgedomain "knowflow/internal/domain/knowledge"
)

func TestService_IndexEntryStoresChunksAndUpdatesStatus(t *testing.T) {
	now := time.Unix(1700000000, 0)
	entryStore := &fakeEntryStore{}
	chunkStore := &fakeChunkStore{}
	svc := NewService(entryStore, chunkStore, fakeEmbedder{}, Config{
		ChunkSize:    32,
		ChunkOverlap: 4,
		Now:          func() time.Time { return now },
		NewChunkID: func(entryID string, index int) string {
			return fmt.Sprintf("%s-chunk-%d", entryID, index)
		},
	})

	result, err := svc.IndexEntry(context.Background(), knowledgedomain.Entry{
		ID:        "knowledge-1",
		UserID:    "demo-user",
		Content:   "GMP 调度模型里，P 负责承载可运行的 G。M 需要绑定 P 才能执行 Go 代码。",
		Status:    "pending_index",
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("IndexEntry() error = %v", err)
	}

	if result.ChunkCount == 0 {
		t.Fatalf("expected indexed chunks")
	}
	if result.Status != "indexed" {
		t.Fatalf("expected indexed status, got %s", result.Status)
	}
	if entryStore.updatedStatus != "indexed" {
		t.Fatalf("expected status update to indexed, got %s", entryStore.updatedStatus)
	}
	if len(chunkStore.chunks) == 0 {
		t.Fatalf("expected persisted knowledge chunks")
	}
}

func TestService_ReindexEntryReloadsPersistedContent(t *testing.T) {
	now := time.Unix(1700000000, 0)
	entryStore := &fakeEntryStore{
		entry: knowledgedomain.Entry{
			ID:        "knowledge-2",
			UserID:    "demo-user",
			Content:   "channel 常见追问包括缓冲区语义、阻塞行为和关闭后的读取规则。",
			Status:    "indexed",
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
	chunkStore := &fakeChunkStore{}
	svc := NewService(entryStore, chunkStore, fakeEmbedder{}, Config{
		ChunkSize:    48,
		ChunkOverlap: 8,
		Now:          func() time.Time { return now },
		NewChunkID: func(entryID string, index int) string {
			return fmt.Sprintf("%s-chunk-%d", entryID, index)
		},
	})

	result, err := svc.ReindexEntry(context.Background(), "knowledge-2")
	if err != nil {
		t.Fatalf("ReindexEntry() error = %v", err)
	}

	if result.EntryID != "knowledge-2" {
		t.Fatalf("expected entry id knowledge-2, got %s", result.EntryID)
	}
	if len(chunkStore.chunks) == 0 {
		t.Fatalf("expected reindexed chunks")
	}
}

type fakeEntryStore struct {
	entry          knowledgedomain.Entry
	updatedEntryID string
	updatedStatus  string
}

func (f *fakeEntryStore) GetByID(_ context.Context, entryID string) (knowledgedomain.Entry, error) {
	if f.entry.ID == "" {
		return knowledgedomain.Entry{}, fmt.Errorf("entry not found: %s", entryID)
	}
	return f.entry, nil
}

func (f *fakeEntryStore) UpdateStatus(_ context.Context, entryID, status string, _ time.Time) error {
	f.updatedEntryID = entryID
	f.updatedStatus = status
	return nil
}

type fakeChunkStore struct {
	entryID string
	chunks  []knowledgedomain.Chunk
}

func (f *fakeChunkStore) ReplaceChunks(_ context.Context, entryID string, chunks []knowledgedomain.Chunk) error {
	f.entryID = entryID
	f.chunks = append([]knowledgedomain.Chunk(nil), chunks...)
	return nil
}

type fakeEmbedder struct{}

func (fakeEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, 0, len(texts))
	for range texts {
		out = append(out, []float32{1, 0, 1, 0})
	}
	return out, nil
}
