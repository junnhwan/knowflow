package memory

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestMemoryService_CompressesWhenThresholdExceeded(t *testing.T) {
	store := newInMemoryStore()
	svc := NewService(store, NewCompressor(fakeSummaryGenerator{}, CompressorConfig{
		RecentRounds:    2,
		TokenThreshold:  8,
		SummaryTokenCap: 64,
	}), ServiceConfig{
		TTLSeconds:       3600,
		FallbackRecentN:  4,
		LockTTL:          5 * time.Second,
		LockRetryTimes:   1,
		LockRetryBackoff: 5 * time.Millisecond,
	})

	result, err := svc.Update(context.Background(), UpdateRequest{
		UserID:    "demo-user",
		SessionID: "s-1",
		Incoming:  buildConversation(24),
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if !result.Compressed {
		t.Fatalf("expected compression to happen")
	}

	if result.Summary == "" {
		t.Fatalf("expected summary")
	}
}

type fakeSummaryGenerator struct{}

func (fakeSummaryGenerator) Summarize(_ context.Context, messages []MessageMemory, _ int) (string, error) {
	return fmt.Sprintf("summary(%d)", len(messages)), nil
}

func buildConversation(size int) []MessageMemory {
	out := make([]MessageMemory, 0, size)
	for i := 0; i < size; i++ {
		role := "user"
		if i%2 == 1 {
			role = "assistant"
		}
		out = append(out, MessageMemory{
			Role:    role,
			Content: fmt.Sprintf("message-%d", i),
		})
	}
	return out
}
