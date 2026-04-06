package knowledge

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	knowledgedomain "knowflow/internal/domain/knowledge"
	"knowflow/internal/platform/llm"
	"knowflow/internal/service/retrieval"
)

var ErrEntryNotFound = errors.New("knowledge entry not found")

type GovernanceEntryStore interface {
	GetByID(ctx context.Context, entryID string) (knowledgedomain.Entry, error)
	ListByUser(ctx context.Context, userID string, filter ListFilter) ([]knowledgedomain.Entry, error)
	Update(ctx context.Context, entry knowledgedomain.Entry) error
}

type ChunkCleaner interface {
	DeleteChunks(ctx context.Context, entryID string) error
}

type VectorCandidateSearcher interface {
	SearchVector(ctx context.Context, userID string, embedding []float32, limit int) ([]retrieval.Candidate, error)
}

type EntryReindexer interface {
	ReindexEntry(ctx context.Context, entryID string) (IndexResult, error)
}

type GovernanceObserver interface {
	RecordKnowledgeDedupe(result string)
	RecordKnowledgeMerge(result string)
}

type GovernanceConfig struct {
	Now                 func() time.Time
	MaxDedupeCandidates int
	Observer            GovernanceObserver
}

type ListFilter struct {
	ReviewStatus string
	Query        string
	Limit        int
}

type UpdateEntryRequest struct {
	UserID       string   `json:"user_id"`
	KnowledgeID  string   `json:"knowledge_id"`
	Title        string   `json:"title"`
	Summary      string   `json:"summary"`
	Content      string   `json:"content"`
	Keywords     []string `json:"keywords"`
	ReviewStatus string   `json:"review_status"`
	SourceType   string   `json:"source_type"`
}

type MergeEntriesRequest struct {
	UserID        string `json:"user_id"`
	SourceEntryID string `json:"source_entry_id"`
	TargetEntryID string `json:"target_entry_id"`
}

type DedupeCandidate struct {
	KnowledgeEntryID string   `json:"knowledge_entry_id"`
	Title            string   `json:"title"`
	Similarity       float64  `json:"similarity"`
	MatchReasons     []string `json:"match_reasons"`
}

type EntryDetail struct {
	Entry            knowledgedomain.Entry `json:"entry"`
	DedupeCandidates []DedupeCandidate     `json:"dedupe_candidates,omitempty"`
}

type MergeResult struct {
	SourceEntry knowledgedomain.Entry `json:"source_entry"`
	TargetEntry knowledgedomain.Entry `json:"target_entry"`
}

type GovernanceService struct {
	entries             GovernanceEntryStore
	chunks              ChunkCleaner
	searcher            VectorCandidateSearcher
	indexer             EntryReindexer
	embedder            llm.Embedder
	now                 func() time.Time
	maxDedupeCandidates int
	observer            GovernanceObserver
}

func NewGovernanceService(entries GovernanceEntryStore, chunks ChunkCleaner, searcher VectorCandidateSearcher, indexer EntryReindexer, embedder llm.Embedder, cfg GovernanceConfig) *GovernanceService {
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	if cfg.MaxDedupeCandidates <= 0 {
		cfg.MaxDedupeCandidates = 5
	}
	return &GovernanceService{
		entries:             entries,
		chunks:              chunks,
		searcher:            searcher,
		indexer:             indexer,
		embedder:            embedder,
		now:                 now,
		maxDedupeCandidates: cfg.MaxDedupeCandidates,
		observer:            cfg.Observer,
	}
}

func (s *GovernanceService) ListEntries(ctx context.Context, userID string, filter ListFilter) ([]knowledgedomain.Entry, error) {
	return s.entries.ListByUser(ctx, userID, filter)
}

func (s *GovernanceService) GetEntry(ctx context.Context, userID, knowledgeID string) (EntryDetail, error) {
	entry, err := s.loadOwnedEntry(ctx, userID, knowledgeID)
	if err != nil {
		return EntryDetail{}, err
	}
	candidates, err := s.findDedupeCandidates(ctx, entry)
	if err != nil {
		return EntryDetail{}, err
	}
	return EntryDetail{
		Entry:            entry,
		DedupeCandidates: candidates,
	}, nil
}

