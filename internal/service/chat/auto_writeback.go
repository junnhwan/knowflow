package chat

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	chatdomain "knowflow/internal/domain/chat"
	"knowflow/internal/service/retrieval"
	"knowflow/internal/service/tools"
)

type AutoKnowledgeConfig struct {
	Enabled          bool
	MinQuestionRunes int
	MinAnswerRunes   int
}

func defaultAutoKnowledgeConfig(cfg AutoKnowledgeConfig) AutoKnowledgeConfig {
	if !cfg.Enabled {
		cfg.Enabled = true
	}
	if cfg.MinQuestionRunes <= 0 {
		cfg.MinQuestionRunes = 8
	}
	if cfg.MinAnswerRunes <= 0 {
		cfg.MinAnswerRunes = 24
	}
	return cfg
}

func shouldAutoWriteback(cfg AutoKnowledgeConfig, question, answer string, citations []retrieval.Citation, meta retrieval.Metadata) bool {
	if !cfg.Enabled {
		return false
	}
	if !meta.Hit || len(citations) == 0 {
		return false
	}
	if utf8.RuneCountInString(strings.TrimSpace(question)) < cfg.MinQuestionRunes {
		return false
	}
	if utf8.RuneCountInString(strings.TrimSpace(answer)) < cfg.MinAnswerRunes {
		return false
	}
	return true
}

func buildAutoKnowledgeContent(question, answer string, citations []retrieval.Citation) string {
	var builder strings.Builder
	builder.WriteString("问题：")
	builder.WriteString(strings.TrimSpace(question))
	builder.WriteString("\n\n回答：")
	builder.WriteString(strings.TrimSpace(answer))

	if len(citations) > 0 {
		builder.WriteString("\n\n引用来源：")
		for _, citation := range citations[:minInt(3, len(citations))] {
			builder.WriteString("\n- ")
			builder.WriteString(citation.SourceName)
		}
	}
	return builder.String()
}

func deriveDocumentID(citations []retrieval.Citation) string {
	if len(citations) == 0 {
		return ""
	}
	first := citations[0].DocumentID
	if first == "" {
		return ""
	}
	for _, citation := range citations[1:] {
		if citation.DocumentID != first {
			return ""
		}
	}
	return first
}

func buildAutoWritebackTrace(toolName string, startedAt time.Time, err error) tools.Trace {
	trace := tools.Trace{
		ToolName:   toolName,
		DurationMs: time.Since(startedAt).Milliseconds(),
	}
	if err != nil {
		trace.Status = "error"
		trace.Error = err.Error()
		return trace
	}
	trace.Status = "success"
	return trace
}

func (o *Orchestrator) maybeAutoWriteback(ctx context.Context, result preparedQuery, round persistedRound, answer string) *tools.Trace {
	if !shouldAutoWriteback(o.autoKnowledge, result.request.Message, answer, result.citations, result.retrievalMeta) {
		return nil
	}

	startedAt := time.Now()
	_, err := o.tools.Execute(ctx, "upsert_knowledge", map[string]any{
		"user_id":           result.request.UserID,
		"session_id":        result.sessionID,
		"source_message_id": round.AssistantMessage.ID,
		"document_id":       deriveDocumentID(result.citations),
		"source_type":       "auto_chat_round",
		"content":           buildAutoKnowledgeContent(result.request.Message, answer, result.citations),
	})
	trace := buildAutoWritebackTrace("upsert_knowledge", startedAt, err)
	return &trace
}

type persistedRound struct {
	UserMessage      chatdomain.Message
	AssistantMessage chatdomain.Message
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
