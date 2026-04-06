package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}

type LocalHasherEmbedder struct {
	Dimension int
}

func (e LocalHasherEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	dimension := e.Dimension
	if dimension <= 0 {
		dimension = 64
	}

	out := make([][]float32, 0, len(texts))
	for _, text := range texts {
		vector := make([]float32, dimension)
		for _, token := range strings.Fields(strings.ToLower(text)) {
			index := hashToken(token) % uint32(dimension)
			vector[index] += 1
		}
		normalize(vector)
		out = append(out, vector)
	}
	return out, nil
}

type OpenAICompatibleEmbedder struct {
	BaseURL    string
	APIKey     string
	Model      string
	Dimensions int
	Client     *http.Client
}

func (e OpenAICompatibleEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	if strings.TrimSpace(e.BaseURL) == "" {
		return nil, fmt.Errorf("embedding base url is required")
	}
	if strings.TrimSpace(e.APIKey) == "" {
		return nil, fmt.Errorf("embedding api key is required")
	}

	body := map[string]any{
		"model":           e.Model,
		"input":           texts,
		"encoding_format": "float",
	}
	if e.Dimensions > 0 {
		body["dimensions"] = e.Dimensions
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, joinURL(e.BaseURL, "/embeddings"), strings.NewReader(string(payload)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+e.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := e.Client
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("embedding request failed with status %d", resp.StatusCode)
	}

	var result struct {
		Data []struct {
			Index     int       `json:"index"`
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	out := make([][]float32, len(result.Data))
	for _, item := range result.Data {
		if item.Index >= 0 && item.Index < len(out) {
			out[item.Index] = item.Embedding
		}
	}
	return out, nil
}

type FallbackEmbedder struct {
	Primary  Embedder
	Fallback Embedder
}

func (e FallbackEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if e.Primary != nil {
		vectors, err := e.Primary.Embed(ctx, texts)
		if err == nil && embeddingsReady(vectors, len(texts)) {
			return vectors, nil
		}
	}
	if e.Fallback == nil {
		return nil, fmt.Errorf("fallback embedder is required")
	}
	return e.Fallback.Embed(ctx, texts)
}

func hashToken(token string) uint32 {
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(token))
	return hasher.Sum32()
}

func normalize(vector []float32) {
	var sum float64
	for _, value := range vector {
		sum += float64(value * value)
	}
	if sum == 0 {
		return
	}
	length := float32(math.Sqrt(sum))
	for index := range vector {
		vector[index] /= length
	}
}

func joinURL(base, path string) string {
	parsed, err := url.Parse(strings.TrimSpace(base))
	if err != nil {
		return strings.TrimRight(base, "/") + path
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + path
	return parsed.String()
}

func embeddingsReady(vectors [][]float32, expected int) bool {
	if len(vectors) != expected {
		return false
	}
	for _, vector := range vectors {
		if len(vector) == 0 {
			return false
		}
	}
	return true
}
