package reindexer

import (
	"context"
	"time"

	documentdomain "knowflow/internal/domain/document"
	knowledgedomain "knowflow/internal/domain/knowledge"
)

type DocumentSource interface {
	ListPendingForReindex(ctx context.Context, before time.Time, limit int) ([]documentdomain.Document, error)
	UpdateStatus(ctx context.Context, documentID, status string, updatedAt time.Time) error
}

type KnowledgeSource interface {
	ListPendingForReindex(ctx context.Context, before time.Time, limit int) ([]knowledgedomain.Entry, error)
	UpdateStatus(ctx context.Context, entryID, status string, updatedAt time.Time) error
}

type DocumentReindexer interface {
	ReindexDocument(ctx context.Context, documentID string) (map[string]any, error)
}

type KnowledgeReindexer interface {
	ReindexKnowledgeEntry(ctx context.Context, entryID string) (map[string]any, error)
}

type Config struct {
	RetryInterval time.Duration
	BatchSize     int
	Now           func() time.Time
	Observer      Observer
	Logger        Logger
}

type Observer interface {
	RecordReindexTask(targetType, result string)
}

type Logger interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
}

type Service struct {
	documents DocumentSource
	knowledge KnowledgeSource
	docRunner DocumentReindexer
	knRunner  KnowledgeReindexer
	config    Config
}

func NewService(documents DocumentSource, knowledge KnowledgeSource, docRunner DocumentReindexer, knRunner KnowledgeReindexer, cfg Config) *Service {
	if cfg.RetryInterval <= 0 {
		cfg.RetryInterval = 30 * time.Second
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 20
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &Service{
		documents: documents,
		knowledge: knowledge,
		docRunner: docRunner,
		knRunner:  knRunner,
		config:    cfg,
	}
}

func (s *Service) Run(ctx context.Context) {
	ticker := time.NewTicker(s.config.RetryInterval)
	defer ticker.Stop()

	for {
		if err := s.ProcessPending(ctx); err != nil {
			// best-effort worker: ignore cycle errors and continue on next tick
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Service) ProcessPending(ctx context.Context) error {
	before := s.config.Now().Add(-s.config.RetryInterval)

	documents, err := s.documents.ListPendingForReindex(ctx, before, s.config.BatchSize)
	if err != nil {
		return err
	}
	for _, doc := range documents {
		if _, err := s.docRunner.ReindexDocument(ctx, doc.ID); err != nil {
			if s.config.Observer != nil {
				s.config.Observer.RecordReindexTask("document", "failed")
			}
			if s.config.Logger != nil {
				s.config.Logger.Warn("background reindex failed",
					"target_type", "document",
					"target_id", doc.ID,
					"error", err.Error(),
				)
			}
			_ = s.documents.UpdateStatus(ctx, doc.ID, "index_failed", s.config.Now())
			continue
		}
		if s.config.Observer != nil {
			s.config.Observer.RecordReindexTask("document", "success")
		}
		if s.config.Logger != nil {
			s.config.Logger.Info("background reindex succeeded",
				"target_type", "document",
				"target_id", doc.ID,
			)
		}
	}

	entries, err := s.knowledge.ListPendingForReindex(ctx, before, s.config.BatchSize)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if _, err := s.knRunner.ReindexKnowledgeEntry(ctx, entry.ID); err != nil {
			if s.config.Observer != nil {
				s.config.Observer.RecordReindexTask("knowledge_entry", "failed")
			}
			if s.config.Logger != nil {
				s.config.Logger.Warn("background reindex failed",
					"target_type", "knowledge_entry",
					"target_id", entry.ID,
					"error", err.Error(),
				)
			}
			_ = s.knowledge.UpdateStatus(ctx, entry.ID, "index_failed", s.config.Now())
			continue
		}
		if s.config.Observer != nil {
			s.config.Observer.RecordReindexTask("knowledge_entry", "success")
		}
		if s.config.Logger != nil {
			s.config.Logger.Info("background reindex succeeded",
				"target_type", "knowledge_entry",
				"target_id", entry.ID,
			)
		}
	}

	return nil
}
