package postgres

import (
	"context"
	"testing"
	"time"

	reindexdomain "knowflow/internal/domain/reindex"

	pgxmock "github.com/pashagolub/pgxmock/v4"
)

func TestReindexTaskRepository_CreateUpdateAndQuery(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer mock.Close()

	repo := NewReindexTaskRepository(mock)
	now := time.Unix(1700000000, 0)
	task := reindexdomain.Task{
		ID:           "task-1",
		UserID:       "demo-user",
		TargetType:   "knowledge_entry",
		TargetID:     "knowledge-1",
		TriggerType:  "manual",
		Status:       "success",
		AttemptCount: 1,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	mock.ExpectExec("INSERT INTO reindex_tasks").
		WithArgs(task.ID, task.UserID, task.TargetType, task.TargetID, task.TriggerType, task.Status, task.AttemptCount, task.ErrorMessage, task.CreatedAt, task.UpdatedAt).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	if err := repo.Create(context.Background(), task); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	task.Status = "failed"
	task.ErrorMessage = "boom"
	mock.ExpectExec("UPDATE reindex_tasks").
		WithArgs(task.ID, task.UserID, task.TargetType, task.TargetID, task.TriggerType, task.Status, task.AttemptCount, task.ErrorMessage, task.CreatedAt, task.UpdatedAt).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	if err := repo.Update(context.Background(), task); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	row := pgxmock.NewRows([]string{
		"id", "user_id", "target_type", "target_id", "trigger_type", "status", "attempt_count", "error_message", "created_at", "updated_at",
	}).AddRow(task.ID, task.UserID, task.TargetType, task.TargetID, task.TriggerType, task.Status, task.AttemptCount, task.ErrorMessage, task.CreatedAt, task.UpdatedAt)

	mock.ExpectQuery("SELECT id, user_id, target_type, target_id, trigger_type, status, attempt_count, error_message, created_at, updated_at FROM reindex_tasks WHERE id = \\$1").
		WithArgs("task-1").
		WillReturnRows(row)

	got, err := repo.GetByID(context.Background(), "task-1")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.Status != "failed" {
		t.Fatalf("unexpected task status: %s", got.Status)
	}

	rows := pgxmock.NewRows([]string{
		"id", "user_id", "target_type", "target_id", "trigger_type", "status", "attempt_count", "error_message", "created_at", "updated_at",
	}).AddRow(task.ID, task.UserID, task.TargetType, task.TargetID, task.TriggerType, task.Status, task.AttemptCount, task.ErrorMessage, task.CreatedAt, task.UpdatedAt)

	mock.ExpectQuery("SELECT id, user_id, target_type, target_id, trigger_type, status, attempt_count, error_message, created_at, updated_at FROM reindex_tasks WHERE user_id = \\$1 ORDER BY updated_at DESC LIMIT \\$2").
		WithArgs("demo-user", 20).
		WillReturnRows(rows)

	list, err := repo.ListByUser(context.Background(), "demo-user", 20)
	if err != nil {
		t.Fatalf("ListByUser() error = %v", err)
	}
	if len(list) != 1 || list[0].ID != "task-1" {
		t.Fatalf("unexpected tasks: %#v", list)
	}
}
