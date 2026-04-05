package chat

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	chatdomain "knowflow/internal/domain/chat"
	"knowflow/internal/service/memory"
	"knowflow/internal/service/retrieval"
	"knowflow/internal/service/tools"
)

var ErrSessionNotFound = errors.New("session not found")

type ChatStore interface {
	CreateSession(ctx context.Context, session chatdomain.Session) error
	GetSession(ctx context.Context, sessionID string) (chatdomain.Session, error)
	ListSessions(ctx context.Context, userID string) ([]chatdomain.Session, error)
	AppendMessage(ctx context.Context, message chatdomain.Message) error
	ListMessages(ctx context.Context, sessionID string) ([]chatdomain.Message, error)
}

type MemoryService interface {
	Load(ctx context.Context, userID, sessionID string) (memory.LoadResult, error)
	Update(ctx context.Context, req memory.UpdateRequest) (memory.UpdateResult, error)
}

type ToolExecutor interface {
	Execute(ctx context.Context, name string, input map[string]any) (tools.Output, error)
}

type Answerer interface {
	Generate(ctx context.Context, req PromptRequest) (PromptResult, error)
	Stream(ctx context.Context, req PromptRequest, onDelta func(string) error) (PromptResult, error)
}

type Dependencies struct {
	ChatStore ChatStore
	Memory    MemoryService
	Tools     ToolExecutor
	Answerer  Answerer
	Now       func() time.Time
	NewID     func(prefix string) string
}

type PromptRequest struct {
	UserID     string
	SessionID  string
	Message    string
	History    []memory.MessageMemory
	Citations  []retrieval.Citation
	Candidates []retrieval.Candidate
}

type PromptResult struct {
	Answer string
}

type QueryRequest struct {
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
}

type QueryResponse struct {
	SessionID     string               `json:"session_id"`
	Answer        string               `json:"answer"`
	Citations     []retrieval.Citation `json:"citations"`
	ToolTraces    []tools.Trace        `json:"tool_traces,omitempty"`
	RetrievalMeta retrieval.Metadata   `json:"retrieval_meta"`
}

type Orchestrator struct {
	store    ChatStore
	memory   MemoryService
	tools    ToolExecutor
	answerer Answerer
	now      func() time.Time
	newID    func(prefix string) string
}

func NewOrchestrator(deps Dependencies) *Orchestrator {
	now := deps.Now
	if now == nil {
		now = time.Now
	}
	newID := deps.NewID
	if newID == nil {
		newID = func(prefix string) string {
			return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
		}
	}
	return &Orchestrator{
		store:    deps.ChatStore,
		memory:   deps.Memory,
		tools:    deps.Tools,
		answerer: deps.Answerer,
		now:      now,
		newID:    newID,
	}
}

func (o *Orchestrator) Query(ctx context.Context, req QueryRequest) (QueryResponse, error) {
	sessionID, err := o.ensureSession(ctx, req)
	if err != nil {
		return QueryResponse{}, err
	}

	history, err := o.memory.Load(ctx, req.UserID, sessionID)
	if err != nil {
		return QueryResponse{}, err
	}

	trace := tools.Trace{ToolName: "retrieve_knowledge", Status: "success"}
	toolOutput, err := o.tools.Execute(ctx, "retrieve_knowledge", map[string]any{
		"user_id": req.UserID,
		"query":   req.Message,
		"top_k":   5,
	})
	if err != nil {
		trace.Status = "error"
		trace.Error = err.Error()
	}

	retrievalResult := extractRetrievalResult(toolOutput)
	if !retrievalResult.Meta.Hit || len(retrievalResult.Citations) == 0 {
		answer := "当前知识库里没有足够依据来支持这个问题，请先补充相关面试资料或知识条目。"
		if err := o.persistRound(ctx, sessionID, req.Message, answer, nil, nil); err != nil {
			return QueryResponse{}, err
		}
		_, _ = o.memory.Update(ctx, memory.UpdateRequest{
			UserID:    req.UserID,
			SessionID: sessionID,
			Incoming: []memory.MessageMemory{
				{Role: "user", Content: req.Message},
				{Role: "assistant", Content: answer},
			},
		})
		return QueryResponse{
			SessionID:     sessionID,
			Answer:        answer,
			Citations:     nil,
			ToolTraces:    []tools.Trace{trace},
			RetrievalMeta: retrievalResult.Meta,
		}, nil
	}

	promptResult, err := o.answerer.Generate(ctx, PromptRequest{
		UserID:     req.UserID,
		SessionID:  sessionID,
		Message:    req.Message,
		History:    history.Combined,
		Citations:  retrievalResult.Citations,
		Candidates: retrievalResult.Chunks,
	})
	if err != nil {
		return QueryResponse{}, err
	}

	if err := o.persistRound(ctx, sessionID, req.Message, promptResult.Answer, nil, nil); err != nil {
		return QueryResponse{}, err
	}

	_, _ = o.memory.Update(ctx, memory.UpdateRequest{
		UserID:    req.UserID,
		SessionID: sessionID,
		Incoming: []memory.MessageMemory{
			{Role: "user", Content: req.Message},
			{Role: "assistant", Content: promptResult.Answer},
		},
	})

	return QueryResponse{
		SessionID:     sessionID,
		Answer:        promptResult.Answer,
		Citations:     retrievalResult.Citations,
		ToolTraces:    []tools.Trace{trace},
		RetrievalMeta: retrievalResult.Meta,
	}, nil
}

func (o *Orchestrator) ensureSession(ctx context.Context, req QueryRequest) (string, error) {
	if req.SessionID != "" {
		if _, err := o.store.GetSession(ctx, req.SessionID); err == nil {
			return req.SessionID, nil
		}
	}

	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = o.newID("session")
	}

	session := chatdomain.Session{
		ID:           sessionID,
		UserID:       req.UserID,
		Title:        buildSessionTitle(req.Message),
		LastActiveAt: o.now(),
		CreatedAt:    o.now(),
	}
	if err := o.store.CreateSession(ctx, session); err != nil {
		return "", err
	}
	return sessionID, nil
}

func buildSessionTitle(message string) string {
	trimmed := strings.TrimSpace(message)
	runes := []rune(trimmed)
	if len(runes) <= 24 {
		return trimmed
	}
	return string(runes[:24])
}

func (o *Orchestrator) persistRound(ctx context.Context, sessionID, question, answer string, toolInput, toolOutput map[string]any) error {
	userMessage := chatdomain.Message{
		ID:        o.newID("msg"),
		SessionID: sessionID,
		Role:      "user",
		Content:   question,
		CreatedAt: o.now(),
	}
	if err := o.store.AppendMessage(ctx, userMessage); err != nil {
		return err
	}

	assistantMessage := chatdomain.Message{
		ID:         o.newID("msg"),
		SessionID:  sessionID,
		Role:       "assistant",
		Content:    answer,
		ToolInput:  toolInput,
		ToolOutput: toolOutput,
		CreatedAt:  o.now(),
	}
	return o.store.AppendMessage(ctx, assistantMessage)
}

func extractRetrievalResult(output tools.Output) retrieval.Result {
	result, ok := output.Data.(retrieval.Result)
	if ok {
		return result
	}
	return retrieval.Result{}
}
