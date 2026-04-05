package chat

import (
	"context"
	"fmt"
	"testing"
	"time"

	chatdomain "knowflow/internal/domain/chat"
	"knowflow/internal/service/memory"
	"knowflow/internal/service/retrieval"
	"knowflow/internal/service/tools"
)

func TestOrchestrator_QueryReturnsAnswerAndCitations(t *testing.T) {
	registry := tools.NewRegistry(tools.ServiceConfig{
		Timeout: time.Second,
	})
	_ = registry.Register("retrieve_knowledge", tools.ToolFunc(func(_ context.Context, input map[string]any) (tools.Output, error) {
		return tools.Output{
			Status: "success",
			Data: retrieval.Result{
				Citations: []retrieval.Citation{
					{
						DocumentID: "doc-1",
						SourceName: "intro.md",
						ChunkID:    "chunk-1",
						Snippet:    "KnowFlow keeps citations.",
					},
				},
				Chunks: []retrieval.Candidate{
					{
						ChunkID:    "chunk-1",
						DocumentID: "doc-1",
						SourceName: "intro.md",
						Content:    "KnowFlow keeps citations.",
					},
				},
				Meta: retrieval.Metadata{Hit: true},
			},
			Meta: input,
		}, nil
	}))

	orch := NewOrchestrator(Dependencies{
		ChatStore: &fakeChatStore{},
		Memory:    fakeMemoryService{},
		Tools:     registry,
		Answerer:  fakeAnswerer{},
		Now: func() time.Time {
			return time.Unix(1700000000, 0)
		},
		NewID: func(prefix string) string {
			return prefix + "-1"
		},
	})

	resp, err := orch.Query(context.Background(), QueryRequest{
		UserID:    "demo-user",
		SessionID: "s-1",
		Message:   "总结一下 KnowFlow 的亮点",
	})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}

	if resp.Answer == "" {
		t.Fatalf("expected answer")
	}

	if len(resp.Citations) == 0 {
		t.Fatalf("expected citations")
	}
}

func TestOrchestrator_QueryStreamUsesAnswererStreamAndPersistsFinalAnswer(t *testing.T) {
	registry := tools.NewRegistry(tools.ServiceConfig{
		Timeout: time.Second,
	})
	_ = registry.Register("retrieve_knowledge", tools.ToolFunc(func(_ context.Context, input map[string]any) (tools.Output, error) {
		return tools.Output{
			Status: "success",
			Data: retrieval.Result{
				Citations: []retrieval.Citation{
					{
						DocumentID: "doc-1",
						SourceName: "intro.md",
						ChunkID:    "chunk-1",
						Snippet:    "KnowFlow keeps citations.",
					},
				},
				Chunks: []retrieval.Candidate{
					{
						ChunkID:    "chunk-1",
						DocumentID: "doc-1",
						SourceName: "intro.md",
						Content:    "KnowFlow keeps citations.",
					},
				},
				Meta: retrieval.Metadata{Hit: true},
			},
		}, nil
	}))

	store := &fakeChatStore{}
	answerer := &streamingFakeAnswerer{}
	orch := NewOrchestrator(Dependencies{
		ChatStore: store,
		Memory:    fakeMemoryService{},
		Tools:     registry,
		Answerer:  answerer,
		Now: func() time.Time {
			return time.Unix(1700000000, 0)
		},
		NewID: func(prefix string) string {
			return prefix + "-1"
		},
	})

	var deltas []string
	resp, err := orch.QueryStream(context.Background(), QueryRequest{
		UserID:    "demo-user",
		SessionID: "s-1",
		Message:   "请流式回答 KnowFlow 的亮点",
	}, func(delta string) error {
		deltas = append(deltas, delta)
		return nil
	})
	if err != nil {
		t.Fatalf("QueryStream() error = %v", err)
	}

	if !answerer.streamCalled {
		t.Fatal("expected answerer stream to be called")
	}

	if len(deltas) != 2 {
		t.Fatalf("expected 2 deltas, got %d", len(deltas))
	}

	if resp.Answer != "第一段第二段" {
		t.Fatalf("unexpected final answer: %s", resp.Answer)
	}

	if len(store.messages) != 2 {
		t.Fatalf("expected persisted user and assistant messages")
	}

	if store.messages[1].Content != "第一段第二段" {
		t.Fatalf("unexpected persisted assistant answer: %s", store.messages[1].Content)
	}
}

type fakeChatStore struct {
	sessions map[string]chatdomain.Session
	messages []chatdomain.Message
}

func (f *fakeChatStore) CreateSession(_ context.Context, session chatdomain.Session) error {
	if f.sessions == nil {
		f.sessions = map[string]chatdomain.Session{}
	}
	f.sessions[session.ID] = session
	return nil
}

func (f *fakeChatStore) GetSession(_ context.Context, sessionID string) (chatdomain.Session, error) {
	if f.sessions == nil {
		return chatdomain.Session{}, ErrSessionNotFound
	}
	session, ok := f.sessions[sessionID]
	if !ok {
		return chatdomain.Session{}, ErrSessionNotFound
	}
	return session, nil
}

func (f *fakeChatStore) ListSessions(_ context.Context, _ string) ([]chatdomain.Session, error) {
	out := make([]chatdomain.Session, 0, len(f.sessions))
	for _, session := range f.sessions {
		out = append(out, session)
	}
	return out, nil
}

func (f *fakeChatStore) AppendMessage(_ context.Context, message chatdomain.Message) error {
	f.messages = append(f.messages, message)
	return nil
}

func (f *fakeChatStore) ListMessages(_ context.Context, sessionID string) ([]chatdomain.Message, error) {
	out := make([]chatdomain.Message, 0)
	for _, message := range f.messages {
		if message.SessionID == sessionID {
			out = append(out, message)
		}
	}
	return out, nil
}

type fakeMemoryService struct{}

func (fakeMemoryService) Load(_ context.Context, _, _ string) (memory.LoadResult, error) {
	return memory.LoadResult{}, nil
}

func (fakeMemoryService) Update(_ context.Context, req memory.UpdateRequest) (memory.UpdateResult, error) {
	return memory.UpdateResult{
		Recent: req.Incoming,
	}, nil
}

type fakeAnswerer struct{}

func (fakeAnswerer) Generate(_ context.Context, req PromptRequest) (PromptResult, error) {
	return PromptResult{
		Answer: fmt.Sprintf("基于 %d 条引用整理的回答", len(req.Citations)),
	}, nil
}

func (fakeAnswerer) Stream(ctx context.Context, req PromptRequest, onDelta func(string) error) (PromptResult, error) {
	result, err := fakeAnswerer{}.Generate(ctx, req)
	if err != nil {
		return PromptResult{}, err
	}
	if err := onDelta(result.Answer); err != nil {
		return PromptResult{}, err
	}
	return result, nil
}

type streamingFakeAnswerer struct {
	streamCalled bool
}

func (*streamingFakeAnswerer) Generate(_ context.Context, _ PromptRequest) (PromptResult, error) {
	return PromptResult{Answer: "不应走到同步回答"}, nil
}

func (f *streamingFakeAnswerer) Stream(_ context.Context, _ PromptRequest, onDelta func(string) error) (PromptResult, error) {
	f.streamCalled = true
	if err := onDelta("第一段"); err != nil {
		return PromptResult{}, err
	}
	if err := onDelta("第二段"); err != nil {
		return PromptResult{}, err
	}
	return PromptResult{Answer: "第一段第二段"}, nil
}
