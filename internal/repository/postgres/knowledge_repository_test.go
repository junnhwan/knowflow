package postgres

import (
	"context"
	"testing"
	"time"

	"knowflow/internal/domain/knowledge"
	knowledgeservice "knowflow/internal/service/knowledge"

	pgxmock "github.com/pashagolub/pgxmock/v4"
)

func TestKnowledgeRepository_GetByIDReturnsPersistedEntry(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer mock.Close()

	repo := NewKnowledgeRepository(mock)
	now := time.Unix(1700000000, 0)

	rows := pgxmock.NewRows([]string{
		"id", "user_id", "session_id", "source_message_id", "document_id", "source_type", "title", "summary", "content", "keywords", "status", "review_status", "quality_score", "dedupe_hash", "merged_into_id", "disabled_at", "created_at", "updated_at",
	}).AddRow(
		"knowledge-1",
		"demo-user",
		"session-1",
		"msg-1",
		"doc-1",
		"qa",
		"GMP 调度核心结论",
		"Go 运行时采用 GMP 协作调度。",
		"Go 面试里经常会追问 GMP 调度模型。",
		`["gmp","scheduler"]`,
		"indexed",
		"draft",
		0.91,
		"hash-gmp",
		"",
		nil,
		now,
		now,
	)

	mock.ExpectQuery("SELECT id, user_id, session_id, source_message_id, document_id, source_type, title, summary, content, keywords, status, review_status, quality_score, dedupe_hash, merged_into_id, disabled_at, created_at, updated_at FROM knowledge_entries WHERE id = \\$1").
		WithArgs("knowledge-1").
		WillReturnRows(rows)

	entry, err := repo.GetByID(context.Background(), "knowledge-1")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if entry.SourceType != "qa" {
		t.Fatalf("expected source type qa, got %s", entry.SourceType)
	}
	if entry.Title != "GMP 调度核心结论" {
		t.Fatalf("expected title to be hydrated, got %s", entry.Title)
	}
	if entry.ReviewStatus != "draft" {
		t.Fatalf("expected review status draft, got %s", entry.ReviewStatus)
	}
	if len(entry.Keywords) != 2 {
		t.Fatalf("expected keywords to be hydrated, got %#v", entry.Keywords)
	}
}

