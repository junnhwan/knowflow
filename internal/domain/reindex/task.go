package reindex

import "time"

type Task struct {
	ID           string
	UserID       string
	TargetType   string
	TargetID     string
	TriggerType  string
	Status       string
	AttemptCount int
	ErrorMessage string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