func (s *GovernanceService) UpdateEntry(ctx context.Context, req UpdateEntryRequest) (EntryDetail, error) {
	entry, err := s.loadOwnedEntry(ctx, req.UserID, req.KnowledgeID)
	if err != nil {
		return EntryDetail{}, err
	}

	if strings.TrimSpace(req.Title) != "" {
		entry.Title = strings.TrimSpace(req.Title)
	}
	if strings.TrimSpace(req.Summary) != "" {
		entry.Summary = strings.TrimSpace(req.Summary)
	}
	if strings.TrimSpace(req.Content) != "" {
		entry.Content = strings.TrimSpace(req.Content)
	}
	if len(req.Keywords) > 0 {
		entry.Keywords = NormalizeKeywords(req.Keywords)
	}
	if strings.TrimSpace(req.ReviewStatus) != "" {
		entry.ReviewStatus = req.ReviewStatus
	}
	if strings.TrimSpace(req.SourceType) != "" {
		entry.SourceType = req.SourceType
	}
	entry.DedupeHash = BuildDedupeHash(entry.Title, entry.Summary, entry.Content)
	entry.QualityScore = BuildQualityScore(entry.Summary, entry.Content, entry.Keywords)
	entry.Status = "pending_index"
	entry.UpdatedAt = s.now()

	if err := s.entries.Update(ctx, entry); err != nil {
		return EntryDetail{}, err
	}
	if s.indexer != nil {
		if _, err := s.indexer.ReindexEntry(ctx, entry.ID); err != nil {
			return EntryDetail{}, err
		}
	}
	return s.GetEntry(ctx, req.UserID, entry.ID)
}

func (s *GovernanceService) DisableEntry(ctx context.Context, userID, knowledgeID string) (knowledgedomain.Entry, error) {
	entry, err := s.loadOwnedEntry(ctx, userID, knowledgeID)
	if err != nil {
		return knowledgedomain.Entry{}, err
	}
	now := s.now()
	entry.ReviewStatus = "disabled"
	entry.DisabledAt = &now
	entry.UpdatedAt = now
	if err := s.entries.Update(ctx, entry); err != nil {
		return knowledgedomain.Entry{}, err
	}
	if s.chunks != nil {
		if err := s.chunks.DeleteChunks(ctx, entry.ID); err != nil {
			return knowledgedomain.Entry{}, err
		}
	}
	return entry, nil
}

func (s *GovernanceService) MergeEntries(ctx context.Context, req MergeEntriesRequest) (MergeResult, error) {
	source, err := s.loadOwnedEntry(ctx, req.UserID, req.SourceEntryID)
	if err != nil {
		return MergeResult{}, err
	}
	target, err := s.loadOwnedEntry(ctx, req.UserID, req.TargetEntryID)
	if err != nil {
		return MergeResult{}, err
	}
	if source.ID == target.ID {
		return MergeResult{}, fmt.Errorf("source and target must be different")
	}

	now := s.now()
	target.Title = fallback(target.Title, source.Title)
	target.Summary = JoinUniqueText(target.Summary, source.Summary)
	target.Content = JoinUniqueText(target.Content, source.Content)
	target.Keywords = MergeKeywords(target.Keywords, source.Keywords)
	target.DedupeHash = BuildDedupeHash(target.Title, target.Summary, target.Content)
	target.QualityScore = BuildQualityScore(target.Summary, target.Content, target.Keywords)
	target.Status = "pending_index"
	target.UpdatedAt = now

	source.ReviewStatus = "merged"
	source.MergedIntoID = target.ID
	source.DisabledAt = &now
	source.UpdatedAt = now

	if err := s.entries.Update(ctx, target); err != nil {
		return MergeResult{}, err
	}
	if err := s.entries.Update(ctx, source); err != nil {
		return MergeResult{}, err
	}
	if s.chunks != nil {
		if err := s.chunks.DeleteChunks(ctx, source.ID); err != nil {
			return MergeResult{}, err
		}
	}
	if s.indexer != nil {
		if _, err := s.indexer.ReindexEntry(ctx, target.ID); err != nil {
			return MergeResult{}, err
		}
	}
	if s.observer != nil {
		s.observer.RecordKnowledgeMerge("success")
	}
	return MergeResult{
		SourceEntry: source,
		TargetEntry: target,
	}, nil
}

