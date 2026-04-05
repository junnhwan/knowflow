package redis

import (
	"context"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"

	"knowflow/internal/service/memory"
)

func TestMemoryStore_SaveAndLoadRecentMessages(t *testing.T) {
	server := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{
		Addr: server.Addr(),
	})
	defer client.Close()

	store := NewMemoryStore(client)
	err := store.SaveRecent(context.Background(), "demo-user", "s-1", []memory.MessageMemory{
		{Role: "user", Content: "你好"},
		{Role: "assistant", Content: "你好，我是 KnowFlow"},
	}, time.Hour)
	if err != nil {
		t.Fatalf("SaveRecent() error = %v", err)
	}

	got, err := store.LoadRecent(context.Background(), "demo-user", "s-1")
	if err != nil {
		t.Fatalf("LoadRecent() error = %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("unexpected messages length: %d", len(got))
	}
}
