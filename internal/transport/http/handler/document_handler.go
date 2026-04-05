package handler

import (
	"context"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"knowflow/internal/service/ingestion"
)

type DocumentIngestionService interface {
	Ingest(ctx context.Context, req ingestion.IngestRequest) (ingestion.IngestResult, error)
}

type DocumentHandler struct {
	service DocumentIngestionService
}

func NewDocumentHandler(service DocumentIngestionService) *DocumentHandler {
	return &DocumentHandler{service: service}
}

func (h *DocumentHandler) Upload(c *gin.Context) {
	userID := c.GetString("user_id")

	sourceName := c.PostForm("source_name")
	content := c.PostForm("content")

	file, err := c.FormFile("file")
	if err == nil {
		sourceName = file.Filename
		src, openErr := file.Open()
		if openErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": openErr.Error()})
			return
		}
		defer src.Close()
		body, readErr := io.ReadAll(src)
		if readErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": readErr.Error()})
			return
		}
		content = string(body)
	}

	result, err := h.service.Ingest(c.Request.Context(), ingestion.IngestRequest{
		UserID:     userID,
		SourceName: sourceName,
		Content:    content,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}
