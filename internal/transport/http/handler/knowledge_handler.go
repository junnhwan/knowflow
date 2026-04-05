package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"knowflow/internal/service/tools"
)

type ToolExecutor interface {
	Execute(ctx context.Context, name string, input map[string]any) (tools.Output, error)
}

type KnowledgeHandler struct {
	tools ToolExecutor
}

func NewKnowledgeHandler(tools ToolExecutor) *KnowledgeHandler {
	return &KnowledgeHandler{tools: tools}
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
