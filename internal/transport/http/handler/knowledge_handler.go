package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	knowledgedomain "knowflow/internal/domain/knowledge"
	knowledgeservice "knowflow/internal/service/knowledge"
	"knowflow/internal/service/tools"
)

type ToolExecutor interface {
	Execute(ctx context.Context, name string, input map[string]any) (tools.Output, error)
}

type KnowledgeGovernanceService interface {
	ListEntries(ctx context.Context, userID string, filter knowledgeservice.ListFilter) ([]knowledgedomain.Entry, error)
	GetEntry(ctx context.Context, userID, knowledgeID string) (knowledgeservice.EntryDetail, error)
	UpdateEntry(ctx context.Context, req knowledgeservice.UpdateEntryRequest) (knowledgeservice.EntryDetail, error)
	DisableEntry(ctx context.Context, userID, knowledgeID string) (knowledgedomain.Entry, error)
	MergeEntries(ctx context.Context, req knowledgeservice.MergeEntriesRequest) (knowledgeservice.MergeResult, error)
}

type KnowledgeHandler struct {
	tools      ToolExecutor
	governance KnowledgeGovernanceService
}

func NewKnowledgeHandler(tools ToolExecutor, governance KnowledgeGovernanceService, _ any) *KnowledgeHandler {
	return &KnowledgeHandler{tools: tools, governance: governance}
}

func (h *KnowledgeHandler) Upsert(c *gin.Context) {
	var payload map[string]any
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, exists := payload["user_id"]; !exists {
		payload["user_id"] = c.GetString("user_id")
	}
	result, err := h.tools.Execute(c.Request.Context(), "upsert_knowledge", payload)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "result": result})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *KnowledgeHandler) Reindex(c *gin.Context) {
	var payload map[string]any
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := h.tools.Execute(c.Request.Context(), "refresh_document_index", payload)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "result": result})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *KnowledgeHandler) List(c *gin.Context) {
	if h.governance == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "knowledge governance is not configured"})
		return
	}
	entries, err := h.governance.ListEntries(c.Request.Context(), c.GetString("user_id"), knowledgeservice.ListFilter{
		ReviewStatus: c.Query("review_status"),
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, entries)
}

func (h *KnowledgeHandler) Get(c *gin.Context) {
	if h.governance == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "knowledge governance is not configured"})
		return
	}
	result, err := h.governance.GetEntry(c.Request.Context(), c.GetString("user_id"), c.Param("knowledge_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *KnowledgeHandler) Update(c *gin.Context) {
	if h.governance == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "knowledge governance is not configured"})
		return
	}
	var req knowledgeservice.UpdateEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.UserID = c.GetString("user_id")
	req.KnowledgeID = c.Param("knowledge_id")
	result, err := h.governance.UpdateEntry(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *KnowledgeHandler) Delete(c *gin.Context) {
	if h.governance == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "knowledge governance is not configured"})
		return
	}
	entry, err := h.governance.DisableEntry(c.Request.Context(), c.GetString("user_id"), c.Param("knowledge_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, entry)
}

func (h *KnowledgeHandler) Merge(c *gin.Context) {
	if h.governance == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "knowledge governance is not configured"})
		return
	}
	var body struct {
		TargetEntryID string `json:"target_entry_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := h.governance.MergeEntries(c.Request.Context(), knowledgeservice.MergeEntriesRequest{
		UserID:        c.GetString("user_id"),
		SourceEntryID: c.Param("knowledge_id"),
		TargetEntryID: body.TargetEntryID,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}
