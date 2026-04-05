package chat

import "time"

type Session struct {
	ID           string
	UserID       string
	Title        string
	LastActiveAt time.Time
	CreatedAt    time.Time
}
