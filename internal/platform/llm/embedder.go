package llm

import (
	"context"
	"hash/fnv"
	"math"
	"strings"
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
