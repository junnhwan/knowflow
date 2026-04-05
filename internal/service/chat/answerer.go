package chat

import (
	"context"
	"fmt"
	"strings"
	"time"

	"knowflow/internal/service/retrieval"
)

type LocalAnswerer struct{}

func NewLocalAnswerer() LocalAnswerer {
	return LocalAnswerer{}
}

type LLMTelemetry interface {
	RecordLLMRequest(provider string)
	RecordLLMLatency(provider string, duration time.Duration)
}

func (LocalAnswerer) Generate(_ context.Context, req PromptRequest) (PromptResult, error) {
	if len(req.Citations) == 0 {
		return PromptResult{
			Answer: "当前没有足够证据支持回答，请先补充相关知识资料。",
		}, nil
	}

	return PromptResult{Answer: buildGroundedAnswer(req.Citations)}, nil
}

func (a LocalAnswerer) Stream(ctx context.Context, req PromptRequest, onDelta func(string) error) (PromptResult, error) {
	result, err := a.Generate(ctx, req)
	if err != nil {
		return PromptResult{}, err
	}

	runes := []rune(result.Answer)
	const chunkSize = 24
	for start := 0; start < len(runes); start += chunkSize {
		end := start + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		if err := onDelta(string(runes[start:end])); err != nil {
			return PromptResult{}, err
		}
	}
	return result, nil
}

func buildGroundedAnswer(citations []retrieval.Citation) string {
	var builder strings.Builder
	builder.WriteString("基于当前知识库资料，我整理出以下结论：\n")
	for index, citation := range citations {
		builder.WriteString(fmt.Sprintf("%d. %s\n", index+1, citation.Snippet))
	}
	builder.WriteString("\n可继续追问具体实现细节、取舍原因或相关面试追问。")
	return builder.String()
}
