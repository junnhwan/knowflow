package knowledge

import "time"

type Chunk struct {
	ID               string
	KnowledgeEntryID string
	UserID           string
	ChunkIndex       int
	Content          string
	Embedding        []float32
	TokenCount       int
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
