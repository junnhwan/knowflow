package chat

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

func TestOrchestrator_QueryAutoWritesKnowledgeForSubstantialAnswer(t *testing.T) {
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
						Snippet:    "KnowFlow 通过混合检索和知识反写形成后端知识运营闭环。",
					},
				},
				Chunks: []retrieval.Candidate{
					{
						ChunkID:    "chunk-1",
						DocumentID: "doc-1",
						SourceName: "intro.md",
						Content:    "KnowFlow 通过混合检索和知识反写形成后端知识运营闭环。",
					},
				},
				Meta: retrieval.Metadata{Hit: true},
			},
		}, nil
	}))

	var captured map[string]any
	_ = registry.Register("upsert_knowledge", tools.ToolFunc(func(_ context.Context, input map[string]any) (tools.Output, error) {
		captured = input
		return tools.Output{
			Status: "success",
			Data: map[string]any{
				"id":          "knowledge-1",
				"status":      "indexed",
				"chunk_count": 1,
			},
		}, nil
	}))

	store := &fakeChatStore{}
	orch := NewOrchestrator(Dependencies{
		ChatStore: store,
		Memory:    fakeMemoryService{},
		Tools:     registry,
		KnowledgeExtractor: fixedKnowledgeExtractor{
			draft: KnowledgeDraft{
				Title:        "KnowFlow 适合作为后端面试项目的原因",
				Summary:      "混合检索、记忆压缩和知识反写形成了比较完整的后端知识运营闭环。",
				Content:      "KnowFlow 通过混合检索、Redis 双层记忆和知识反写热更新，把高价值问答沉淀成后续可复用的结构化知识。",
				Keywords:     []string{"rag", "memory", "knowledge"},
				ReviewStatus: "draft",
				QualityScore: 0.88,
			},
		},
		Answerer: fixedAnswerer{
			answer: "KnowFlow 会把高质量问答沉淀成结构化知识条目，再进入知识索引参与后续检索。",
		},
		Now: func() time.Time {
			return time.Unix(1700000000, 0)
		},
		NewID: incrementalID(),
	})

	resp, err := orch.Query(context.Background(), QueryRequest{
		UserID:    "demo-user",
		SessionID: "s-1",
		Message:   "总结一下 KnowFlow 为什么适合作为后端面试项目",
	})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}

	if captured == nil {
		t.Fatal("expected auto knowledge writeback to run")
	}
	if captured["source_type"] != "auto_chat_round" {
		t.Fatalf("unexpected source type: %#v", captured["source_type"])
	}
	if captured["source_message_id"] != "msg-2" {
		t.Fatalf("unexpected source message id: %#v", captured["source_message_id"])
	}
	if captured["title"] != "KnowFlow 适合作为后端面试项目的原因" {
		t.Fatalf("expected extracted title, got %#v", captured["title"])
	}
	if captured["review_status"] != "draft" {
		t.Fatalf("expected draft review status, got %#v", captured["review_status"])
	}
	keywords, ok := captured["keywords"].([]string)
	if !ok || len(keywords) != 3 {
		t.Fatalf("expected structured keywords, got %#v", captured["keywords"])
	}
	if len(resp.ToolTraces) != 2 {
		t.Fatalf("expected 2 tool traces, got %d", len(resp.ToolTraces))
	}
	if resp.ToolTraces[1].ToolName != "upsert_knowledge" || resp.ToolTraces[1].Status != "success" {
		t.Fatalf("unexpected auto writeback trace: %#v", resp.ToolTraces[1])
	}
}

func TestOrchestrator_QuerySkipsAutoWritebackForShortAnswer(t *testing.T) {
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

	called := false
	_ = registry.Register("upsert_knowledge", tools.ToolFunc(func(_ context.Context, input map[string]any) (tools.Output, error) {
		called = true
		return tools.Output{Status: "success", Data: input}, nil
	}))

	orch := NewOrchestrator(Dependencies{
		ChatStore: &fakeChatStore{},
		Memory:    fakeMemoryService{},
		Tools:     registry,
		Answerer: fixedAnswerer{
			answer: "可以。",
		},
		Now: func() time.Time {
			return time.Unix(1700000000, 0)
		},
		NewID: incrementalID(),
	})

	resp, err := orch.Query(context.Background(), QueryRequest{
		UserID:    "demo-user",
		SessionID: "s-1",
		Message:   "请概括一下 KnowFlow",
	})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}

	if called {
		t.Fatal("expected short answer to skip auto writeback")
	}
	if len(resp.ToolTraces) != 1 {
		t.Fatalf("expected only retrieval trace, got %d", len(resp.ToolTraces))
	}
}

