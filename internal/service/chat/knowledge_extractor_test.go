package chat

import (
	"context"
	"errors"
	"testing"

	"knowflow/internal/platform/llm"
	"knowflow/internal/service/retrieval"
)

func TestLLMKnowledgeExtractor_ExtractParsesStructuredDraft(t *testing.T) {
	extractor := LLMKnowledgeExtractor{
		Generator: fakeTextGenerator{
			text: "这里是解释```json\n{\"title\":\"Redis 双层记忆\",\"summary\":\"最近窗口配合历史摘要压缩。\",\"content\":\"Redis 双层记忆会保留最近多轮上下文，并在阈值触发后将更早消息压缩为摘要。\",\"keywords\":[\"redis\",\"memory\",\"summary\"],\"review_status\":\"draft\",\"quality_score\":0.93}\n```",
		},
	}

	draft, err := extractor.Extract(context.Background(), KnowledgeExtractionRequest{
		UserID:   "demo-user",
		Question: "总结一下 Redis 双层记忆的亮点",
		Answer:   "Redis 双层记忆会保留最近多轮上下文，并在阈值触发后将更早消息压缩为摘要。",
		Citations: []retrieval.Citation{
			{SourceName: "memory.md", Snippet: "最近窗口 + 历史摘要。"},
		},
	})
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	if draft.Title != "Redis 双层记忆" {
		t.Fatalf("unexpected title: %s", draft.Title)
	}
	if draft.ReviewStatus != "draft" {
		t.Fatalf("unexpected review status: %s", draft.ReviewStatus)
	}
	if len(draft.Keywords) != 3 {
		t.Fatalf("unexpected keywords: %#v", draft.Keywords)
	}
	if draft.QualityScore != 0.93 {
		t.Fatalf("unexpected quality score: %v", draft.QualityScore)
	}
}

func TestFallbackKnowledgeExtractor_UsesFallbackWhenPrimaryFails(t *testing.T) {
	extractor := FallbackKnowledgeExtractor{
		Primary: failingKnowledgeExtractor{err: errors.New("llm timeout")},
		Fallback: fixedKnowledgeExtractor{
			draft: KnowledgeDraft{
				Title:        "回退知识草稿",
				Summary:      "回退策略生效。",
				Content:      "当远程 LLM 提炼失败时，系统会退回规则式知识草稿生成。",
				ReviewStatus: "draft",
			},
		},
	}

	draft, err := extractor.Extract(context.Background(), KnowledgeExtractionRequest{})
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	if draft.Title != "回退知识草稿" {
		t.Fatalf("unexpected fallback title: %s", draft.Title)
	}
}

type fakeTextGenerator struct {
	text string
	err  error
}

func (f fakeTextGenerator) GenerateText(_ context.Context, _ []llm.ChatMessage) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.text, nil
}

type failingKnowledgeExtractor struct {
	err error
}

func (f failingKnowledgeExtractor) Extract(_ context.Context, _ KnowledgeExtractionRequest) (KnowledgeDraft, error) {
	return KnowledgeDraft{}, f.err
}
