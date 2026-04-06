package knowledge

import (
	"context"
	"fmt"
	"time"
	"unicode/utf8"

	knowledgedomain "knowflow/internal/domain/knowledge"
	"knowflow/internal/platform/llm"
	"knowflow/internal/service/ingestion"
)

type EntryStore interface {
	GetByID(ctx context.Context, entryID string) (knowledgedomain.Entry, error)
	UpdateStatus(ctx context.Context, entryID, status string, updatedAt time.Time) error
}

type ChunkStore interface {
	ReplaceChunks(ctx context.Context, entryID string, chunks []knowledgedomain.Chunk) error
}

type Config struct {
	ChunkSize    int
	ChunkOverlap int
	Now          func() time.Time
	NewChunkID   func(entryID string, index int) string
}

type IndexResult struct {
	EntryID    string `json:"entry_id"`
	ChunkCount int    `json:"chunk_count"`
	Status     string `json:"status"`
}

type Service struct {
	entries    EntryStore
	chunks     ChunkStore
	embedder   llm.Embedder
	splitter   ingestion.Splitter
	now        func() time.Time
	newChunkID func(entryID string, index int) string
}

func NewService(entries EntryStore, chunks ChunkStore, embedder llm.Embedder, cfg Config) *Service {
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	return &Service{
		entries:    entries,
		chunks:     chunks,
		embedder:   embedder,
		splitter:   ingestion.NewSplitter(cfg.ChunkSize, cfg.ChunkOverlap),
		now:        now,
		newChunkID: cfg.NewChunkID,
	}
}

func (s *Service) IndexEntry(ctx context.Context, entry knowledgedomain.Entry) (IndexResult, error) {
	if entry.ID == "" {
		return IndexResult{}, fmt.Errorf("knowledge entry id is required")
	}
	if entry.UserID == "" {
		return IndexResult{}, fmt.Errorf("knowledge entry user id is required")
	}
	if entry.Content == "" {
		return IndexResult{}, fmt.Errorf("knowledge entry content is required")
	}

	chunks := s.splitter.Split(entry.Content)
	embeddings, err := s.embedder.Embed(ctx, chunks)
	if err != nil {
		return IndexResult{}, err
	}

	now := s.now()
	records := make([]knowledgedomain.Chunk, 0, len(chunks))
	for index, chunk := range chunks {
		embedding := []float32(nil)
		if index < len(embeddings) {
			embedding = embeddings[index]
		}
		records = append(records, knowledgedomain.Chunk{
			ID:               newChunkID(entry.ID, index, s.newChunkID),
			KnowledgeEntryID: entry.ID,
			UserID:           entry.UserID,
			ChunkIndex:       index,
			Content:          chunk,
			Embedding:        embedding,
			TokenCount:       estimateTokens(chunk),
			CreatedAt:        now,
			UpdatedAt:        now,
		})
	}

	if err := s.chunks.ReplaceChunks(ctx, entry.ID, records); err != nil {
		return IndexResult{}, err
	}
	if err := s.entries.UpdateStatus(ctx, entry.ID, "indexed", now); err != nil {
		return IndexResult{}, err
	}

	return IndexResult{
		EntryID:    entry.ID,
		ChunkCount: len(records),
		Status:     "indexed",
	}, nil
}

func (s *Service) ReindexEntry(ctx context.Context, entryID string) (IndexResult, error) {
	entry, err := s.entries.GetByID(ctx, entryID)
	if err != nil {
		return IndexResult{}, err
	}
	return s.IndexEntry(ctx, entry)
}

func newChunkID(entryID string, index int, generator func(entryID string, index int) string) string {
	if generator != nil {
		return generator(entryID, index)
	}
	return fmt.Sprintf("%s-chunk-%d", entryID, index)
}

func estimateTokens(content string) int {
	return utf8.RuneCountInString(content) / 4
}