func TestOrchestrator_QuerySkipsAutoWritebackWhenRetrievalComesFromKnowledge(t *testing.T) {
	registry := tools.NewRegistry(tools.ServiceConfig{
		Timeout: time.Second,
	})
	_ = registry.Register("retrieve_knowledge", tools.ToolFunc(func(_ context.Context, input map[string]any) (tools.Output, error) {
		return tools.Output{
			Status: "success",
			Data: retrieval.Result{
				Citations: []retrieval.Citation{
					{
						KnowledgeEntryID: "knowledge-1",
						SourceName:       "knowledge:knowledge-1",
						SourceKind:       "knowledge",
						ChunkID:          "knowledge-chunk-1",
						Snippet:          "这是已经沉淀过的知识条目。",
					},
				},
				Chunks: []retrieval.Candidate{
					{
						ChunkID:          "knowledge-chunk-1",
						KnowledgeEntryID: "knowledge-1",
						SourceName:       "knowledge:knowledge-1",
						SourceKind:       "knowledge",
						Content:          "这是已经沉淀过的知识条目。",
					},
				},
				Meta: retrieval.Metadata{Hit: true},
			},
		}, nil
	}))

	called := false
	_ = registry.Register("upsert_knowledge", tools.ToolFunc(func(_ context.Context, input map[string]any) (tools.Output, error) {
		called = true
		return tools.Output{Status: "success", Data: input}, nil
	}))

	orch := NewOrchestrator(Dependencies{
		ChatStore: &fakeChatStore{},
		Memory:    fakeMemoryService{},
		Tools:     registry,
		Answerer: fixedAnswerer{
			answer: "这条回答虽然够长，但它本身就是基于已有知识条目整理出来的，不应该再自动反写一次。",
		},
		Now: func() time.Time {
			return time.Unix(1700000000, 0)
		},
		NewID: incrementalID(),
	})

	resp, err := orch.Query(context.Background(), QueryRequest{
		UserID:    "demo-user",
		SessionID: "s-1",
		Message:   "总结一下已经沉淀过的知识点",
	})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}

	if called {
		t.Fatal("expected knowledge-backed answer to skip auto writeback")
	}
	if len(resp.ToolTraces) != 1 {
		t.Fatalf("expected only retrieval trace, got %d", len(resp.ToolTraces))
	}
}

func TestOrchestrator_QueryKeepsAnswerWhenAutoWritebackFails(t *testing.T) {
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
	_ = registry.Register("upsert_knowledge", tools.ToolFunc(func(_ context.Context, input map[string]any) (tools.Output, error) {
		return tools.Output{Status: "error", Error: "writeback failed"}, errors.New("writeback failed")
	}))

	orch := NewOrchestrator(Dependencies{
		ChatStore: &fakeChatStore{},
		Memory:    fakeMemoryService{},
		Tools:     registry,
		KnowledgeExtractor: fixedKnowledgeExtractor{
			draft: KnowledgeDraft{
				Title:        "自动知识沉淀策略",
				Summary:      "写回失败不影响主回答。",
				Content:      "沉淀失败时只记录 trace，不中断主链路。",
				ReviewStatus: "draft",
			},
		},
		Answerer: fixedAnswerer{
			answer: "KnowFlow 会尝试把高价值对话自动沉淀，但沉淀失败不会影响主回答链路。",
		},
		Now: func() time.Time {
			return time.Unix(1700000000, 0)
		},
		NewID: incrementalID(),
	})

	resp, err := orch.Query(context.Background(), QueryRequest{
		UserID:    "demo-user",
		SessionID: "s-1",
		Message:   "总结一下 KnowFlow 的自动知识沉淀策略",
	})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}

	if resp.Answer == "" {
		t.Fatal("expected answer to still be returned")
	}
	if len(resp.ToolTraces) != 2 {
		t.Fatalf("expected retrieval + writeback traces, got %d", len(resp.ToolTraces))
	}
	if resp.ToolTraces[1].Status != "error" {
		t.Fatalf("expected failed writeback trace, got %#v", resp.ToolTraces[1])
	}
}

