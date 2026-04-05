package memory

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Store interface {
	LoadRecent(ctx context.Context, userID, sessionID string) ([]MessageMemory, error)
	SaveRecent(ctx context.Context, userID, sessionID string, messages []MessageMemory, ttl time.Duration) error
	LoadSummary(ctx context.Context, userID, sessionID string) (string, error)
	SaveSummary(ctx context.Context, userID, sessionID, summary string, ttl time.Duration) error
	AcquireLock(ctx context.Context, userID, sessionID string, ttl time.Duration) (string, bool, error)
	ReleaseLock(ctx context.Context, userID, sessionID, token string) error
}

type ServiceConfig struct {
	TTLSeconds       int
	FallbackRecentN  int
	LockTTL          time.Duration
	LockRetryTimes   int
	LockRetryBackoff time.Duration
}

type LoadResult struct {
	Recent   []MessageMemory
	Summary  string
	Combined []MessageMemory
}

type UpdateRequest struct {
	UserID    string
	SessionID string
	Incoming  []MessageMemory
}

type UpdateResult struct {
	Recent     []MessageMemory
	Summary    string
	Compressed bool
}

type Service struct {
	store      Store
	compressor *Compressor
	config     ServiceConfig
}

func NewService(store Store, compressor *Compressor, cfg ServiceConfig) *Service {
	if cfg.TTLSeconds <= 0 {
		cfg.TTLSeconds = 3600
	}
	if cfg.FallbackRecentN <= 0 {
		cfg.FallbackRecentN = 10
	}
	if cfg.LockTTL <= 0 {
		cfg.LockTTL = 5 * time.Second
	}
	if cfg.LockRetryTimes <= 0 {
		cfg.LockRetryTimes = 3
	}
	if cfg.LockRetryBackoff <= 0 {
		cfg.LockRetryBackoff = 50 * time.Millisecond
	}
	return &Service{
		store:      store,
		compressor: compressor,
		config:     cfg,
	}
}

func (s *Service) Load(ctx context.Context, userID, sessionID string) (LoadResult, error) {
	recent, err := s.store.LoadRecent(ctx, userID, sessionID)
	if err != nil {
		return LoadResult{}, err
	}
	summary, err := s.store.LoadSummary(ctx, userID, sessionID)
	if err != nil {
		return LoadResult{}, err
	}

	combined := make([]MessageMemory, 0, len(recent)+1)
	if summary != "" {
		combined = append(combined, MessageMemory{
			Role:    "system",
			Content: "历史摘要: " + summary,
		})
	}
	combined = append(combined, recent...)
	return LoadResult{
		Recent:   recent,
		Summary:  summary,
		Combined: combined,
	}, nil
}

func (s *Service) Update(ctx context.Context, req UpdateRequest) (UpdateResult, error) {
	ttl := time.Duration(s.config.TTLSeconds) * time.Second
	lockToken, acquired, err := s.acquireWithRetry(ctx, req.UserID, req.SessionID)
	if err != nil {
		return UpdateResult{}, err
	}
	if !acquired {
		return s.degradeUpdate(ctx, req, ttl)
	}
	defer func() {
		_ = s.store.ReleaseLock(context.Background(), req.UserID, req.SessionID, lockToken)
	}()

	recent, err := s.store.LoadRecent(ctx, req.UserID, req.SessionID)
	if err != nil {
		return UpdateResult{}, err
	}
	allMessages := append(recent, req.Incoming...)

	compression, err := s.compressor.Compress(ctx, allMessages)
	if err != nil {
		return UpdateResult{}, err
	}

	if err := s.store.SaveRecent(ctx, req.UserID, req.SessionID, compression.Recent, ttl); err != nil {
		return UpdateResult{}, err
	}
	if compression.Summary != "" {
		if err := s.store.SaveSummary(ctx, req.UserID, req.SessionID, compression.Summary, ttl); err != nil {
			return UpdateResult{}, err
		}
	}

	return UpdateResult{
		Recent:     compression.Recent,
		Summary:    compression.Summary,
		Compressed: compression.Compressed,
	}, nil
}

func (s *Service) acquireWithRetry(ctx context.Context, userID, sessionID string) (string, bool, error) {
	for attempt := 0; attempt < s.config.LockRetryTimes; attempt++ {
		token, ok, err := s.store.AcquireLock(ctx, userID, sessionID, s.config.LockTTL)
		if err != nil {
			return "", false, err
		}
		if ok {
			return token, true, nil
		}
		time.Sleep(s.config.LockRetryBackoff)
	}
	return "", false, nil
}

func (s *Service) degradeUpdate(ctx context.Context, req UpdateRequest, ttl time.Duration) (UpdateResult, error) {
	recent, err := s.store.LoadRecent(ctx, req.UserID, req.SessionID)
	if err != nil {
		return UpdateResult{}, err
	}
	recent = append(recent, req.Incoming...)
	if len(recent) > s.config.FallbackRecentN {
		recent = recent[len(recent)-s.config.FallbackRecentN:]
	}
	if err := s.store.SaveRecent(ctx, req.UserID, req.SessionID, recent, ttl); err != nil {
		return UpdateResult{}, err
	}
	return UpdateResult{
		Recent: recent,
	}, nil
}

type inMemoryStore struct {
	mu       sync.Mutex
	recent   map[string][]MessageMemory
	summary  map[string]string
	lockKeys map[string]string
}

func newInMemoryStore() *inMemoryStore {
	return &inMemoryStore{
		recent:   map[string][]MessageMemory{},
		summary:  map[string]string{},
		lockKeys: map[string]string{},
	}
}

func (s *inMemoryStore) LoadRecent(_ context.Context, userID, sessionID string) ([]MessageMemory, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := memoryKey(userID, sessionID)
	return append([]MessageMemory(nil), s.recent[key]...), nil
}

func (s *inMemoryStore) SaveRecent(_ context.Context, userID, sessionID string, messages []MessageMemory, _ time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := memoryKey(userID, sessionID)
	s.recent[key] = append([]MessageMemory(nil), messages...)
	return nil
}

func (s *inMemoryStore) LoadSummary(_ context.Context, userID, sessionID string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.summary[memoryKey(userID, sessionID)], nil
}

func (s *inMemoryStore) SaveSummary(_ context.Context, userID, sessionID, summary string, _ time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.summary[memoryKey(userID, sessionID)] = summary
	return nil
}

func (s *inMemoryStore) AcquireLock(_ context.Context, userID, sessionID string, _ time.Duration) (string, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := memoryKey(userID, sessionID)
	if _, exists := s.lockKeys[key]; exists {
		return "", false, nil
	}
	token := fmt.Sprintf("lock-%s", key)
	s.lockKeys[key] = token
	return token, true, nil
}

func (s *inMemoryStore) ReleaseLock(_ context.Context, userID, sessionID, token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := memoryKey(userID, sessionID)
	if s.lockKeys[key] == token {
		delete(s.lockKeys, key)
	}
	return nil
}

func memoryKey(userID, sessionID string) string {
	return userID + ":" + sessionID
}
