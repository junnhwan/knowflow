package knowledge

import (
	"fmt"
	"hash/fnv"
	"strings"
)

func BuildDedupeHash(title, summary, content string) string {
	normalized := normalizeDedupeText(title + "\n" + summary + "\n" + content)
	if normalized == "" {
		return ""
	}
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(normalized))
	return fmt.Sprintf("%x", hasher.Sum64())
}

func BuildQualityScore(summary, content string, keywords []string) float64 {
	score := 0.35
	if strings.TrimSpace(summary) != "" {
		score += 0.2
	}
	if len([]rune(strings.TrimSpace(content))) >= 80 {
		score += 0.25
	}
	if len(NormalizeKeywords(keywords)) >= 3 {
		score += 0.2
	}
	if score > 1 {
		return 1
	}
	return score
}

func NormalizeKeywords(keywords []string) []string {
	if len(keywords) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(keywords))
	out := make([]string, 0, len(keywords))
	for _, keyword := range keywords {
		trimmed := strings.TrimSpace(keyword)
		if trimmed == "" {
			continue
		}
		lower := strings.ToLower(trimmed)
		if _, exists := seen[lower]; exists {
			continue
		}
		seen[lower] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func MergeKeywords(left, right []string) []string {
	return NormalizeKeywords(append(append([]string(nil), left...), right...))
}

func JoinUniqueText(parts ...string) string {
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return strings.Join(out, "\n\n")
}

func normalizeDedupeText(text string) string {
	replacer := strings.NewReplacer(
		"，", " ",
		"。", " ",
		"、", " ",
		"：", " ",
		":", " ",
		"\n", " ",
		"\t", " ",
	)
	return strings.Join(strings.Fields(strings.ToLower(replacer.Replace(strings.TrimSpace(text)))), " ")
}
