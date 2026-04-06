package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type TextGenerator interface {
	GenerateText(ctx context.Context, messages []ChatMessage) (string, error)
}

type OpenAICompatibleTextGenerator struct {
	BaseURL     string
	APIKey      string
	Model       string
	Temperature float64
	Client      *http.Client
}

func (g OpenAICompatibleTextGenerator) GenerateText(ctx context.Context, messages []ChatMessage) (string, error) {
	if strings.TrimSpace(g.BaseURL) == "" {
		return "", fmt.Errorf("chat base url is required")
	}
	if strings.TrimSpace(g.APIKey) == "" {
		return "", fmt.Errorf("chat api key is required")
	}

	payload, err := json.Marshal(map[string]any{
		"model":       g.Model,
		"messages":    messages,
		"temperature": g.Temperature,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, joinURL(g.BaseURL, "/chat/completions"), bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+g.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := g.Client
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return "", fmt.Errorf("chat request failed with status %d", resp.StatusCode)
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("chat response choices is empty")
	}
	return strings.TrimSpace(result.Choices[0].Message.Content), nil
}
