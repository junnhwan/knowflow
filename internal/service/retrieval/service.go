package retrieval

import (
	"context"
	"errors"

	"knowflow/internal/platform/llm"
)

type Candidate struct {
	ChunkID      string  `json:"chunk_id"`
	DocumentID   string  `json:"document_id"`
	SourceName   string  `json:"source_name"`
	Content      string  `json:"content"`
	VectorScore  float64 `json:"vector_score"`
	KeywordScore float64 `json:"keyword_score"`
	FusionScore  float64 `json:"fusion_score"`
	FinalScore   float64 `json:"final_score"`
}

type Metadata struct {
	Hit               bool `json:"hit"`
	Fallback          bool `json:"fallback"`
	VectorCandidates  int  `json:"vector_candidates"`
	KeywordCandidates int  `json:"keyword_candidates"`
	FinalCandidates   int  `json:"final_candidates"`
}

type Result struct {
	Chunks    []Candidate `json:"chunks"`
	Citations []Citation  `json:"citations"`
	Meta      Metadata    `json:"meta"`
}

type RetrieveRequest struct {
	UserID    string
	SessionID string
	Query     string
	TopK      int
}

type Config struct {
	VectorTopK  int
	KeywordTopK int
	FinalTopK   int
	RRFK        int
}

type Service struct {
	preprocessor     Preprocessor
	vectorRetriever  VectorRetriever
	keywordRetriever KeywordRetriever
	reranker         llm.Reranker
	config           Config
	telemetry        Telemetry
}

type Telemetry interface {
	RecordRAGHit(userID, sessionID string)
	RecordRAGMiss(userID, sessionID string)
	RecordRerankFallback(reason string)
}

func NewService(embedder llm.Embedder, store SearchStore, reranker llm.Reranker, cfg Config) *Service {
	if cfg.VectorTopK <= 0 {
		cfg.VectorTopK = 8
	}
	if cfg.KeywordTopK <= 0 {
		cfg.KeywordTopK = 8
	}
	if cfg.FinalTopK <= 0 {
		cfg.FinalTopK = 5
	}
	if cfg.RRFK <= 0 {
		cfg.RRFK = 60
	}

	return &Service{
		preprocessor:     NewPreprocessor(),
		vectorRetriever:  NewVectorRetriever(embedder, store),
		keywordRetriever: NewKeywordRetriever(store),
		reranker:         reranker,
		config:           cfg,
	}
}

func (s *Service) SetTelemetry(telemetry Telemetry) {
	s.telemetry = telemetry
}

func (s *Service) Retrieve(ctx context.Context, req RetrieveRequest) (Result, error) {
	if req.Query == "" {
		return Result{}, errors.New("query is required")
	}

	processed := s.preprocessor.Process(req.Query)
	vectorCandidates, err := s.vectorRetriever.Retrieve(ctx, req.UserID, processed, limitValue(req.TopK, s.config.VectorTopK))
	if err != nil {
		return Result{}, err
	}
	keywordCandidates, err := s.keywordRetriever.Retrieve(ctx, req.UserID, processed, limitValue(req.TopK, s.config.KeywordTopK))
	if err != nil {
		return Result{}, err
	}

	fused := FuseWithRRF(vectorCandidates, keywordCandidates, s.config.RRFK)
	meta := Metadata{
		Hit:               len(fused) > 0,
		VectorCandidates:  len(vectorCandidates),
		KeywordCandidates: len(keywordCandidates),
	}
	if len(fused) == 0 {
		if s.telemetry != nil {
			s.telemetry.RecordRAGMiss(req.UserID, req.SessionID)
		}
		return Result{Meta: meta}, nil
	}

	reranked, fallback, _ := ApplyRerank(ctx, s.reranker, processed.Normalized, fused, limitValue(req.TopK, s.config.FinalTopK))
	meta.Fallback = fallback
	meta.FinalCandidates = len(reranked)
	if s.telemetry != nil {
		s.telemetry.RecordRAGHit(req.UserID, req.SessionID)
		if fallback {
			s.telemetry.RecordRerankFallback("rerank_error")
		}
	}

	return Result{
		Chunks:    reranked,
		Citations: BuildCitations(reranked),
		Meta:      meta,
	}, nil
}

func limitValue(requested, fallback int) int {
	if requested > 0 {
		return requested
	}
	return fallback
}
