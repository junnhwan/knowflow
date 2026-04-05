package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"knowflow/internal/service/memory"
)

type MemoryStore struct {
	client *goredis.Client
}

func NewMemoryStore(client *goredis.Client) *MemoryStore {
	return &MemoryStore{client: client}
}

func (s *MemoryStore) LoadRecent(ctx context.Context, userID, sessionID string) ([]memory.MessageMemory, error) {
	value, err := s.client.Get(ctx, recentKey(userID, sessionID)).Result()
	if err == goredis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var messages []memory.MessageMemory
	if err := json.Unmarshal([]byte(value), &messages); err != nil {
		return nil, err
	}
	return messages, nil
}

func (s *MemoryStore) SaveRecent(ctx context.Context, userID, sessionID string, messages []memory.MessageMemory, ttl time.Duration) error {
	payload, err := json.Marshal(messages)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, recentKey(userID, sessionID), payload, ttl).Err()
}

func (s *MemoryStore) LoadSummary(ctx context.Context, userID, sessionID string) (string, error) {
	value, err := s.client.Get(ctx, summaryKey(userID, sessionID)).Result()
	if err == goredis.Nil {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return value, nil
}

func (s *MemoryStore) SaveSummary(ctx context.Context, userID, sessionID, summary string, ttl time.Duration) error {
	return s.client.Set(ctx, summaryKey(userID, sessionID), summary, ttl).Err()
}

func (s *MemoryStore) AcquireLock(ctx context.Context, userID, sessionID string, ttl time.Duration) (string, bool, error) {
	token := fmt.Sprintf("%d", time.Now().UnixNano())
	ok, err := s.client.SetNX(ctx, lockKey(userID, sessionID), token, ttl).Result()
	if err != nil {
		return "", false, err
	}
	return token, ok, nil
}

func (s *MemoryStore) ReleaseLock(ctx context.Context, userID, sessionID, token string) error {
	script := `
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("DEL", KEYS[1])
end
return 0
`
	return s.client.Eval(ctx, script, []string{lockKey(userID, sessionID)}, token).Err()
}

func recentKey(userID, sessionID string) string {
	return fmt.Sprintf("knowflow:memory:%s:%s:recent", userID, sessionID)
}

func summaryKey(userID, sessionID string) string {
	return fmt.Sprintf("knowflow:memory:%s:%s:summary", userID, sessionID)
}

func lockKey(userID, sessionID string) string {
	return fmt.Sprintf("knowflow:memory:lock:%s:%s", userID, sessionID)
}
