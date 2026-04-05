package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"knowflow/internal/config"
	"knowflow/internal/platform/llm"
	"knowflow/internal/platform/observability"
	pgplatform "knowflow/internal/platform/postgres"
	redisplatform "knowflow/internal/platform/redis"
	pgrepo "knowflow/internal/repository/postgres"
	redisrepo "knowflow/internal/repository/redis"
	chatservice "knowflow/internal/service/chat"
	"knowflow/internal/service/ingestion"
	"knowflow/internal/service/memory"
	"knowflow/internal/service/retrieval"
	"knowflow/internal/service/tools"
	"knowflow/internal/transport/http/handler"
)

type App struct {
	Config           config.Config
	Server           *http.Server
	Router           *gin.Engine
	Logger           *observability.Logger
	Metrics          *observability.Metrics
	DocumentHandler  *handler.DocumentHandler
	ChatHandler      *handler.ChatHandler
	KnowledgeHandler *handler.KnowledgeHandler
	postgres         *pgplatform.Client
	redis            *redisplatform.Client
}

func New(cfg config.Config) (*App, error) {
	ctx := context.Background()
	logger := observability.NewLogger(cfg.Observability.LogLevel)
	metrics := observability.NewMetrics()

	postgresClient, err := pgplatform.Open(ctx, cfg.Postgres.DSN)
	if err != nil {
		return nil, err
	}
	if err := postgresClient.Ping(ctx); err != nil {
		return nil, err
	}
	if err := pgplatform.RunMigrations(ctx, postgresClient.Pool, "migrations"); err != nil {
		return nil, err
	}

	redisClient := redisplatform.Open(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if err := redisClient.Ping(ctx); err != nil {
		return nil, err
	}

	documentRepo := pgrepo.NewDocumentRepository(postgresClient.Pool)
	chatRepo := pgrepo.NewChatRepository(postgresClient.Pool)
	knowledgeRepo := pgrepo.NewKnowledgeRepository(postgresClient.Pool)
	memoryStore := redisrepo.NewMemoryStore(redisClient.Raw)

	embedder := llm.LocalHasherEmbedder{Dimension: cfg.Model.EmbeddingDimension}
	reranker := llm.LocalOverlapReranker{}

	ingestionService := ingestion.NewService(documentRepo, embedder, ingestion.ServiceConfig{
		ChunkSize:    700,
		ChunkOverlap: 150,
	})
	retrievalService := retrieval.NewService(embedder, documentRepo, reranker, retrieval.Config{
		VectorTopK:  cfg.Retrieval.VectorTopK,
		KeywordTopK: cfg.Retrieval.KeywordTopK,
		FinalTopK:   cfg.Retrieval.FinalTopK,
		RRFK:        60,
	})
	retrievalService.SetTelemetry(metrics)
	memoryService := memory.NewService(memoryStore, memory.NewCompressor(memory.HeuristicSummaryGenerator{}, memory.CompressorConfig{
		RecentRounds:    cfg.Memory.RecentRounds,
		TokenThreshold:  cfg.Memory.TokenThreshold,
		SummaryTokenCap: 256,
	}), memory.ServiceConfig{
		TTLSeconds:       cfg.Memory.TTLSeconds,
		FallbackRecentN:  cfg.Memory.RecentMessages,
		LockTTL:          5 * time.Second,
		LockRetryTimes:   3,
		LockRetryBackoff: 50 * time.Millisecond,
	})

	registry := tools.NewRegistry(tools.ServiceConfig{Timeout: 3 * time.Second})
	registry.SetObserver(metrics)
	if err := registry.Register("retrieve_knowledge", tools.NewRetrieveKnowledgeTool(retrievalService)); err != nil {
		return nil, err
	}
	if err := registry.Register("upsert_knowledge", tools.NewUpsertKnowledgeTool(knowledgeRepo, time.Now, nil)); err != nil {
		return nil, err
	}
	if err := registry.Register("refresh_document_index", tools.NewRefreshDocumentIndexTool(documentReindexer{
		documents: documentRepo,
		ingestion: ingestionService,
	})); err != nil {
		return nil, err
	}

	answerer, providerLabel, err := buildAnswerer(ctx, cfg)
	if err != nil {
		return nil, err
	}

	orchestrator := chatservice.NewOrchestrator(chatservice.Dependencies{
		ChatStore: chatRepo,
		Memory:    memoryService,
		Tools:     registry,
		Answerer:  chatservice.NewTelemetryAnswerer(providerLabel, answerer, metrics),
	})

	app := &App{
		Config:           cfg,
		Logger:           logger,
		Metrics:          metrics,
		DocumentHandler:  handler.NewDocumentHandler(ingestionService),
		ChatHandler:      handler.NewChatHandler(orchestrator, chatRepo),
		KnowledgeHandler: handler.NewKnowledgeHandler(registry),
		postgres:         postgresClient,
		redis:            redisClient,
	}
	router := NewRouter(app)
	app.Router = router
	app.Server = &http.Server{
		Addr:              ":" + cfg.HTTP.Port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	return app, nil
}

func buildAnswerer(ctx context.Context, cfg config.Config) (chatservice.Answerer, string, error) {
	if cfg.Model.Provider != "local" && cfg.Model.APIKey != "" {
		answerer, err := chatservice.NewEinoAnswerer(ctx, cfg.Model.BaseURL, cfg.Model.APIKey, cfg.Model.ChatModel)
		if err != nil {
			return nil, "", err
		}
		return answerer, cfg.Model.Provider, nil
	}
	return chatservice.NewLocalAnswerer(), "local", nil
}

func (a *App) Run() error {
	if err := a.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("listen and serve: %w", err)
	}
	return nil
}

func (a *App) Shutdown(ctx context.Context) error {
	if a.redis != nil {
		_ = a.redis.Close()
	}
	if a.postgres != nil {
		a.postgres.Close()
	}
	return a.Server.Shutdown(ctx)
}

type documentReindexer struct {
	documents *pgrepo.DocumentRepository
	ingestion *ingestion.Service
}

func (r documentReindexer) Reindex(ctx context.Context, documentID string) (map[string]any, error) {
	doc, err := r.documents.GetByID(ctx, documentID)
	if err != nil {
		return nil, err
	}
	result, err := r.ingestion.Ingest(ctx, ingestion.IngestRequest{
		UserID:     doc.UserID,
		SourceName: doc.SourceName,
		Content:    doc.RawContent,
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"document_id": result.DocumentID,
		"chunk_count": result.ChunkCount,
		"status":      result.Status,
	}, nil
}
