package postgres

import (
	"context"
	"time"

	"knowflow/internal/domain/knowledge"
	pgplatform "knowflow/internal/platform/postgres"
	"knowflow/internal/service/retrieval"
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

func (r *KnowledgeRepository) GetByID(ctx context.Context, entryID string) (knowledge.Entry, error) {
	row := r.db.QueryRow(ctx, `
SELECT id, user_id, session_id, source_message_id, document_id, source_type, content, status, created_at, updated_at
FROM knowledge_entries
WHERE id = $1
`, entryID)

	var entry knowledge.Entry
	if err := row.Scan(
		&entry.ID,
		&entry.UserID,
		&entry.SessionID,
		&entry.SourceMessageID,
		&entry.DocumentID,
		&entry.SourceType,
		&entry.Content,
		&entry.Status,
		&entry.CreatedAt,
		&entry.UpdatedAt,
	); err != nil {
		return knowledge.Entry{}, err
	}
	return entry, nil
}

func (r *KnowledgeRepository) UpdateStatus(ctx context.Context, entryID, status string, updatedAt time.Time) error {
	_, err := r.db.Exec(ctx, `
UPDATE knowledge_entries
SET status = $2, updated_at = $3
WHERE id = $1
`, entryID, status, updatedAt)
	return err
}

func (r *KnowledgeRepository) ReplaceChunks(ctx context.Context, knowledgeEntryID string, chunks []knowledge.Chunk) error {
	if _, err := r.db.Exec(ctx, `DELETE FROM knowledge_chunks WHERE knowledge_entry_id = $1`, knowledgeEntryID); err != nil {
		return err
	}

	for _, chunk := range chunks {
		_, err := r.db.Exec(ctx, `
INSERT INTO knowledge_chunks (id, knowledge_entry_id, user_id, chunk_index, content, embedding, token_count, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6::vector, $7, $8, $9)
`, chunk.ID, chunk.KnowledgeEntryID, chunk.UserID, chunk.ChunkIndex, chunk.Content, vectorLiteral(chunk.Embedding), chunk.TokenCount, chunk.CreatedAt, chunk.UpdatedAt)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *KnowledgeRepository) SearchVector(ctx context.Context, userID string, embedding []float32, limit int) ([]retrieval.Candidate, error) {
	rows, err := r.db.Query(ctx, `
SELECT kc.id, kc.knowledge_entry_id, CONCAT('knowledge:', kc.knowledge_entry_id) AS source_name, kc.content, 1 - (kc.embedding <=> $2::vector) AS score, 'knowledge' AS source_kind
FROM knowledge_chunks kc
WHERE kc.user_id = $1
ORDER BY kc.embedding <=> $2::vector
LIMIT $3
`, userID, vectorLiteral(embedding), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var candidates []retrieval.Candidate
	for rows.Next() {
		var candidate retrieval.Candidate
		if err := rows.Scan(&candidate.ChunkID, &candidate.KnowledgeEntryID, &candidate.SourceName, &candidate.Content, &candidate.VectorScore, &candidate.SourceKind); err != nil {
			return nil, err
		}
		candidates = append(candidates, candidate)
	}
	return candidates, rows.Err()
}

func (r *KnowledgeRepository) SearchKeyword(ctx context.Context, userID string, query string, _ []string, limit int) ([]retrieval.Candidate, error) {
	rows, err := r.db.Query(ctx, `
SELECT kc.id, kc.knowledge_entry_id, CONCAT('knowledge:', kc.knowledge_entry_id) AS source_name, kc.content, similarity(kc.content, $2) AS score, 'knowledge' AS source_kind
FROM knowledge_chunks kc
WHERE kc.user_id = $1
  AND kc.content % $2
ORDER BY similarity(kc.content, $2) DESC
LIMIT $3
`, userID, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var candidates []retrieval.Candidate
	for rows.Next() {
		var candidate retrieval.Candidate
		if err := rows.Scan(&candidate.ChunkID, &candidate.KnowledgeEntryID, &candidate.SourceName, &candidate.Content, &candidate.KeywordScore, &candidate.SourceKind); err != nil {
			return nil, err
		}
		candidates = append(candidates, candidate)
	}
	return candidates, rows.Err()
}
