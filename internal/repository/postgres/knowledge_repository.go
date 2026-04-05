package postgres

import (
	"context"

	"knowflow/internal/domain/knowledge"
	pgplatform "knowflow/internal/platform/postgres"
)

type KnowledgeRepository struct {
	db pgplatform.DB
}

func NewKnowledgeRepository(db pgplatform.DB) *KnowledgeRepository {
	return &KnowledgeRepository{db: db}
}

func (r *KnowledgeRepository) Create(ctx context.Context, entry knowledge.Entry) error {
	_, err := r.db.Exec(ctx, `
INSERT INTO knowledge_entries (id, user_id, session_id, source_message_id, document_id, source_type, content, status, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
`, entry.ID, entry.UserID, entry.SessionID, entry.SourceMessageID, entry.DocumentID, entry.SourceType, entry.Content, entry.Status, entry.CreatedAt, entry.UpdatedAt)
	return err
}
