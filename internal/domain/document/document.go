package document

import "time"

type Document struct {
	ID          string
	UserID      string
	SourceName  string
	Status      string
	ContentHash string
	RawContent  string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Chunk struct {
	ID         string
	DocumentID string
	UserID     string
	SourceName string
	ChunkIndex int
	Content    string
	Embedding  []float32
	TokenCount int
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
