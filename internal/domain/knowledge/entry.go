package knowledge

import "time"

type Entry struct {
	ID              string
	UserID          string
	SessionID       string
	SourceMessageID string
	DocumentID      string
	SourceType      string
	Title           string
	Summary         string
	Content         string
	Keywords        []string
	Status          string
	ReviewStatus    string
	QualityScore    float64
	DedupeHash      string
	MergedIntoID    string
	DisabledAt      *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
