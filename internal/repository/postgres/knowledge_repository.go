package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	knowledgedomain "knowflow/internal/domain/knowledge"
	pgplatform "knowflow/internal/platform/postgres"
	knowledgeservice "knowflow/internal/service/knowledge"
	"knowflow/internal/service/retrieval"
)

type KnowledgeRepository struct {
	db pgplatform.DB
}

func NewKnowledgeRepository(db pgplatform.DB) *KnowledgeRepository {
	return &KnowledgeRepository{db: db}
}

func (r *KnowledgeRepository) Create(ctx context.Context, entry knowledgedomain.Entry) error {
	_, err := r.db.Exec(ctx, `
INSERT INTO knowledge_entries (
    id, user_id, session_id, source_message_id, document_id, source_type,
    title, summary, content, keywords, status, review_status, quality_score,
    dedupe_hash, merged_into_id, disabled_at, created_at, updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb, $11, $12, $13, $14, $15, $16, $17, $18)
`, entry.ID, entry.UserID, entry.SessionID, entry.SourceMessageID, entry.DocumentID, entry.SourceType, entry.Title, entry.Summary, entry.Content, keywordsJSON(entry.Keywords), entry.Status, entry.ReviewStatus, entry.QualityScore, entry.DedupeHash, entry.MergedIntoID, entry.DisabledAt, entry.CreatedAt, entry.UpdatedAt)
	return err
}

func (r *KnowledgeRepository) GetByID(ctx context.Context, entryID string) (knowledgedomain.Entry, error) {
	row := r.db.QueryRow(ctx, `
SELECT id, user_id, session_id, source_message_id, document_id, source_type, title, summary, content, keywords, status, review_status, quality_score, dedupe_hash, merged_into_id, disabled_at, created_at, updated_at
FROM knowledge_entries
WHERE id = $1
`, entryID)

	entry, err := scanKnowledgeEntry(row)
	if err != nil {
		return knowledgedomain.Entry{}, err
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

func (r *KnowledgeRepository) Update(ctx context.Context, entry knowledgedomain.Entry) error {
	_, err := r.db.Exec(ctx, `
UPDATE knowledge_entries SET
    user_id = $2,
    session_id = $3,
    source_message_id = $4,
    document_id = $5,
    source_type = $6,
    title = $7,
    summary = $8,
    content = $9,
    keywords = $10::jsonb,
    status = $11,
    review_status = $12,
    quality_score = $13,
    dedupe_hash = $14,
    merged_into_id = $15,
    disabled_at = $16,
    created_at = $17,
    updated_at = $18
WHERE id = $1
`, entry.ID, entry.UserID, entry.SessionID, entry.SourceMessageID, entry.DocumentID, entry.SourceType, entry.Title, entry.Summary, entry.Content, keywordsJSON(entry.Keywords), entry.Status, entry.ReviewStatus, entry.QualityScore, entry.DedupeHash, entry.MergedIntoID, entry.DisabledAt, entry.CreatedAt, entry.UpdatedAt)
	return err
}

func (r *KnowledgeRepository) ListByUser(ctx context.Context, userID string, filter knowledgeservice.ListFilter) ([]knowledgedomain.Entry, error) {
	query := `
SELECT id, user_id, session_id, source_message_id, document_id, source_type, title, summary, content, keywords, status, review_status, quality_score, dedupe_hash, merged_into_id, disabled_at, created_at, updated_at
FROM knowledge_entries
WHERE user_id = $1
`
	args := []any{userID}
	if strings.TrimSpace(filter.ReviewStatus) != "" {
		query += " AND review_status = $2"
		args = append(args, filter.ReviewStatus)
	}
	query += " ORDER BY updated_at DESC"
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []knowledgedomain.Entry
	for rows.Next() {
		entry, err := scanKnowledgeEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func (r *KnowledgeRepository) ListPendingForReindex(ctx context.Context, before time.Time, limit int) ([]knowledgedomain.Entry, error) {
	rows, err := r.db.Query(ctx, `
SELECT id, user_id, session_id, source_message_id, document_id, source_type, title, summary, content, keywords, status, review_status, quality_score, dedupe_hash, merged_into_id, disabled_at, created_at, updated_at
FROM knowledge_entries
WHERE status IN ('pending_index', 'index_failed')
  AND updated_at <= $1
ORDER BY updated_at ASC
LIMIT $2
`, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []knowledgedomain.Entry
	for rows.Next() {
		entry, err := scanKnowledgeEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func (r *KnowledgeRepository) DeleteChunks(ctx context.Context, knowledgeEntryID string) error {
	if _, err := r.db.Exec(ctx, `DELETE FROM knowledge_chunks WHERE knowledge_entry_id = $1`, knowledgeEntryID); err != nil {
		return err
	}
	return nil
}

func (r *KnowledgeRepository) ReplaceChunks(ctx context.Context, knowledgeEntryID string, chunks []knowledgedomain.Chunk) error {
	if err := r.DeleteChunks(ctx, knowledgeEntryID); err != nil {
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
JOIN knowledge_entries ke ON ke.id = kc.knowledge_entry_id
WHERE kc.user_id = $1
  AND ke.review_status NOT IN ('disabled', 'merged')
  AND ke.disabled_at IS NULL
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
JOIN knowledge_entries ke ON ke.id = kc.knowledge_entry_id
WHERE kc.user_id = $1
  AND ke.review_status NOT IN ('disabled', 'merged')
  AND ke.disabled_at IS NULL
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

type knowledgeEntryScanner interface {
	Scan(dest ...any) error
}

func scanKnowledgeEntry(scanner knowledgeEntryScanner) (knowledgedomain.Entry, error) {
	var (
		entry       knowledgedomain.Entry
		keywordsRaw []byte
	)
	if err := scanner.Scan(
		&entry.ID,
		&entry.UserID,
		&entry.SessionID,
		&entry.SourceMessageID,
		&entry.DocumentID,
		&entry.SourceType,
		&entry.Title,
		&entry.Summary,
		&entry.Content,
		&keywordsRaw,
		&entry.Status,
		&entry.ReviewStatus,
		&entry.QualityScore,
		&entry.DedupeHash,
		&entry.MergedIntoID,
		&entry.DisabledAt,
		&entry.CreatedAt,
		&entry.UpdatedAt,
	); err != nil {
		return knowledgedomain.Entry{}, err
	}
	if len(keywordsRaw) > 0 {
		if err := json.Unmarshal(keywordsRaw, &entry.Keywords); err != nil {
			return knowledgedomain.Entry{}, err
		}
	}
	return entry, nil
}

func keywordsJSON(keywords []string) string {
	payload, err := json.Marshal(knowledgeservice.NormalizeKeywords(keywords))
	if err != nil {
		return "[]"
	}
	return string(payload)
}
