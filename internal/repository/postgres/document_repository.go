package postgres

import (
	"context"
	"fmt"
	"strings"

	"knowflow/internal/domain/document"
	pgplatform "knowflow/internal/platform/postgres"
	"knowflow/internal/service/retrieval"
)

type DocumentRepository struct {
	db pgplatform.DB
}

func NewDocumentRepository(db pgplatform.DB) *DocumentRepository {
	return &DocumentRepository{db: db}
}

func (r *DocumentRepository) Create(ctx context.Context, doc document.Document) error {
	_, err := r.db.Exec(ctx, `
INSERT INTO documents (id, user_id, source_name, status, content_hash, raw_content, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
`, doc.ID, doc.UserID, doc.SourceName, doc.Status, doc.ContentHash, doc.RawContent, doc.CreatedAt, doc.UpdatedAt)
	return err
}

func (r *DocumentRepository) UpsertDocument(ctx context.Context, doc document.Document) (document.Document, error) {
	row := r.db.QueryRow(ctx, `
INSERT INTO documents (id, user_id, source_name, status, content_hash, raw_content, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (user_id, source_name)
DO UPDATE SET
  status = EXCLUDED.status,
  content_hash = EXCLUDED.content_hash,
  raw_content = EXCLUDED.raw_content,
  updated_at = EXCLUDED.updated_at
RETURNING id, user_id, source_name, status, content_hash, raw_content, created_at, updated_at
`, doc.ID, doc.UserID, doc.SourceName, doc.Status, doc.ContentHash, doc.RawContent, doc.CreatedAt, doc.UpdatedAt)

	var saved document.Document
	if err := row.Scan(
		&saved.ID,
		&saved.UserID,
		&saved.SourceName,
		&saved.Status,
		&saved.ContentHash,
		&saved.RawContent,
		&saved.CreatedAt,
		&saved.UpdatedAt,
	); err != nil {
		return document.Document{}, err
	}
	return saved, nil
}

func (r *DocumentRepository) GetByID(ctx context.Context, documentID string) (document.Document, error) {
	row := r.db.QueryRow(ctx, `
SELECT id, user_id, source_name, status, content_hash, raw_content, created_at, updated_at
FROM documents
WHERE id = $1
`, documentID)

	var doc document.Document
	if err := row.Scan(&doc.ID, &doc.UserID, &doc.SourceName, &doc.Status, &doc.ContentHash, &doc.RawContent, &doc.CreatedAt, &doc.UpdatedAt); err != nil {
		return document.Document{}, err
	}
	return doc, nil
}

func (r *DocumentRepository) DeleteByID(ctx context.Context, documentID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM documents WHERE id = $1`, documentID)
	return err
}

func (r *DocumentRepository) ReplaceChunks(ctx context.Context, documentID string, chunks []document.Chunk) error {
	if _, err := r.db.Exec(ctx, `DELETE FROM document_chunks WHERE document_id = $1`, documentID); err != nil {
		return err
	}

	for _, chunk := range chunks {
		_, err := r.db.Exec(ctx, `
INSERT INTO document_chunks (id, document_id, user_id, source_name, chunk_index, content, embedding, token_count, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7::vector, $8, $9, $10)
`, chunk.ID, chunk.DocumentID, chunk.UserID, chunk.SourceName, chunk.ChunkIndex, chunk.Content, vectorLiteral(chunk.Embedding), chunk.TokenCount, chunk.CreatedAt, chunk.UpdatedAt)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *DocumentRepository) SearchVector(ctx context.Context, userID string, embedding []float32, limit int) ([]retrieval.Candidate, error) {
	rows, err := r.db.Query(ctx, `
SELECT id, document_id, source_name, content, 1 - (embedding <=> $2::vector) AS score
FROM document_chunks
WHERE user_id = $1
ORDER BY embedding <=> $2::vector
LIMIT $3
`, userID, vectorLiteral(embedding), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var candidates []retrieval.Candidate
	for rows.Next() {
		var candidate retrieval.Candidate
		if err := rows.Scan(&candidate.ChunkID, &candidate.DocumentID, &candidate.SourceName, &candidate.Content, &candidate.VectorScore); err != nil {
			return nil, err
		}
		candidates = append(candidates, candidate)
	}
	return candidates, rows.Err()
}

func (r *DocumentRepository) SearchKeyword(ctx context.Context, userID string, query string, _ []string, limit int) ([]retrieval.Candidate, error) {
	rows, err := r.db.Query(ctx, `
SELECT id, document_id, source_name, content, similarity(content, $2) AS score
FROM document_chunks
WHERE user_id = $1
  AND content % $2
ORDER BY similarity(content, $2) DESC
LIMIT $3
`, userID, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var candidates []retrieval.Candidate
	for rows.Next() {
		var candidate retrieval.Candidate
		if err := rows.Scan(&candidate.ChunkID, &candidate.DocumentID, &candidate.SourceName, &candidate.Content, &candidate.KeywordScore); err != nil {
			return nil, err
		}
		candidates = append(candidates, candidate)
	}
	return candidates, rows.Err()
}

func vectorLiteral(values []float32) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, fmt.Sprintf("%f", value))
	}
	return "[" + strings.Join(parts, ",") + "]"
}
