package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"knowflow/internal/platform/llm"
)

type LLMKnowledgeExtractor struct {
	Generator llm.TextGenerator
}

func (e LLMKnowledgeExtractor) Extract(ctx context.Context, req KnowledgeExtractionRequest) (KnowledgeDraft, error) {
	if e.Generator == nil {
		return KnowledgeDraft{}, fmt.Errorf("text generator is required")
	}
	text, err := e.Generator.GenerateText(ctx, []llm.ChatMessage{
		{
			Role:    "system",
			Content: "你是 KnowFlow 的知识提炼器。请把问答和引用资料提炼成可复用的后端面试知识草稿，只输出 JSON，不要输出额外解释。",
		},
		{
			Role:    "user",
			Content: buildKnowledgeExtractionPrompt(req),
		},
	})
	if err != nil {
		return KnowledgeDraft{}, err
	}

	payload, err := extractJSONObject(text)
	if err != nil {
		return KnowledgeDraft{}, err
	}

	var draft struct {
		Title        string   `json:"title"`
		Summary      string   `json:"summary"`
		Content      string   `json:"content"`
		Keywords     []string `json:"keywords"`
		ReviewStatus string   `json:"review_status"`
		QualityScore float64  `json:"quality_score"`
	}
	if err := json.Unmarshal([]byte(payload), &draft); err != nil {
		return KnowledgeDraft{}, err
	}

	return normalizeDraft(KnowledgeDraft{
		Title:        draft.Title,
		Summary:      draft.Summary,
		Content:      draft.Content,
		Keywords:     draft.Keywords,
		ReviewStatus: draft.ReviewStatus,
		QualityScore: draft.QualityScore,
	}), nil
}

func buildKnowledgeExtractionPrompt(req KnowledgeExtractionRequest) string {
	var citations strings.Builder
	for index, citation := range req.Citations {
		citations.WriteString(fmt.Sprintf("[%d] 来源=%s 片段=%s\n", index+1, citation.SourceName, citation.Snippet))
	}

	return fmt.Sprintf(`请基于下面的问答和引用资料，输出一个 JSON 对象，字段严格固定为：
title, summary, content, keywords, review_status, quality_score

要求：
1. review_status 固定输出 "draft"
2. quality_score 输出 0 到 1 的小数
3. keywords 输出 3 到 8 个中文或英文关键词
4. content 输出可复用的知识正文，不要保留“你”“我”这类对话口吻
5. 不要输出 markdown，不要输出代码块

问题：
%s

回答：
%s

引用资料：
%s`, strings.TrimSpace(req.Question), strings.TrimSpace(req.Answer), strings.TrimSpace(citations.String()))
}

func extractJSONObject(text string) (string, error) {
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start < 0 || end < 0 || end <= start {
		return "", fmt.Errorf("json object not found")
	}
	return text[start : end+1], nil
}
