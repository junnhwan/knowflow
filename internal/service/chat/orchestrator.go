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

type KnowledgeExtractor interface {
	Extract(ctx context.Context, req KnowledgeExtractionRequest) (KnowledgeDraft, error)
}

type OutputGuardrail interface {
	ValidateOutput(answer string) error
}

type GuardrailObserver interface {
	RecordGuardrailReject(endpoint, reason string)
}

type GuardrailLogger interface {
	Warn(msg string, args ...any)
}

type Answerer interface {
	Generate(ctx context.Context, req PromptRequest) (PromptResult, error)
	Stream(ctx context.Context, req PromptRequest, onDelta func(string) error) (PromptResult, error)
}

type Dependencies struct {
	ChatStore          ChatStore
	Memory             MemoryService
	Tools              ToolExecutor
	KnowledgeExtractor KnowledgeExtractor
	OutputGuardrail    OutputGuardrail
	GuardrailObserver  GuardrailObserver
	GuardrailLogger    GuardrailLogger
	Answerer           Answerer
	AutoKnowledge      AutoKnowledgeConfig
	Now                func() time.Time
	NewID              func(prefix string) string
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
	store              ChatStore
	memory             MemoryService
	tools              ToolExecutor
	knowledgeExtractor KnowledgeExtractor
	outputGuardrail    OutputGuardrail
	guardrailObserver  GuardrailObserver
	guardrailLogger    GuardrailLogger
	answerer           Answerer
	autoKnowledge      AutoKnowledgeConfig
	now                func() time.Time
	newID              func(prefix string) string
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
		store:              deps.ChatStore,
		memory:             deps.Memory,
		tools:              deps.Tools,
		knowledgeExtractor: defaultKnowledgeExtractor(deps.KnowledgeExtractor),
		outputGuardrail:    deps.OutputGuardrail,
		guardrailObserver:  deps.GuardrailObserver,
		guardrailLogger:    deps.GuardrailLogger,
		answerer:           deps.Answerer,
		autoKnowledge:      defaultAutoKnowledgeConfig(deps.AutoKnowledge),
		now:                now,
		newID:              newID,
	}
}

func (o *Orchestrator) Query(ctx context.Context, req QueryRequest) (QueryResponse, error) {
	result, err := o.prepareQuery(ctx, req)
	if err != nil {
		return QueryResponse{}, err
	}
	if result.response != nil {
		return *result.response, nil
	}

	promptResult, err := o.answerer.Generate(ctx, result.prompt)
	if err != nil {
		return QueryResponse{}, err
	}

	return o.finalizeQuery(ctx, result, o.guardOutput("/api/chat/query#output", result.sessionID, result.request.UserID, promptResult.Answer))
}

func (o *Orchestrator) QueryStream(ctx context.Context, req QueryRequest, onDelta func(string) error) (QueryResponse, error) {
	result, err := o.prepareQuery(ctx, req)
	if err != nil {
		return QueryResponse{}, err
	}
	if result.response != nil {
		result.response.Answer = o.guardOutput("/api/chat/query/stream#output", result.sessionID, result.request.UserID, result.response.Answer)
		if onDelta != nil {
			if err := onDelta(result.response.Answer); err != nil {
				return QueryResponse{}, err
			}
		}
		return *result.response, nil
	}

	var streamed strings.Builder
	promptResult, err := o.answerer.Stream(ctx, result.prompt, func(delta string) error {
		candidate := streamed.String() + delta
		if guardErr := o.validateOutput(candidate); guardErr != nil {
			return outputBlockedError{err: guardErr}
		}
		streamed.WriteString(delta)
		if onDelta != nil {
			return onDelta(delta)
		}
		return nil
	})
	if err != nil {
		var blocked outputBlockedError
		if errors.As(err, &blocked) {
			return o.finalizeQuery(ctx, result, o.guardOutput("/api/chat/query/stream#output", result.sessionID, result.request.UserID, safeOutputAnswer()))
		}
		return QueryResponse{}, err
	}

	answer := promptResult.Answer
	if streamed.Len() > 0 {
		answer = streamed.String()
	}
	return o.finalizeQuery(ctx, result, o.guardOutput("/api/chat/query/stream#output", result.sessionID, result.request.UserID, answer))
}

type preparedQuery struct {
	sessionID     string
	request       QueryRequest
	prompt        PromptRequest
	citations     []retrieval.Citation
	toolTraces    []tools.Trace
	retrievalMeta retrieval.Metadata
	response      *QueryResponse
}

