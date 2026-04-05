package retrieval

import "context"

type KeywordRetriever struct {
	store SearchStore
}

func NewKeywordRetriever(store SearchStore) KeywordRetriever {
	return KeywordRetriever{store: store}
}

func (r KeywordRetriever) Retrieve(ctx context.Context, userID string, query ProcessedQuery, limit int) ([]Candidate, error) {
	return r.store.SearchKeyword(ctx, userID, query.Normalized, query.Tokens, limit)
}
