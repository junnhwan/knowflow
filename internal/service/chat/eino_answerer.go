package chat

import (
	"context"
	"fmt"
	"io"
	"strings"

	openai "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

type EinoAnswerer struct {
	llm model.ToolCallingChatModel
}

func NewEinoAnswerer(ctx context.Context, baseURL, apiKey, modelName string) (*EinoAnswerer, error) {
	llmModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Model:   modelName,
	})
	if err != nil {
		return nil, fmt.Errorf("create eino chat model: %w", err)
	}
	return &EinoAnswerer{llm: llmModel}, nil
}

func (a *EinoAnswerer) Generate(ctx context.Context, req PromptRequest) (PromptResult, error) {
	messages := groundedMessages(req)
	resp, err := a.llm.Generate(ctx, messages)
	if err != nil {
		return PromptResult{}, err
	}
	return PromptResult{Answer: resp.Content}, nil
}

func (a *EinoAnswerer) Stream(ctx context.Context, req PromptRequest, onDelta func(string) error) (PromptResult, error) {
	messages := groundedMessages(req)
	stream, err := a.llm.Stream(ctx, messages)
	if err != nil {
		return PromptResult{}, err
	}
	defer stream.Close()

	var builder strings.Builder
	for {
		message, recvErr := stream.Recv()
		if recvErr == io.EOF {
			break
		}
		if recvErr != nil {
			return PromptResult{}, recvErr
		}
		if message.Content == "" {
			continue
		}
		builder.WriteString(message.Content)
		if err := onDelta(message.Content); err != nil {
			return PromptResult{}, err
		}
	}
	return PromptResult{Answer: builder.String()}, nil
}

func groundedMessages(req PromptRequest) []*schema.Message {
	var citations strings.Builder
	for index, citation := range req.Citations {
		citations.WriteString(fmt.Sprintf("[%d] %s (%s)\n", index+1, citation.Snippet, citation.SourceName))
	}

	prompt := fmt.Sprintf(`你是 KnowFlow 的问答助手。只能依据给定资料回答，不要编造。

用户问题：
%s

检索到的资料：
%s

请输出一段简洁、结构化的中文回答，并尽量引用资料要点。`, req.Message, citations.String())

	return []*schema.Message{
		{
			Role:    schema.System,
			Content: "你是一个面向后端面试知识运营场景的问答助手。",
		},
		{
			Role:    schema.User,
			Content: prompt,
		},
	}
}
