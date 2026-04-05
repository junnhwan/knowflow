package chat

import "time"

type Message struct {
	ID         string
	SessionID  string
	Role       string
	Content    string
	ToolName   string
	ToolInput  map[string]any
	ToolOutput map[string]any
	CreatedAt  time.Time
}
