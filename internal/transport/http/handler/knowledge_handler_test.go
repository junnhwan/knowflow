package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	knowledgedomain "knowflow/internal/domain/knowledge"
	reindexdomain "knowflow/internal/domain/reindex"
	knowledgeservice "knowflow/internal/service/knowledge"
	"knowflow/internal/service/reindexer"
	"knowflow/internal/service/tools"
)

func TestKnowledgeHandler_ListAndGetEntries(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	governance := &fakeKnowledgeGovernanceService{
		listResult: []knowledgedomain.Entry{
			{
				ID:           "knowledge-1",
				UserID:       "demo-user",
				Title:        "Redis 双层记忆",
				ReviewStatus: "active",
				Status:       "indexed",
				UpdatedAt:    time.Unix(1700000000, 0),
			},
		},
		getResult: knowledgeservice.EntryDetail{
			Entry: knowledgedomain.Entry{
				ID:           "knowledge-1",
				UserID:       "demo-user",
				Title:        "Redis 双层记忆",
				ReviewStatus: "active",
				Status:       "indexed",
			},
		},
	}
	handler := NewKnowledgeHandler(fakeToolExecutor{}, governance, nil)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "demo-user")
	})
	router.GET("/api/kb/knowledge", handler.List)
	router.GET("/api/kb/knowledge/:knowledge_id", handler.Get)

	listReq := httptest.NewRequest(http.MethodGet, "/api/kb/knowledge", nil)
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("unexpected list status: %d", listRec.Code)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/kb/knowledge/knowledge-1", nil)
	getRec := httptest.NewRecorder()
	router.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("unexpected get status: %d", getRec.Code)
	}

	var detail knowledgeservice.EntryDetail
	if err := json.Unmarshal(getRec.Body.Bytes(), &detail); err != nil {
		t.Fatalf("unmarshal detail: %v", err)
	}
	if detail.Entry.ID != "knowledge-1" {
		t.Fatalf("unexpected detail payload: %#v", detail)
	}
}

func TestKnowledgeHandler_UpdateDeleteAndMerge(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	governance := &fakeKnowledgeGovernanceService{
		updateResult: knowledgeservice.EntryDetail{
			Entry: knowledgedomain.Entry{
				ID:           "knowledge-1",
				Title:        "更新后的知识标题",
				ReviewStatus: "active",
			},
		},
		disableResult: knowledgedomain.Entry{
			ID:           "knowledge-1",
			ReviewStatus: "disabled",
		},
		mergeResult: knowledgeservice.MergeResult{
			SourceEntry: knowledgedomain.Entry{
				ID:           "knowledge-1",
				ReviewStatus: "merged",
				MergedIntoID: "knowledge-2",
			},
			TargetEntry: knowledgedomain.Entry{
				ID:           "knowledge-2",
				ReviewStatus: "active",
			},
		},
	}
	handler := NewKnowledgeHandler(fakeToolExecutor{}, governance, nil)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "demo-user")
	})
	router.PUT("/api/kb/knowledge/:knowledge_id", handler.Update)
	router.DELETE("/api/kb/knowledge/:knowledge_id", handler.Delete)
	router.POST("/api/kb/knowledge/:knowledge_id/merge", handler.Merge)

	updateReq := httptest.NewRequest(http.MethodPut, "/api/kb/knowledge/knowledge-1", strings.NewReader(`{"title":"更新后的知识标题","review_status":"active"}`))
	updateReq.Header.Set("Content-Type", "application/json")
	updateRec := httptest.NewRecorder()
	router.ServeHTTP(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("unexpected update status: %d", updateRec.Code)
	}
	if governance.lastUpdate.KnowledgeID != "knowledge-1" {
		t.Fatalf("unexpected update request: %#v", governance.lastUpdate)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/kb/knowledge/knowledge-1", nil)
	deleteRec := httptest.NewRecorder()
	router.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("unexpected delete status: %d", deleteRec.Code)
	}

	mergeReq := httptest.NewRequest(http.MethodPost, "/api/kb/knowledge/knowledge-1/merge", strings.NewReader(`{"target_entry_id":"knowledge-2"}`))
	mergeReq.Header.Set("Content-Type", "application/json")
	mergeRec := httptest.NewRecorder()
	router.ServeHTTP(mergeRec, mergeReq)
	if mergeRec.Code != http.StatusOK {
		t.Fatalf("unexpected merge status: %d", mergeRec.Code)
	}
	if governance.lastMerge.SourceEntryID != "knowledge-1" || governance.lastMerge.TargetEntryID != "knowledge-2" {
		t.Fatalf("unexpected merge request: %#v", governance.lastMerge)
	}
}

