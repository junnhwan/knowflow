package memory

import (
	"context"
	"fmt"
	"strings"

	"knowflow/internal/platform/llm"
)

type FallbackSummaryGenerator struct {
	Primary  SummaryGenerator
	Fallback SummaryGenerator
}

func (g FallbackSummaryGenerator) Summarize(ctx context.Context, messages []MessageMemory, tokenLimit int) (string, error) {
	if g.Primary != nil {
		summary, err := g.Primary.Summarize(ctx, messages, tokenLimit)
		if err == nil && strings.TrimSpace(summary) != "" {
			return strings.TrimSpace(summary), nil
		}
	}
	if g.Fallback == nil {
		return "", fmt.Errorf("fallback summary generator is required")
	}
	return g.Fallback.Summarize(ctx, messages, tokenLimit)
}

type LLMSummaryGenerator struct {
	Generator llm.TextGenerator
}

func (g LLMSummaryGenerator) Summarize(ctx context.Context, messages []MessageMemory, tokenLimit int) (string, error) {
	if g.Generator == nil {
		return "", fmt.Errorf("text generator is required")
	}
	text, err := g.Generator.GenerateText(ctx, []llm.ChatMessage{
		{
			Role:    "system",
			Content: "你是 KnowFlow 的对话摘要器。请用中文输出一段简洁摘要，保留用户目标、关键结论和未解决问题。",
		},
		{
			Role:    "user",
			Content: buildSummaryPrompt(messages, tokenLimit),
		},
	})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(text), nil
}

func buildSummaryPrompt(messages []MessageMemory, tokenLimit int) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("请把下面的多轮对话压缩成不超过 %d tokens 的摘要：\n", tokenLimit))
	for _, message := range messages {
		builder.WriteString(message.Role)
		builder.WriteString(": ")
		builder.WriteString(message.Content)
		builder.WriteString("\n")
	}
	return builder.String()
}