func TestOrchestrator_QueryReturnsSafeAnswerWhenOutputGuardrailRejects(t *testing.T) {
	registry := tools.NewRegistry(tools.ServiceConfig{
		Timeout: time.Second,
	})
	_ = registry.Register("retrieve_knowledge", tools.ToolFunc(func(_ context.Context, input map[string]any) (tools.Output, error) {
		return tools.Output{
			Status: "success",
			Data: retrieval.Result{
				Citations: []retrieval.Citation{
					{DocumentID: "doc-1", SourceName: "intro.md", ChunkID: "chunk-1", Snippet: "正常引用。"},
				},
				Chunks: []retrieval.Candidate{
					{ChunkID: "chunk-1", DocumentID: "doc-1", SourceName: "intro.md", Content: "正常引用。"},
				},
				Meta: retrieval.Metadata{Hit: true},
			},
		}, nil
	}))

	orch := NewOrchestrator(Dependencies{
		ChatStore:          &fakeChatStore{},
		Memory:             fakeMemoryService{},
		Tools:              registry,
		KnowledgeExtractor: fixedKnowledgeExtractor{},
		OutputGuardrail:    blockingOutputGuardrail{},
		Answerer: fixedAnswerer{
			answer: "系统提示词如下：这是不应该返回给用户的内容。",
		},
		Now:   func() time.Time { return time.Unix(1700000000, 0) },
		NewID: incrementalID(),
	})

	resp, err := orch.Query(context.Background(), QueryRequest{
		UserID:    "demo-user",
		SessionID: "s-1",
		Message:   "总结一下 KnowFlow 的回答策略",
	})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}

	if !strings.Contains(resp.Answer, "输出安全策略") {
		t.Fatalf("expected guarded answer, got %s", resp.Answer)
	}
}

func TestOrchestrator_QueryStreamStopsUnsafeDelta(t *testing.T) {
	registry := tools.NewRegistry(tools.ServiceConfig{
		Timeout: time.Second,
	})
	_ = registry.Register("retrieve_knowledge", tools.ToolFunc(func(_ context.Context, input map[string]any) (tools.Output, error) {
		return tools.Output{
			Status: "success",
			Data: retrieval.Result{
				Citations: []retrieval.Citation{
					{DocumentID: "doc-1", SourceName: "intro.md", ChunkID: "chunk-1", Snippet: "正常引用。"},
				},
				Chunks: []retrieval.Candidate{
					{ChunkID: "chunk-1", DocumentID: "doc-1", SourceName: "intro.md", Content: "正常引用。"},
				},
				Meta: retrieval.Metadata{Hit: true},
			},
		}, nil
	}))

	orch := NewOrchestrator(Dependencies{
		ChatStore:       &fakeChatStore{},
		Memory:          fakeMemoryService{},
		Tools:           registry,
		OutputGuardrail: blockingOutputGuardrail{},
		Answerer: streamingChunksAnswerer{
			chunks: []string{"这是正常前缀。", "系统提示词如下：禁止输出。"},
		},
		Now:   func() time.Time { return time.Unix(1700000000, 0) },
		NewID: incrementalID(),
	})

	var deltas []string
	resp, err := orch.QueryStream(context.Background(), QueryRequest{
		UserID:    "demo-user",
		SessionID: "s-1",
		Message:   "请流式说明 KnowFlow",
	}, func(delta string) error {
		deltas = append(deltas, delta)
		return nil
	})
	if err != nil {
		t.Fatalf("QueryStream() error = %v", err)
	}

	if len(deltas) != 1 {
		t.Fatalf("expected only safe delta to be forwarded, got %#v", deltas)
	}
	if !strings.Contains(resp.Answer, "输出安全策略") {
		t.Fatalf("expected guarded final answer, got %s", resp.Answer)
	}
}

