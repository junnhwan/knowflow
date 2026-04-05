package postgres

import (
	"context"
	"encoding/json"

	chatdomain "knowflow/internal/domain/chat"
	pgplatform "knowflow/internal/platform/postgres"
)

type ChatRepository struct {
	db pgplatform.DB
}

func NewChatRepository(db pgplatform.DB) *ChatRepository {
	return &ChatRepository{db: db}
}

func (r *ChatRepository) CreateSession(ctx context.Context, session chatdomain.Session) error {
	_, err := r.db.Exec(ctx, `
INSERT INTO sessions (id, user_id, title, last_active_at, created_at)
VALUES ($1, $2, $3, $4, $5)
`, session.ID, session.UserID, session.Title, session.LastActiveAt, session.CreatedAt)
	return err
}

func (r *ChatRepository) GetSession(ctx context.Context, sessionID string) (chatdomain.Session, error) {
	row := r.db.QueryRow(ctx, `
SELECT id, user_id, title, last_active_at, created_at
FROM sessions
WHERE id = $1
`, sessionID)

	var session chatdomain.Session
	if err := row.Scan(&session.ID, &session.UserID, &session.Title, &session.LastActiveAt, &session.CreatedAt); err != nil {
		return chatdomain.Session{}, err
	}
	return session, nil
}

func (r *ChatRepository) ListSessions(ctx context.Context, userID string) ([]chatdomain.Session, error) {
	rows, err := r.db.Query(ctx, `
SELECT id, user_id, title, last_active_at, created_at
FROM sessions
WHERE user_id = $1
ORDER BY last_active_at DESC
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []chatdomain.Session
	for rows.Next() {
		var session chatdomain.Session
		if err := rows.Scan(&session.ID, &session.UserID, &session.Title, &session.LastActiveAt, &session.CreatedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}
	return sessions, rows.Err()
}

func (r *ChatRepository) AppendMessage(ctx context.Context, message chatdomain.Message) error {
	toolInput, _ := json.Marshal(message.ToolInput)
	toolOutput, _ := json.Marshal(message.ToolOutput)
	_, err := r.db.Exec(ctx, `
INSERT INTO messages (id, session_id, role, content, tool_name, tool_input, tool_output, created_at)
VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7::jsonb, $8)
`, message.ID, message.SessionID, message.Role, message.Content, message.ToolName, toolInput, toolOutput, message.CreatedAt)
	return err
}

func (r *ChatRepository) ListMessages(ctx context.Context, sessionID string) ([]chatdomain.Message, error) {
	rows, err := r.db.Query(ctx, `
SELECT id, session_id, role, content, tool_name, tool_input, tool_output, created_at
FROM messages
WHERE session_id = $1
ORDER BY created_at ASC
`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []chatdomain.Message
	for rows.Next() {
		var message chatdomain.Message
		var toolInput []byte
		var toolOutput []byte
		if err := rows.Scan(&message.ID, &message.SessionID, &message.Role, &message.Content, &message.ToolName, &toolInput, &toolOutput, &message.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(toolInput, &message.ToolInput)
		_ = json.Unmarshal(toolOutput, &message.ToolOutput)
		messages = append(messages, message)
	}
	return messages, rows.Err()
}