func TestKnowledgeRepository_ReplaceChunksAndSearchKnowledgeChunks(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer mock.Close()

	repo := NewKnowledgeRepository(mock)
	now := time.Unix(1700000000, 0)
	chunks := []knowledge.Chunk{
		{
			ID:               "knowledge-1-chunk-0",
			KnowledgeEntryID: "knowledge-1",
			UserID:           "demo-user",
			ChunkIndex:       0,
			Content:          "GMP 中 P 负责承载可运行的 G，M 需要绑定 P 才能执行 Go 代码。",
			Embedding:        []float32{0.1, 0.2, 0.3},
			TokenCount:       16,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
	}

	mock.ExpectExec("DELETE FROM knowledge_chunks WHERE knowledge_entry_id = \\$1").
		WithArgs("knowledge-1").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	mock.ExpectExec("INSERT INTO knowledge_chunks").
		WithArgs(
			chunks[0].ID,
			chunks[0].KnowledgeEntryID,
			chunks[0].UserID,
			chunks[0].ChunkIndex,
			chunks[0].Content,
			pgxmock.AnyArg(),
			chunks[0].TokenCount,
			pgxmock.AnyArg(),
			pgxmock.AnyArg(),
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	if err := repo.ReplaceChunks(context.Background(), "knowledge-1", chunks); err != nil {
		t.Fatalf("ReplaceChunks() error = %v", err)
	}

	vectorRows := pgxmock.NewRows([]string{
		"id", "knowledge_entry_id", "source_name", "content", "score", "source_kind",
	}).AddRow(
		"knowledge-1-chunk-0",
		"knowledge-1",
		"knowledge:knowledge-1",
		chunks[0].Content,
		0.93,
		"knowledge",
	)
	mock.ExpectQuery("SELECT kc.id, kc.knowledge_entry_id, CONCAT\\('knowledge:', kc.knowledge_entry_id\\) AS source_name, kc.content, 1 - \\(kc.embedding <=> \\$2::vector\\) AS score, 'knowledge' AS source_kind").
		WithArgs("demo-user", pgxmock.AnyArg(), 3).
		WillReturnRows(vectorRows)

	vectorCandidates, err := repo.SearchVector(context.Background(), "demo-user", []float32{0.1, 0.2, 0.3}, 3)
	if err != nil {
		t.Fatalf("SearchVector() error = %v", err)
	}
	if len(vectorCandidates) != 1 {
		t.Fatalf("expected 1 vector candidate, got %d", len(vectorCandidates))
	}
	if vectorCandidates[0].SourceKind != "knowledge" {
		t.Fatalf("expected knowledge source kind, got %s", vectorCandidates[0].SourceKind)
	}

	keywordRows := pgxmock.NewRows([]string{
		"id", "knowledge_entry_id", "source_name", "content", "score", "source_kind",
	}).AddRow(
		"knowledge-1-chunk-0",
		"knowledge-1",
		"knowledge:knowledge-1",
		chunks[0].Content,
		0.88,
		"knowledge",
	)
	mock.ExpectQuery("SELECT kc.id, kc.knowledge_entry_id, CONCAT\\('knowledge:', kc.knowledge_entry_id\\) AS source_name, kc.content, similarity\\(kc.content, \\$2\\) AS score, 'knowledge' AS source_kind").
		WithArgs("demo-user", "GMP 调度", 3).
		WillReturnRows(keywordRows)

	keywordCandidates, err := repo.SearchKeyword(context.Background(), "demo-user", "GMP 调度", []string{"GMP", "调度"}, 3)
	if err != nil {
		t.Fatalf("SearchKeyword() error = %v", err)
	}
	if len(keywordCandidates) != 1 {
		t.Fatalf("expected 1 keyword candidate, got %d", len(keywordCandidates))
	}
	if keywordCandidates[0].KnowledgeEntryID != "knowledge-1" {
		t.Fatalf("expected knowledge entry id knowledge-1, got %s", keywordCandidates[0].KnowledgeEntryID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestKnowledgeRepository_UpdateStatusAndListPendingForReindex(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer mock.Close()

	repo := NewKnowledgeRepository(mock)
	now := time.Unix(1700000000, 0)

	mock.ExpectExec("UPDATE knowledge_entries SET status = \\$2, updated_at = \\$3 WHERE id = \\$1").
		WithArgs("knowledge-1", "index_failed", now).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	if err := repo.UpdateStatus(context.Background(), "knowledge-1", "index_failed", now); err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}

	rows := pgxmock.NewRows([]string{
		"id", "user_id", "session_id", "source_message_id", "document_id", "source_type", "title", "summary", "content", "keywords", "status", "review_status", "quality_score", "dedupe_hash", "merged_into_id", "disabled_at", "created_at", "updated_at",
	}).AddRow(
		"knowledge-1",
		"demo-user",
		"session-1",
		"msg-1",
		"doc-1",
		"qa",
		"GMP 调度",
		"运行时调度摘要",
		"GMP 中 P 负责承载可运行的 G。",
		`["gmp","runtime"]`,
		"index_failed",
		"draft",
		0.78,
		"hash-gmp",
		"",
		nil,
		now.Add(-time.Hour),
		now.Add(-time.Minute),
	)

	mock.ExpectQuery("SELECT id, user_id, session_id, source_message_id, document_id, source_type, title, summary, content, keywords, status, review_status, quality_score, dedupe_hash, merged_into_id, disabled_at, created_at, updated_at FROM knowledge_entries WHERE status IN \\('pending_index', 'index_failed'\\) AND updated_at <= \\$1 ORDER BY updated_at ASC LIMIT \\$2").
		WithArgs(now, 10).
		WillReturnRows(rows)

	entries, err := repo.ListPendingForReindex(context.Background(), now, 10)
	if err != nil {
		t.Fatalf("ListPendingForReindex() error = %v", err)
	}
	if len(entries) != 1 || entries[0].ID != "knowledge-1" {
		t.Fatalf("unexpected entries: %#v", entries)
	}
}

func TestKnowledgeRepository_UpdateDeleteChunksAndListByUser(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer mock.Close()

	repo := NewKnowledgeRepository(mock)
	now := time.Unix(1700000000, 0)
	disabledAt := now
	entry := knowledge.Entry{
		ID:           "knowledge-1",
		UserID:       "demo-user",
		SessionID:    "session-1",
		SourceType:   "manual",
		Title:        "Redis 双层记忆",
		Summary:      "最近窗口与历史摘要一起工作。",
		Content:      "Redis 双层记忆会保留最近多轮上下文，并在阈值触发时压缩更早历史。",
		Keywords:     []string{"redis", "memory"},
		Status:       "indexed",
		ReviewStatus: "active",
		QualityScore: 0.9,
		DedupeHash:   "hash-redis",
		DisabledAt:   &disabledAt,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	mock.ExpectExec("UPDATE knowledge_entries SET").
		WithArgs(
			entry.ID,
			entry.UserID,
			entry.SessionID,
			entry.SourceMessageID,
			entry.DocumentID,
			entry.SourceType,
			entry.Title,
			entry.Summary,
			entry.Content,
			pgxmock.AnyArg(),
			entry.Status,
			entry.ReviewStatus,
			entry.QualityScore,
			entry.DedupeHash,
			entry.MergedIntoID,
			entry.DisabledAt,
			entry.CreatedAt,
			entry.UpdatedAt,
		).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	if err := repo.Update(context.Background(), entry); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	mock.ExpectExec("DELETE FROM knowledge_chunks WHERE knowledge_entry_id = \\$1").
		WithArgs("knowledge-1").
		WillReturnResult(pgxmock.NewResult("DELETE", 2))

	if err := repo.DeleteChunks(context.Background(), "knowledge-1"); err != nil {
		t.Fatalf("DeleteChunks() error = %v", err)
	}

	rows := pgxmock.NewRows([]string{
		"id", "user_id", "session_id", "source_message_id", "document_id", "source_type", "title", "summary", "content", "keywords", "status", "review_status", "quality_score", "dedupe_hash", "merged_into_id", "disabled_at", "created_at", "updated_at",
	}).AddRow(
		entry.ID,
		entry.UserID,
		entry.SessionID,
		entry.SourceMessageID,
		entry.DocumentID,
		entry.SourceType,
		entry.Title,
		entry.Summary,
		entry.Content,
		`["redis","memory"]`,
		entry.Status,
		entry.ReviewStatus,
		entry.QualityScore,
		entry.DedupeHash,
		entry.MergedIntoID,
		entry.DisabledAt,
		entry.CreatedAt,
		entry.UpdatedAt,
	)

	mock.ExpectQuery("SELECT id, user_id, session_id, source_message_id, document_id, source_type, title, summary, content, keywords, status, review_status, quality_score, dedupe_hash, merged_into_id, disabled_at, created_at, updated_at FROM knowledge_entries WHERE user_id = \\$1").
		WithArgs("demo-user").
		WillReturnRows(rows)

	entries, err := repo.ListByUser(context.Background(), "demo-user", knowledgeservice.ListFilter{})
	if err != nil {
		t.Fatalf("ListByUser() error = %v", err)
	}
	if len(entries) != 1 || entries[0].Title != "Redis 双层记忆" {
		t.Fatalf("unexpected entries: %#v", entries)
	}
}
