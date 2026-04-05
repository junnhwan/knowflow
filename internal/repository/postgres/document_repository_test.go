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
