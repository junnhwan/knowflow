package chat

import (
	"context"
	"fmt"
	"strings"

	"knowflow/internal/service/retrieval"
)

type KnowledgeExtractionRequest struct {
	UserID    string
	SessionID string
	Question  string
	Answer    string
	Citations []retrieval.Citation
}

type KnowledgeDraft struct {
	Title        string
	Summary      string
	Content      string
	Keywords     []string
	ReviewStatus string
	QualityScore float64
}

type RuleKnowledgeExtractor struct{}

type FallbackKnowledgeExtractor struct {
	Primary  KnowledgeExtractor
	Fallback KnowledgeExtractor
}

func defaultKnowledgeExtractor(extractor KnowledgeExtractor) KnowledgeExtractor {
	if extractor != nil {
		return extractor
	}
	return RuleKnowledgeExtractor{}
}

func (e FallbackKnowledgeExtractor) Extract(ctx context.Context, req KnowledgeExtractionRequest) (KnowledgeDraft, error) {
	if e.Primary != nil {
		draft, err := e.Primary.Extract(ctx, req)
		if err == nil && draft.Content != "" {
			return normalizeDraft(draft), nil
		}
	}
	if e.Fallback == nil {
		return KnowledgeDraft{}, fmt.Errorf("fallback knowledge extractor is required")
	}
	draft, err := e.Fallback.Extract(ctx, req)
	if err != nil {
		return KnowledgeDraft{}, err
	}
	return normalizeDraft(draft), nil
}

func (RuleKnowledgeExtractor) Extract(_ context.Context, req KnowledgeExtractionRequest) (KnowledgeDraft, error) {
	return normalizeDraft(buildFallbackKnowledgeDraft(req.Question, req.Answer, req.Citations)), nil
}

func buildFallbackKnowledgeDraft(question, answer string, citations []retrieval.Citation) KnowledgeDraft {
	title := deriveDraftTitle(question)
	summary := deriveDraftSummary(answer)
	content := buildAutoKnowledgeContent(question, answer, citations)
	keywords := deriveKeywords(question, answer, citations)
	return KnowledgeDraft{
		Title:        title,
		Summary:      summary,
		Content:      content,
		Keywords:     keywords,
		ReviewStatus: "draft",
		QualityScore: heuristicQualityScore(summary, content, keywords),
	}
}

func normalizeDraft(draft KnowledgeDraft) KnowledgeDraft {
	draft.Title = strings.TrimSpace(draft.Title)
	draft.Summary = strings.TrimSpace(draft.Summary)
	draft.Content = strings.TrimSpace(draft.Content)
	draft.Keywords = compactDraftKeywords(draft.Keywords)
	if draft.ReviewStatus == "" {
		draft.ReviewStatus = "draft"
	}
	if draft.QualityScore <= 0 {
		draft.QualityScore = heuristicQualityScore(draft.Summary, draft.Content, draft.Keywords)
	}
	if draft.Title == "" {
		draft.Title = deriveDraftTitle(draft.Summary)
	}
	if draft.Summary == "" {
		draft.Summary = deriveDraftSummary(draft.Content)
	}
	return draft
}

func deriveDraftTitle(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return "知识草稿"
	}
	runes := []rune(trimmed)
	if len(runes) > 20 {
		runes = runes[:20]
	}
	return string(runes)
}

func deriveDraftSummary(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ""
	}
	runes := []rune(trimmed)
	if len(runes) > 72 {
		runes = runes[:72]
	}
	return string(runes)
}

func deriveKeywords(question, answer string, citations []retrieval.Citation) []string {
	raw := []string{}
	for _, segment := range []string{question, answer} {
		raw = append(raw, splitChineseLikeKeywords(segment)...)
	}
	for _, citation := range citations {
		raw = append(raw, splitChineseLikeKeywords(citation.SourceName)...)
	}
	return compactDraftKeywords(raw)
}

func splitChineseLikeKeywords(text string) []string {
	normalized := strings.NewReplacer(
		"，", " ",
		"。", " ",
		"、", " ",
		"：", " ",
		":", " ",
		"（", " ",
		"）", " ",
		"(", " ",
		")", " ",
		"\n", " ",
		"\t", " ",
	).Replace(strings.ToLower(text))
	fields := strings.Fields(normalized)
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		if len([]rune(field)) < 2 {
			continue
		}
		out = append(out, field)
	}
	return out
}

func compactDraftKeywords(keywords []string) []string {
	if len(keywords) == 0 {
		return nil
	}
	out := make([]string, 0, len(keywords))
	seen := make(map[string]struct{}, len(keywords))
	for _, keyword := range keywords {
		trimmed := strings.TrimSpace(keyword)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
		if len(out) >= 8 {
			break
		}
	}
	return out
}

func heuristicQualityScore(summary, content string, keywords []string) float64 {
	score := 0.4
	if strings.TrimSpace(summary) != "" {
		score += 0.2
	}
	if len([]rune(strings.TrimSpace(content))) >= 80 {
		score += 0.2
	}
	if len(keywords) >= 3 {
		score += 0.2
	}
	if score > 1 {
		return 1
	}
	return score
}