func (o *Orchestrator) prepareQuery(ctx context.Context, req QueryRequest) (preparedQuery, error) {
	sessionID, err := o.ensureSession(ctx, req)
	if err != nil {
		return preparedQuery{}, err
	}

	history, err := o.memory.Load(ctx, req.UserID, sessionID)
	if err != nil {
		return preparedQuery{}, err
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
		if _, err := o.persistRound(ctx, sessionID, req.Message, answer, nil, nil); err != nil {
			return preparedQuery{}, err
		}
		_, _ = o.memory.Update(ctx, memory.UpdateRequest{
			UserID:    req.UserID,
			SessionID: sessionID,
			Incoming: []memory.MessageMemory{
				{Role: "user", Content: req.Message},
				{Role: "assistant", Content: answer},
			},
		})
		response := QueryResponse{
			SessionID:     sessionID,
			Answer:        answer,
			Citations:     nil,
			ToolTraces:    []tools.Trace{trace},
			RetrievalMeta: retrievalResult.Meta,
		}
		return preparedQuery{
			sessionID: sessionID,
			request:   req,
			response:  &response,
		}, nil
	}

	return preparedQuery{
		sessionID: sessionID,
		request:   req,
		prompt: PromptRequest{
			UserID:     req.UserID,
			SessionID:  sessionID,
			Message:    req.Message,
			History:    history.Combined,
			Citations:  retrievalResult.Citations,
			Candidates: retrievalResult.Chunks,
		},
		citations:     retrievalResult.Citations,
		toolTraces:    []tools.Trace{trace},
		retrievalMeta: retrievalResult.Meta,
	}, nil
}

func (o *Orchestrator) finalizeQuery(ctx context.Context, result preparedQuery, answer string) (QueryResponse, error) {
	round, err := o.persistRound(ctx, result.sessionID, result.request.Message, answer, nil, nil)
	if err != nil {
		return QueryResponse{}, err
	}

	_, _ = o.memory.Update(ctx, memory.UpdateRequest{
		UserID:    result.request.UserID,
		SessionID: result.sessionID,
		Incoming: []memory.MessageMemory{
			{Role: "user", Content: result.request.Message},
			{Role: "assistant", Content: answer},
		},
	})

	toolTraces := append([]tools.Trace(nil), result.toolTraces...)
	if trace := o.maybeAutoWriteback(ctx, result, round, answer); trace != nil {
		toolTraces = append(toolTraces, *trace)
	}

	return QueryResponse{
		SessionID:     result.sessionID,
		Answer:        answer,
		Citations:     result.citations,
		ToolTraces:    toolTraces,
		RetrievalMeta: result.retrievalMeta,
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

func (o *Orchestrator) persistRound(ctx context.Context, sessionID, question, answer string, toolInput, toolOutput map[string]any) (persistedRound, error) {
	userMessage := chatdomain.Message{
		ID:        o.newID("msg"),
		SessionID: sessionID,
		Role:      "user",
		Content:   question,
		CreatedAt: o.now(),
	}
	if err := o.store.AppendMessage(ctx, userMessage); err != nil {
		return persistedRound{}, err
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
	if err := o.store.AppendMessage(ctx, assistantMessage); err != nil {
		return persistedRound{}, err
	}
	return persistedRound{
		UserMessage:      userMessage,
		AssistantMessage: assistantMessage,
	}, nil
}

func extractRetrievalResult(output tools.Output) retrieval.Result {
	result, ok := output.Data.(retrieval.Result)
	if ok {
		return result
	}
	return retrieval.Result{}
}

type outputBlockedError struct {
	err error
}

func (e outputBlockedError) Error() string {
	if e.err == nil {
		return "output blocked"
	}
	return e.err.Error()
}

func (o *Orchestrator) validateOutput(answer string) error {
	if o.outputGuardrail == nil {
		return nil
	}
	return o.outputGuardrail.ValidateOutput(answer)
}

func (o *Orchestrator) guardOutput(endpoint, sessionID, userID, answer string) string {
	if err := o.validateOutput(answer); err != nil {
		reason := "unsafe_output"
		if o.guardrailObserver != nil {
			o.guardrailObserver.RecordGuardrailReject(endpoint, reason)
		}
		if o.guardrailLogger != nil {
			o.guardrailLogger.Warn("guardrail rejected response",
				"path", endpoint,
				"user_id", userID,
				"session_id", sessionID,
				"reason", reason,
			)
		}
		return safeOutputAnswer()
	}
	return answer
}

func safeOutputAnswer() string {
	return "当前回答命中输出安全策略，已停止返回，请换个问法或补充更明确的资料。"
}