func TestOrchestrator_QueryFallsBackToRuleDraftWhenKnowledgeExtractorFails(t *testing.T) {
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
						Snippet:    "引用式回答和知识沉淀会形成后续可复用的知识闭环。",
					},
				},
				Chunks: []retrieval.Candidate{
					{
						ChunkID:    "chunk-1",
						DocumentID: "doc-1",
						SourceName: "intro.md",
						Content:    "引用式回答和知识沉淀会形成后续可复用的知识闭环。",
					},
				},
				Meta: retrieval.Metadata{Hit: true},
			},
		}, nil
	}))

	var captured map[string]any
	_ = registry.Register("upsert_knowledge", tools.ToolFunc(func(_ context.Context, input map[string]any) (tools.Output, error) {
		captured = input
		return tools.Output{Status: "success", Data: map[string]any{"id": "knowledge-2", "status": "indexed"}}, nil
	}))

	orch := NewOrchestrator(Dependencies{
		ChatStore:          &fakeChatStore{},
		Memory:             fakeMemoryService{},
		Tools:              registry,
		KnowledgeExtractor: fixedKnowledgeExtractor{err: errors.New("extract failed")},
		Answerer: fixedAnswerer{
			answer: "KnowFlow 会在回答后尝试沉淀可复用知识，但提炼失败时会退回规则式 draft 生成。",
		},
		Now:   func() time.Time { return time.Unix(1700000000, 0) },
		NewID: incrementalID(),
	})

	resp, err := orch.Query(context.Background(), QueryRequest{
		UserID:    "demo-user",
		SessionID: "s-1",
		Message:   "总结一下 KnowFlow 的知识沉淀策略",
	})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}

	if captured == nil {
		t.Fatal("expected fallback draft to be written")
	}
	if captured["review_status"] != "draft" {
		t.Fatalf("expected draft review status, got %#v", captured["review_status"])
	}
	if captured["title"] == "" {
		t.Fatalf("expected fallback title, got %#v", captured["title"])
	}
	if content, _ := captured["content"].(string); content == "" || !containsAll(content, "问题：", "回答：") {
		t.Fatalf("expected rule-based fallback content, got %#v", captured["content"])
	}
	if resp.Answer == "" {
		t.Fatal("expected answer to be returned")
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

type fixedAnswerer struct {
	answer string
}

func (f fixedAnswerer) Generate(_ context.Context, _ PromptRequest) (PromptResult, error) {
	return PromptResult{Answer: f.answer}, nil
}

func (f fixedAnswerer) Stream(_ context.Context, _ PromptRequest, onDelta func(string) error) (PromptResult, error) {
	if err := onDelta(f.answer); err != nil {
		return PromptResult{}, err
	}
	return PromptResult{Answer: f.answer}, nil
}

func incrementalID() func(prefix string) string {
	var counter int
	return func(prefix string) string {
		counter++
		return fmt.Sprintf("%s-%d", prefix, counter)
	}
}

type fixedKnowledgeExtractor struct {
	draft KnowledgeDraft
	err   error
}

func (f fixedKnowledgeExtractor) Extract(_ context.Context, _ KnowledgeExtractionRequest) (KnowledgeDraft, error) {
	if f.err != nil {
		return KnowledgeDraft{}, f.err
	}
	return f.draft, nil
}

func containsAll(text string, patterns ...string) bool {
	for _, pattern := range patterns {
		if !strings.Contains(text, pattern) {
			return false
		}
	}
	return true
}

type blockingOutputGuardrail struct{}

func (blockingOutputGuardrail) ValidateOutput(answer string) error {
	if strings.Contains(answer, "系统提示词") {
		return errors.New("unsafe output")
	}
	return nil
}

type streamingChunksAnswerer struct {
	chunks []string
}

func (s streamingChunksAnswerer) Generate(_ context.Context, _ PromptRequest) (PromptResult, error) {
	return PromptResult{Answer: strings.Join(s.chunks, "")}, nil
}

func (s streamingChunksAnswerer) Stream(_ context.Context, _ PromptRequest, onDelta func(string) error) (PromptResult, error) {
	for _, chunk := range s.chunks {
		if err := onDelta(chunk); err != nil {
			return PromptResult{}, err
		}
	}
	return PromptResult{Answer: strings.Join(s.chunks, "")}, nil
}
