package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	chatdomain "knowflow/internal/domain/chat"
	chatservice "knowflow/internal/service/chat"
)

type ChatQueryService interface {
	Query(ctx context.Context, req chatservice.QueryRequest) (chatservice.QueryResponse, error)
}

type ConversationReader interface {
	ListSessions(ctx context.Context, userID string) ([]chatdomain.Session, error)
	ListMessages(ctx context.Context, sessionID string) ([]chatdomain.Message, error)
}

type ChatHandler struct {
	queryService ChatQueryService
	reader       ConversationReader
}

func NewChatHandler(queryService ChatQueryService, reader ConversationReader) *ChatHandler {
	return &ChatHandler{
		queryService: queryService,
		reader:       reader,
	}
}

func (h *ChatHandler) Query(c *gin.Context) {
	var request chatservice.QueryRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	request.UserID = c.GetString("user_id")
	if request.SessionID != "" {
		c.Set("session_id", request.SessionID)
	}

	response, err := h.queryService.Query(c.Request.Context(), request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, response)
}

func (h *ChatHandler) QueryStream(c *gin.Context) {
	var request chatservice.QueryRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	request.UserID = c.GetString("user_id")

	response, err := h.queryService.Query(c.Request.Context(), request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	for _, chunk := range chunkAnswer(response.Answer, 24) {
		c.SSEvent("delta", gin.H{"content": chunk})
		c.Writer.Flush()
	}
	c.SSEvent("done", response)
}

func (h *ChatHandler) ListSessions(c *gin.Context) {
	sessions, err := h.reader.ListSessions(c.Request.Context(), c.GetString("user_id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sessions)
}

func (h *ChatHandler) ListMessages(c *gin.Context) {
	messages, err := h.reader.ListMessages(c.Request.Context(), c.Param("session_id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, messages)
}

func chunkAnswer(answer string, size int) []string {
	runes := []rune(answer)
	if size <= 0 || len(runes) <= size {
		return []string{answer}
	}

	chunks := make([]string, 0, (len(runes)/size)+1)
	for start := 0; start < len(runes); start += size {
		end := start + size
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[start:end]))
	}
	return chunks
}
