package memory

import (
	"context"
	"fmt"
	"strings"
)

type MessageMemory struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type SummaryGenerator interface {
	Summarize(ctx context.Context, messages []MessageMemory, tokenLimit int) (string, error)
}

type CompressorConfig struct {
	RecentRounds    int
	TokenThreshold  int
	SummaryTokenCap int
}

type CompressionResult struct {
	Summary    string
	Recent     []MessageMemory
	Compressed bool
}

type Compressor struct {
	summaryGenerator SummaryGenerator
	config           CompressorConfig
}

func NewCompressor(summaryGenerator SummaryGenerator, cfg CompressorConfig) *Compressor {
	if cfg.RecentRounds <= 0 {
		cfg.RecentRounds = 5
	}
	if cfg.TokenThreshold <= 0 {
		cfg.TokenThreshold = 6000
	}
	if cfg.SummaryTokenCap <= 0 {
		cfg.SummaryTokenCap = 256
	}
	return &Compressor{
		summaryGenerator: summaryGenerator,
		config:           cfg,
	}
}

func (c *Compressor) ShouldCompress(messages []MessageMemory) bool {
	return estimateMessagesTokens(messages) > c.config.TokenThreshold || len(messages) > c.config.RecentRounds*4
}

func (c *Compressor) Compress(ctx context.Context, messages []MessageMemory) (CompressionResult, error) {
	if !c.ShouldCompress(messages) {
		return CompressionResult{
			Recent:     messages,
			Compressed: false,
		}, nil
	}

	recentCount := c.config.RecentRounds * 2
	if recentCount > len(messages) {
		recentCount = len(messages)
	}
	historyCount := len(messages) - recentCount
	history := messages[:historyCount]
	recent := messages[historyCount:]

	summary, err := c.summaryGenerator.Summarize(ctx, history, c.config.SummaryTokenCap)
	if err != nil || strings.TrimSpace(summary) == "" {
		summary = heuristicSummary(history)
	}

	return CompressionResult{
		Summary:    summary,
		Recent:     recent,
		Compressed: true,
	}, nil
}

func estimateMessagesTokens(messages []MessageMemory) int {
	total := 0
	for _, message := range messages {
		total += len([]rune(message.Content)) / 4
	}
	return total
}

func heuristicSummary(messages []MessageMemory) string {
	if len(messages) == 0 {
		return ""
	}
	parts := make([]string, 0, min(3, len(messages)))
	for _, message := range messages[:min(3, len(messages))] {
		content := message.Content
		if len([]rune(content)) > 48 {
			content = string([]rune(content)[:48])
		}
		parts = append(parts, fmt.Sprintf("%s:%s", message.Role, content))
	}
	return strings.Join(parts, " | ")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
