package knowledge

import "time"

type Entry struct {
	ID              string
	UserID          string
	SessionID       string
	SourceMessageID string
	DocumentID      string
	SourceType      string
	Content         string
	Status          string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
