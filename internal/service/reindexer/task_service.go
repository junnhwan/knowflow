package reindexer

import (
	"context"
	"errors"
	"fmt"
	"time"

	reindexdomain "knowflow/internal/domain/reindex"
)

var ErrTaskNotFound = errors.New("reindex task not found")

type TaskStore interface {
	Create(ctx context.Context, task reindexdomain.Task) error
	Update(ctx context.Context, task reindexdomain.Task) error
	GetByID(ctx context.Context, taskID string) (reindexdomain.Task, error)
	ListByUser(ctx context.Context, userID string, limit int) ([]reindexdomain.Task, error)
}

type TaskExecutor interface {
	Execute(ctx context.Context, toolName string, input map[string]any) error
}

type TaskObserver interface {
	RecordReindexTask(targetType, result string)
}

type TaskServiceConfig struct {
	Now      func() time.Time
	NewID    func() string
	Observer TaskObserver
}

type CreateTaskRequest struct {
	UserID           string `json:"user_id"`
	DocumentID       string `json:"document_id"`
	KnowledgeEntryID string `json:"knowledge_entry_id"`
	TriggerType      string `json:"trigger_type"`
}

type TaskService struct {
	store    TaskStore
	executor TaskExecutor
	now      func() time.Time
	newID    func() string
	observer TaskObserver
}

func NewTaskService(store TaskStore, executor TaskExecutor, cfg TaskServiceConfig) *TaskService {
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	newID := cfg.NewID
	if newID == nil {
		newID = func() string {
			return fmt.Sprintf("reindex-task-%d", time.Now().UnixNano())
		}
	}
	return &TaskService{
		store:    store,
		executor: executor,
		now:      now,
		newID:    newID,
		observer: cfg.Observer,
	}
}

func (s *TaskService) CreateAndProcess(ctx context.Context, req CreateTaskRequest) (reindexdomain.Task, error) {
	targetType, targetID, payload, err := buildTaskPayload(req)
	if err != nil {
		return reindexdomain.Task{}, err
	}
	if req.TriggerType == "" {
		req.TriggerType = "manual"
	}

	now := s.now()
	task := reindexdomain.Task{
		ID:          s.newID(),
		UserID:      req.UserID,
		TargetType:  targetType,
		TargetID:    targetID,
		TriggerType: req.TriggerType,
		Status:      "pending",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.store.Create(ctx, task); err != nil {
		return reindexdomain.Task{}, err
	}

	task.Status = "running"
	task.AttemptCount = 1
	task.UpdatedAt = s.now()
	if err := s.store.Update(ctx, task); err != nil {
		return reindexdomain.Task{}, err
	}

	execErr := s.executor.Execute(ctx, "refresh_document_index", payload)
	task.UpdatedAt = s.now()
	if execErr != nil {
		task.Status = "failed"
		task.ErrorMessage = execErr.Error()
		if s.observer != nil {
			s.observer.RecordReindexTask(targetType, "failed")
		}
	} else {
		task.Status = "success"
		task.ErrorMessage = ""
		if s.observer != nil {
			s.observer.RecordReindexTask(targetType, "success")
		}
	}
	if err := s.store.Update(ctx, task); err != nil {
		return reindexdomain.Task{}, err
	}
	return task, nil
}

func (s *TaskService) ListTasks(ctx context.Context, userID string) ([]reindexdomain.Task, error) {
	return s.store.ListByUser(ctx, userID, 20)
}

func (s *TaskService) GetTask(ctx context.Context, userID, taskID string) (reindexdomain.Task, error) {
	task, err := s.store.GetByID(ctx, taskID)
	if err != nil {
		return reindexdomain.Task{}, err
	}
	if task.UserID != userID {
		return reindexdomain.Task{}, ErrTaskNotFound
	}
	return task, nil
}

func buildTaskPayload(req CreateTaskRequest) (string, string, map[string]any, error) {
	if req.DocumentID != "" && req.KnowledgeEntryID != "" {
		return "", "", nil, fmt.Errorf("document_id and knowledge_entry_id cannot both be set")
	}
	if req.DocumentID == "" && req.KnowledgeEntryID == "" {
		return "", "", nil, fmt.Errorf("document_id or knowledge_entry_id is required")
	}
	if req.DocumentID != "" {
		return "document", req.DocumentID, map[string]any{"document_id": req.DocumentID}, nil
	}
	return "knowledge_entry", req.KnowledgeEntryID, map[string]any{"knowledge_entry_id": req.KnowledgeEntryID}, nil
}
