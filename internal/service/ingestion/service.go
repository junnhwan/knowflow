package ingestion

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
	"unicode/utf8"

	"knowflow/internal/domain/document"
	"knowflow/internal/platform/llm"
)

type DocumentStore interface {
	UpsertDocument(ctx context.Context, doc document.Document) (document.Document, error)
	ReplaceChunks(ctx context.Context, documentID string, chunks []document.Chunk) error
	UpdateStatus(ctx context.Context, documentID, status string, updatedAt time.Time) error
}

type ServiceConfig struct {
	ChunkSize    int
	ChunkOverlap int
	Now          func() time.Time
	NewID        func() string
	NewChunkID   func(index int) string
}

type IngestRequest struct {
	UserID     string
	SourceName string
	Content    string
}

type IngestResult struct {
	DocumentID string
	ChunkCount int
	Status     string
}

type Service struct {
	store    DocumentStore
	embedder llm.Embedder
	parser   Parser
	splitter Splitter
	config   ServiceConfig
}

func NewService(store DocumentStore, embedder llm.Embedder, cfg ServiceConfig) *Service {
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	return &Service{
		store:    store,
		embedder: embedder,
		parser:   NewParser(),
		splitter: NewSplitter(cfg.ChunkSize, cfg.ChunkOverlap),
		config: ServiceConfig{
			ChunkSize:    cfg.ChunkSize,
			ChunkOverlap: cfg.ChunkOverlap,
			Now:          now,
			NewID:        cfg.NewID,
			NewChunkID:   cfg.NewChunkID,
		},
	}
}

func (s *Service) Ingest(ctx context.Context, req IngestRequest) (IngestResult, error) {
	if req.UserID == "" {
		return IngestResult{}, fmt.Errorf("user id is required")
	}
	if req.SourceName == "" {
		return IngestResult{}, fmt.Errorf("source name is required")
	}

	normalized, err := s.parser.Parse(req.SourceName, req.Content)
	if err != nil {
		return IngestResult{}, err
	}

	now := s.config.Now()
	docID := newDocumentID(s.config.NewID)
	doc := document.Document{
		ID:          docID,
		UserID:      req.UserID,
		SourceName:  req.SourceName,
		Status:      "pending_index",
		ContentHash: hashContent(normalized),
		RawContent:  normalized,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	saved, err := s.store.UpsertDocument(ctx, doc)
	if err != nil {
		return IngestResult{}, err
	}

	chunks := s.splitter.Split(normalized)
	embeddings, err := s.embedder.Embed(ctx, chunks)
	if err != nil {
		_ = s.store.UpdateStatus(ctx, saved.ID, "index_failed", now)
		return IngestResult{}, err
	}

	records := make([]document.Chunk, 0, len(chunks))
	for index, chunk := range chunks {
		embedding := []float32(nil)
		if index < len(embeddings) {
			embedding = embeddings[index]
		}
		records = append(records, document.Chunk{
			ID:         newChunkID(saved.ID, index, s.config.NewChunkID),
			DocumentID: saved.ID,
			UserID:     req.UserID,
			SourceName: req.SourceName,
			ChunkIndex: index,
			Content:    chunk,
			Embedding:  embedding,
			TokenCount: estimateTokens(chunk),
			CreatedAt:  now,
			UpdatedAt:  now,
		})
	}

	if err := s.store.ReplaceChunks(ctx, saved.ID, records); err != nil {
		_ = s.store.UpdateStatus(ctx, saved.ID, "index_failed", now)
		return IngestResult{}, err
	}
	if err := s.store.UpdateStatus(ctx, saved.ID, "indexed", now); err != nil {
		return IngestResult{}, err
	}

	return IngestResult{
		DocumentID: saved.ID,
		ChunkCount: len(records),
		Status:     "indexed",
	}, nil
}

func newDocumentID(generator func() string) string {
	if generator != nil {
		return generator()
	}
	return fmt.Sprintf("doc-%d", time.Now().UnixNano())
}

func newChunkID(documentID string, index int, generator func(int) string) string {
	if generator != nil {
		return generator(index)
	}
	return fmt.Sprintf("%s-chunk-%d", documentID, index)
}

func hashContent(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func estimateTokens(content string) int {
	return utf8.RuneCountInString(content) / 4
}
