package postgres

import (
	"context"

	reindexdomain "knowflow/internal/domain/reindex"
	pgplatform "knowflow/internal/platform/postgres"
)

type ReindexTaskRepository struct {
	db pgplatform.DB
}

func NewReindexTaskRepository(db pgplatform.DB) *ReindexTaskRepository {
	return &ReindexTaskRepository{db: db}
}

func (r *ReindexTaskRepository) Create(ctx context.Context, task reindexdomain.Task) error {
	_, err := r.db.Exec(ctx, `
INSERT INTO reindex_tasks (id, user_id, target_type, target_id, trigger_type, status, attempt_count, error_message, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
`, task.ID, task.UserID, task.TargetType, task.TargetID, task.TriggerType, task.Status, task.AttemptCount, task.ErrorMessage, task.CreatedAt, task.UpdatedAt)
	return err
}

func (r *ReindexTaskRepository) Update(ctx context.Context, task reindexdomain.Task) error {
	_, err := r.db.Exec(ctx, `
UPDATE reindex_tasks
SET user_id = $2, target_type = $3, target_id = $4, trigger_type = $5, status = $6, attempt_count = $7, error_message = $8, created_at = $9, updated_at = $10
WHERE id = $1
`, task.ID, task.UserID, task.TargetType, task.TargetID, task.TriggerType, task.Status, task.AttemptCount, task.ErrorMessage, task.CreatedAt, task.UpdatedAt)
	return err
}

func (r *ReindexTaskRepository) GetByID(ctx context.Context, taskID string) (reindexdomain.Task, error) {
	row := r.db.QueryRow(ctx, `
SELECT id, user_id, target_type, target_id, trigger_type, status, attempt_count, error_message, created_at, updated_at
FROM reindex_tasks
WHERE id = $1
`, taskID)

	var task reindexdomain.Task
	if err := row.Scan(&task.ID, &task.UserID, &task.TargetType, &task.TargetID, &task.TriggerType, &task.Status, &task.AttemptCount, &task.ErrorMessage, &task.CreatedAt, &task.UpdatedAt); err != nil {
		return reindexdomain.Task{}, err
	}
	return task, nil
}

func (r *ReindexTaskRepository) ListByUser(ctx context.Context, userID string, limit int) ([]reindexdomain.Task, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.db.Query(ctx, `
SELECT id, user_id, target_type, target_id, trigger_type, status, attempt_count, error_message, created_at, updated_at
FROM reindex_tasks
WHERE user_id = $1
ORDER BY updated_at DESC
LIMIT $2
`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []reindexdomain.Task
	for rows.Next() {
		var task reindexdomain.Task
		if err := rows.Scan(&task.ID, &task.UserID, &task.TargetType, &task.TargetID, &task.TriggerType, &task.Status, &task.AttemptCount, &task.ErrorMessage, &task.CreatedAt, &task.UpdatedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}