func (s *GovernanceService) loadOwnedEntry(ctx context.Context, userID, entryID string) (knowledgedomain.Entry, error) {
	entry, err := s.entries.GetByID(ctx, entryID)
	if err != nil {
		if errors.Is(err, ErrEntryNotFound) {
			return knowledgedomain.Entry{}, err
		}
		return knowledgedomain.Entry{}, err
	}
	if entry.UserID != userID {
		return knowledgedomain.Entry{}, ErrEntryNotFound
	}
	return entry, nil
}

func (s *GovernanceService) findDedupeCandidates(ctx context.Context, entry knowledgedomain.Entry) ([]DedupeCandidate, error) {
	if s.searcher == nil || s.embedder == nil || strings.TrimSpace(entry.Content) == "" {
		return nil, nil
	}
	vectors, err := s.embedder.Embed(ctx, []string{entry.Content})
	if err != nil || len(vectors) == 0 {
		return nil, err
	}
	vectorCandidates, err := s.searcher.SearchVector(ctx, entry.UserID, vectors[0], s.maxDedupeCandidates+1)
	if err != nil {
		return nil, err
	}
	entries, err := s.entries.ListByUser(ctx, entry.UserID, ListFilter{Limit: s.maxDedupeCandidates * 2})
	if err != nil {
		return nil, err
	}
	index := make(map[string]knowledgedomain.Entry, len(entries))
	for _, candidate := range entries {
		index[candidate.ID] = candidate
	}

	out := make([]DedupeCandidate, 0, len(vectorCandidates))
	for _, candidate := range vectorCandidates {
		if candidate.KnowledgeEntryID == "" || candidate.KnowledgeEntryID == entry.ID {
			continue
		}
		target, ok := index[candidate.KnowledgeEntryID]
		if !ok {
			continue
		}
		reasons := dedupeReasons(entry, target, candidate.VectorScore)
		if len(reasons) == 0 {
			continue
		}
		out = append(out, DedupeCandidate{
			KnowledgeEntryID: target.ID,
			Title:            target.Title,
			Similarity:       candidate.VectorScore,
			MatchReasons:     reasons,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Similarity > out[j].Similarity
	})
	if len(out) > s.maxDedupeCandidates {
		out = out[:s.maxDedupeCandidates]
	}
	if len(out) > 0 && s.observer != nil {
		s.observer.RecordKnowledgeDedupe("candidate")
	}
	return out, nil
}

func dedupeReasons(source, target knowledgedomain.Entry, similarity float64) []string {
	reasons := make([]string, 0, 3)
	if source.DedupeHash != "" && source.DedupeHash == target.DedupeHash {
		reasons = append(reasons, "dedupe_hash")
	}
	if similarity >= 0.85 {
		reasons = append(reasons, "embedding")
	}
	if keywordOverlap(source.Keywords, target.Keywords) > 0 {
		reasons = append(reasons, "keywords")
	}
	return reasons
}

func keywordOverlap(left, right []string) int {
	if len(left) == 0 || len(right) == 0 {
		return 0
	}
	lookup := make(map[string]struct{}, len(left))
	for _, item := range NormalizeKeywords(left) {
		lookup[strings.ToLower(item)] = struct{}{}
	}
	count := 0
	for _, item := range NormalizeKeywords(right) {
		if _, ok := lookup[strings.ToLower(item)]; ok {
			count++
		}
	}
	return count
}

func fallback(primary, secondary string) string {
	if strings.TrimSpace(primary) != "" {
		return primary
	}
	return strings.TrimSpace(secondary)
}
