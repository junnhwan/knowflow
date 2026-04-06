package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	chatdomain "knowflow/internal/domain/chat"
	chatservice "knowflow/internal/service/chat"
	"knowflow/internal/service/guardrail"
)

type ChatQueryService interface {
	Query(ctx context.Context, req chatservice.QueryRequest) (chatservice.QueryResponse, error)
	QueryStream(ctx context.Context, req chatservice.QueryRequest, onDelta func(string) error) (chatservice.QueryResponse, error)
}

type ConversationReader interface {
	ListSessions(ctx context.Context, userID string) ([]chatdomain.Session, error)
	ListMessages(ctx context.Context, sessionID string) ([]chatdomain.Message, error)
}

type ChatHandler struct {
	queryService ChatQueryService
	reader       ConversationReader
	guardrail    *guardrail.Service
	observer     GuardrailObserver
	logger       RequestLogger
}

type GuardrailObserver interface {
	RecordGuardrailReject(endpoint, reason string)
}

type RequestLogger interface {
	Warn(msg string, args ...any)
}

func NewChatHandler(queryService ChatQueryService, reader ConversationReader, guardrailService *guardrail.Service, observer GuardrailObserver, logger RequestLogger) *ChatHandler {
	return &ChatHandler{
		queryService: queryService,
		reader:       reader,
		guardrail:    guardrailService,
		observer:     observer,
		logger:       logger,
	}
}

func (h *ChatHandler) Query(c *gin.Context) {
	var request chatservice.QueryRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.guardrail != nil {
		if err := h.guardrail.Validate(request.Message); err != nil {
			h.recordGuardrailReject(c, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
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
	if h.guardrail != nil {
		if err := h.guardrail.Validate(request.Message); err != nil {
			h.recordGuardrailReject(c, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	request.UserID = c.GetString("user_id")

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	streamStarted := false
	response, err := h.queryService.QueryStream(c.Request.Context(), request, func(delta string) error {
		streamStarted = true
		c.SSEvent("delta", gin.H{"content": delta})
		c.Writer.Flush()
		return nil
	})
	if err != nil {
		if !streamStarted {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.SSEvent("error", gin.H{"error": err.Error()})
		c.Writer.Flush()
		return
	}
	if response.SessionID != "" {
		c.Set("session_id", response.SessionID)
	}
	c.SSEvent("done", response)
	c.Writer.Flush()
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

func (h *ChatHandler) recordGuardrailReject(c *gin.Context, err error) {
	reason := guardrail.Reason(err)
	if h.observer != nil {
		h.observer.RecordGuardrailReject(c.FullPath(), reason)
	}
	if h.logger != nil {
		h.logger.Warn("guardrail rejected request",
			"path", c.FullPath(),
			"request_id", c.GetString("request_id"),
			"user_id", c.GetString("user_id"),
			"session_id", c.GetString("session_id"),
			"reason", reason,
		)
	}
}
