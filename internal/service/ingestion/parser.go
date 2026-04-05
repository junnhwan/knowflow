package ingestion

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var blankLinePattern = regexp.MustCompile(`\n{3,}`)

type Parser struct{}

func NewParser() Parser {
	return Parser{}
}

func (Parser) Parse(sourceName, content string) (string, error) {
	ext := strings.ToLower(filepath.Ext(sourceName))
	if ext != ".md" && ext != ".txt" {
		return "", fmt.Errorf("unsupported file type: %s", ext)
	}

	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	normalized = blankLinePattern.ReplaceAllString(normalized, "\n\n")
	normalized = strings.TrimSpace(normalized)
	return normalized, nil
}
