package postgres

import (
	"context"
	"testing"
	"time"

	"knowflow/internal/domain/document"

	pgxmock "github.com/pashagolub/pgxmock/v4"
)

func TestDocumentRepository_CreateDocument(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer mock.Close()

	repo := NewDocumentRepository(mock)
	now := time.Unix(1700000000, 0)
	doc := document.Document{
		ID:         "doc-1",
		UserID:     "demo-user",
		SourceName: "intro.md",
		Status:     "indexed",
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	mock.ExpectExec("INSERT INTO documents").
		WithArgs(doc.ID, doc.UserID, doc.SourceName, doc.Status, doc.ContentHash, doc.RawContent, pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	rows := pgxmock.NewRows([]string{
		"id", "user_id", "source_name", "status", "content_hash", "raw_content", "created_at", "updated_at",
	}).AddRow(doc.ID, doc.UserID, doc.SourceName, doc.Status, doc.ContentHash, doc.RawContent, now, now)
	mock.ExpectQuery("SELECT id, user_id, source_name").
		WithArgs("doc-1").
		WillReturnRows(rows)

	if err := repo.Create(context.Background(), doc); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	saved, err := repo.GetByID(context.Background(), "doc-1")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if saved.SourceName != "intro.md" {
		t.Fatalf("unexpected source name: %s", saved.SourceName)
	}
}

func TestDocumentRepository_UpsertDocumentReturnsPersistedDocumentOnConflict(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer mock.Close()

	repo := NewDocumentRepository(mock)
	now := time.Unix(1700000000, 0)
	incoming := document.Document{
		ID:          "doc-new",
		UserID:      "demo-user",
		SourceName:  "backend-interview-notes.md",
		Status:      "indexed",
		ContentHash: "hash-new",
		RawContent:  "new content",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	mock.ExpectQuery("INSERT INTO documents").
		WithArgs(
			incoming.ID,
			incoming.UserID,
			incoming.SourceName,
			incoming.Status,
			incoming.ContentHash,
			incoming.RawContent,
			pgxmock.AnyArg(),
			pgxmock.AnyArg(),
		).
		WillReturnRows(
			pgxmock.NewRows([]string{
				"id", "user_id", "source_name", "status", "content_hash", "raw_content", "created_at", "updated_at",
			}).AddRow(
				"doc-existing",
				incoming.UserID,
				incoming.SourceName,
				incoming.Status,
				incoming.ContentHash,
				incoming.RawContent,
				now.Add(-time.Hour),
				now,
			),
		)

	saved, err := repo.UpsertDocument(context.Background(), incoming)
	if err != nil {
		t.Fatalf("UpsertDocument() error = %v", err)
	}

	if saved.ID != "doc-existing" {
		t.Fatalf("expected persisted document id, got %s", saved.ID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestDocumentRepository_UpdateStatusAndListPendingForReindex(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer mock.Close()

	repo := NewDocumentRepository(mock)
	now := time.Unix(1700000000, 0)

	mock.ExpectExec("UPDATE documents SET status = \\$2, updated_at = \\$3 WHERE id = \\$1").
		WithArgs("doc-1", "index_failed", now).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	if err := repo.UpdateStatus(context.Background(), "doc-1", "index_failed", now); err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}

	rows := pgxmock.NewRows([]string{
		"id", "user_id", "source_name", "status", "content_hash", "raw_content", "created_at", "updated_at",
	}).AddRow(
		"doc-1",
		"demo-user",
		"backend.md",
		"index_failed",
		"hash-1",
		"doc content",
		now.Add(-time.Hour),
		now.Add(-time.Minute),
	)

	mock.ExpectQuery("SELECT id, user_id, source_name, status, content_hash, raw_content, created_at, updated_at FROM documents WHERE status IN \\('pending_index', 'index_failed'\\) AND updated_at <= \\$1 ORDER BY updated_at ASC LIMIT \\$2").
		WithArgs(now, 10).
		WillReturnRows(rows)

	documents, err := repo.ListPendingForReindex(context.Background(), now, 10)
	if err != nil {
		t.Fatalf("ListPendingForReindex() error = %v", err)
	}
	if len(documents) != 1 || documents[0].ID != "doc-1" {
		t.Fatalf("unexpected documents: %#v", documents)
	}
}
