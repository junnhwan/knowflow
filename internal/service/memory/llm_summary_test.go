package memory

import (
	"context"
	"errors"
	"testing"

	"knowflow/internal/platform/llm"
)

func TestLLMSummaryGenerator_SummarizeUsesRemoteGenerator(t *testing.T) {
	generator := LLMSummaryGenerator{
		Generator: fakeMemoryTextGenerator{
			text: "这段摘要保留了 Redis 双层记忆和历史压缩的关键结论。",
		},
	}

	summary, err := generator.Summarize(context.Background(), []MessageMemory{
		{Role: "user", Content: "请解释一下 Redis 双层记忆"},
		{Role: "assistant", Content: "最近窗口和历史摘要会一起工作。"},
	}, 128)
	if err != nil {
		t.Fatalf("Summarize() error = %v", err)
	}
	if summary == "" {
		t.Fatal("expected non-empty summary")
	}
}

func TestFallbackSummaryGenerator_UsesFallbackWhenPrimaryFails(t *testing.T) {
	generator := FallbackSummaryGenerator{
		Primary:  failingSummaryGenerator{err: errors.New("timeout")},
		Fallback: HeuristicSummaryGenerator{},
	}

	summary, err := generator.Summarize(context.Background(), []MessageMemory{
		{Role: "user", Content: "问题一"},
		{Role: "assistant", Content: "回答一"},
	}, 64)
	if err != nil {
		t.Fatalf("Summarize() error = %v", err)
	}
	if summary == "" {
		t.Fatal("expected fallback summary")
	}
}

type fakeMemoryTextGenerator struct {
	text string
	err  error
}

func (f fakeMemoryTextGenerator) GenerateText(_ context.Context, _ []llm.ChatMessage) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.text, nil
}

type failingSummaryGenerator struct {
	err error
}

func (f failingSummaryGenerator) Summarize(_ context.Context, _ []MessageMemory, _ int) (string, error) {
	return "", f.err
}