func TestKnowledgeHandler_ReindexAndTaskQuery(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	tasks := &fakeReindexTaskService{
		createResult: reindexdomain.Task{
			ID:           "task-1",
			Status:       "success",
			TargetType:   "knowledge_entry",
			TargetID:     "knowledge-1",
			AttemptCount: 1,
		},
		listResult: []reindexdomain.Task{
			{ID: "task-1", Status: "success"},
		},
		getResult: reindexdomain.Task{
			ID:         "task-1",
			Status:     "success",
			TargetType: "knowledge_entry",
			TargetID:   "knowledge-1",
		},
	}
	handler := NewKnowledgeHandler(fakeToolExecutor{}, &fakeKnowledgeGovernanceService{}, tasks)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "demo-user")
	})
	router.POST("/api/kb/reindex", handler.Reindex)
	router.GET("/api/kb/reindex/tasks", handler.ListReindexTasks)
	router.GET("/api/kb/reindex/tasks/:task_id", handler.GetReindexTask)

	reindexReq := httptest.NewRequest(http.MethodPost, "/api/kb/reindex", strings.NewReader(`{"knowledge_entry_id":"knowledge-1"}`))
	reindexReq.Header.Set("Content-Type", "application/json")
	reindexRec := httptest.NewRecorder()
	router.ServeHTTP(reindexRec, reindexReq)
	if reindexRec.Code != http.StatusOK {
		t.Fatalf("unexpected reindex status: %d", reindexRec.Code)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/kb/reindex/tasks", nil)
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("unexpected task list status: %d", listRec.Code)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/kb/reindex/tasks/task-1", nil)
	getRec := httptest.NewRecorder()
	router.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("unexpected task get status: %d", getRec.Code)
	}
	if tasks.lastCreate.KnowledgeEntryID != "knowledge-1" {
		t.Fatalf("unexpected create request: %#v", tasks.lastCreate)
	}
}

type fakeToolExecutor struct{}

func (fakeToolExecutor) Execute(_ context.Context, _ string, _ map[string]any) (tools.Output, error) {
	return tools.Output{Status: "success"}, nil
}

type fakeKnowledgeGovernanceService struct {
	listResult    []knowledgedomain.Entry
	getResult     knowledgeservice.EntryDetail
	updateResult  knowledgeservice.EntryDetail
	disableResult knowledgedomain.Entry
	mergeResult   knowledgeservice.MergeResult
	lastUpdate    knowledgeservice.UpdateEntryRequest
	lastMerge     knowledgeservice.MergeEntriesRequest
}

func (f *fakeKnowledgeGovernanceService) ListEntries(_ context.Context, _ string, _ knowledgeservice.ListFilter) ([]knowledgedomain.Entry, error) {
	return f.listResult, nil
}

func (f *fakeKnowledgeGovernanceService) GetEntry(_ context.Context, _ string, _ string) (knowledgeservice.EntryDetail, error) {
	return f.getResult, nil
}

func (f *fakeKnowledgeGovernanceService) UpdateEntry(_ context.Context, req knowledgeservice.UpdateEntryRequest) (knowledgeservice.EntryDetail, error) {
	f.lastUpdate = req
	return f.updateResult, nil
}

func (f *fakeKnowledgeGovernanceService) DisableEntry(_ context.Context, _ string, _ string) (knowledgedomain.Entry, error) {
	return f.disableResult, nil
}

func (f *fakeKnowledgeGovernanceService) MergeEntries(_ context.Context, req knowledgeservice.MergeEntriesRequest) (knowledgeservice.MergeResult, error) {
	f.lastMerge = req
	return f.mergeResult, nil
}

type fakeReindexTaskService struct {
	createResult reindexdomain.Task
	listResult   []reindexdomain.Task
	getResult    reindexdomain.Task
	lastCreate   reindexer.CreateTaskRequest
}

func (f *fakeReindexTaskService) CreateAndProcess(_ context.Context, req reindexer.CreateTaskRequest) (reindexdomain.Task, error) {
	f.lastCreate = req
	return f.createResult, nil
}

func (f *fakeReindexTaskService) ListTasks(_ context.Context, _ string) ([]reindexdomain.Task, error) {
	return f.listResult, nil
}

func (f *fakeReindexTaskService) GetTask(_ context.Context, _ string, _ string) (reindexdomain.Task, error) {
	return f.getResult, nil
}
