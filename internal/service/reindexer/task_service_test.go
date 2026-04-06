package reindexer

import (
	"context"
	"errors"
	"testing"
	"time"

	reindexdomain "knowflow/internal/domain/reindex"
)

func TestTaskService_CreateAndProcessMarksSuccess(t *testing.T) {
	store := &fakeTaskStore{}
	service := NewTaskService(store, fakeTaskExecutor{}, TaskServiceConfig{
		Now:   func() time.Time { return time.Unix(1700000000, 0) },
		NewID: func() string { return "task-1" },
	})

	task, err := service.CreateAndProcess(context.Background(), CreateTaskRequest{
		UserID:     "demo-user",
		DocumentID: "doc-1",
	})
	if err != nil {
		t.Fatalf("CreateAndProcess() error = %v", err)
	}

	if task.ID != "task-1" {
		t.Fatalf("unexpected task id: %s", task.ID)
	}
	if task.Status != "success" {
		t.Fatalf("expected success status, got %s", task.Status)
	}
	if task.AttemptCount != 1 {
		t.Fatalf("expected one attempt, got %d", task.AttemptCount)
	}
}

func TestTaskService_CreateAndProcessMarksFailure(t *testing.T) {
	store := &fakeTaskStore{}
	service := NewTaskService(store, fakeTaskExecutor{err: errors.New("reindex failed")}, TaskServiceConfig{
		Now:   func() time.Time { return time.Unix(1700000000, 0) },
		NewID: func() string { return "task-2" },
	})

	task, err := service.CreateAndProcess(context.Background(), CreateTaskRequest{
		UserID:           "demo-user",
		KnowledgeEntryID: "knowledge-1",
	})
	if err != nil {
		t.Fatalf("CreateAndProcess() error = %v", err)
	}

	if task.Status != "failed" {
		t.Fatalf("expected failed status, got %s", task.Status)
	}
	if task.ErrorMessage == "" {
		t.Fatal("expected error message to be recorded")
	}
}

func TestTaskService_ListAndGetByUser(t *testing.T) {
	store := &fakeTaskStore{
		tasks: map[string]reindexdomain.Task{
			"task-1": {ID: "task-1", UserID: "demo-user", Status: "success"},
			"task-2": {ID: "task-2", UserID: "other-user", Status: "failed"},
		},
	}
	service := NewTaskService(store, fakeTaskExecutor{}, TaskServiceConfig{})

	tasks, err := service.ListTasks(context.Background(), "demo-user")
	if err != nil {
		t.Fatalf("ListTasks() error = %v", err)
	}
	if len(tasks) != 1 || tasks[0].ID != "task-1" {
		t.Fatalf("unexpected tasks: %#v", tasks)
	}

	task, err := service.GetTask(context.Background(), "demo-user", "task-1")
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}
	if task.ID != "task-1" {
		t.Fatalf("unexpected task: %#v", task)
	}
}

type fakeTaskStore struct {
	tasks map[string]reindexdomain.Task
}

func (f *fakeTaskStore) Create(_ context.Context, task reindexdomain.Task) error {
	if f.tasks == nil {
		f.tasks = map[string]reindexdomain.Task{}
	}
	f.tasks[task.ID] = task
	return nil
}

func (f *fakeTaskStore) Update(_ context.Context, task reindexdomain.Task) error {
	if f.tasks == nil {
		f.tasks = map[string]reindexdomain.Task{}
	}
	f.tasks[task.ID] = task
	return nil
}

func (f *fakeTaskStore) GetByID(_ context.Context, taskID string) (reindexdomain.Task, error) {
	task, ok := f.tasks[taskID]
	if !ok {
		return reindexdomain.Task{}, ErrTaskNotFound
	}
	return task, nil
}

func (f *fakeTaskStore) ListByUser(_ context.Context, userID string, _ int) ([]reindexdomain.Task, error) {
	out := make([]reindexdomain.Task, 0, len(f.tasks))
	for _, task := range f.tasks {
		if task.UserID == userID {
			out = append(out, task)
		}
	}
	return out, nil
}

type fakeTaskExecutor struct {
	err error
}

func (f fakeTaskExecutor) Execute(_ context.Context, _ string, _ map[string]any) error {
	return f.err
}
