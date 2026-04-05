package retrieval

import (
	"regexp"
	"strings"
)

var punctuationPattern = regexp.MustCompile(`[[:punct:]]+`)

type ProcessedQuery struct {
	Normalized string
	Tokens     []string
}

type Preprocessor struct {
	stopWords map[string]struct{}
}

func NewPreprocessor() Preprocessor {
	stopWords := []string{"的", "了", "是", "在", "我", "有", "和", "就", "不", "吗", "呢", "吧", "啊"}
	set := make(map[string]struct{}, len(stopWords))
	for _, word := range stopWords {
		set[word] = struct{}{}
	}
	return Preprocessor{stopWords: set}
}

func (p Preprocessor) Process(query string) ProcessedQuery {
	normalized := punctuationPattern.ReplaceAllString(query, " ")
	normalized = strings.ReplaceAll(normalized, "\u3000", " ")
	normalized = strings.TrimSpace(strings.Join(strings.Fields(normalized), " "))

	rawTokens := strings.Fields(strings.ToLower(normalized))
	tokens := make([]string, 0, len(rawTokens))
	for _, token := range rawTokens {
		if _, blocked := p.stopWords[token]; blocked {
			continue
		}
		tokens = append(tokens, token)
	}

	if normalized == "" {
		normalized = query
	}

	return ProcessedQuery{
		Normalized: normalized,
		Tokens:     tokens,
	}
}
