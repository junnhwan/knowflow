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
	"knowflow/internal/service/guardrail"
	"knowflow/internal/service/ingestion"
	knowledgeservice "knowflow/internal/service/knowledge"
	"knowflow/internal/service/memory"
	"knowflow/internal/service/reindexer"
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
	backgroundCancel context.CancelFunc
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
	reindexTaskRepo := pgrepo.NewReindexTaskRepository(postgresClient.Pool)
	memoryStore := redisrepo.NewMemoryStore(redisClient.Raw)

	embedder := buildEmbedder(cfg)
	reranker := buildReranker(cfg)

	ingestionService := ingestion.NewService(documentRepo, embedder, ingestion.ServiceConfig{
		ChunkSize:    700,
		ChunkOverlap: 150,
	})
	knowledgeIndexService := knowledgeservice.NewService(knowledgeRepo, knowledgeRepo, embedder, knowledgeservice.Config{
		ChunkSize:    700,
		ChunkOverlap: 150,
	})
	knowledgeGovernanceService := knowledgeservice.NewGovernanceService(
		knowledgeRepo,
		knowledgeRepo,
		knowledgeRepo,
		knowledgeReindexer{knowledge: knowledgeIndexService},
		embedder,
		knowledgeservice.GovernanceConfig{},
	)
	searchRepo := pgrepo.NewHybridSearchRepository(documentRepo, knowledgeRepo)
	retrievalService := retrieval.NewService(embedder, searchRepo, reranker, retrieval.Config{
		VectorTopK:  cfg.Retrieval.VectorTopK,
		KeywordTopK: cfg.Retrieval.KeywordTopK,
		FinalTopK:   cfg.Retrieval.FinalTopK,
		RRFK:        60,
	})
	retrievalService.SetTelemetry(metrics)
	memoryService := memory.NewService(memoryStore, memory.NewCompressor(buildSummaryGenerator(cfg), memory.CompressorConfig{
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
	if err := registry.Register("upsert_knowledge", tools.NewUpsertKnowledgeTool(knowledgeRepo, knowledgeIndexService, time.Now, nil)); err != nil {
		return nil, err
	}
	if err := registry.Register("refresh_document_index", tools.NewRefreshDocumentIndexTool(documentReindexer{
		documents: documentRepo,
		ingestion: ingestionService,
	}, knowledgeReindexer{
		knowledge: knowledgeIndexService,
	})); err != nil {
		return nil, err
	}
	reindexTaskService := reindexer.NewTaskService(reindexTaskRepo, toolTaskExecutor{registry: registry}, reindexer.TaskServiceConfig{
		Observer: metrics,
	})

	answerer, providerLabel, err := buildAnswerer(ctx, cfg)
	if err != nil {
		return nil, err
	}
	guardrailService := guardrail.NewService(guardrail.Config{MaxMessageLength: 2000})

	orchestrator := chatservice.NewOrchestrator(chatservice.Dependencies{
		ChatStore:          chatRepo,
		Memory:             memoryService,
		Tools:              registry,
		KnowledgeExtractor: buildKnowledgeExtractor(cfg),
		OutputGuardrail:    guardrailService,
		GuardrailObserver:  metrics,
		GuardrailLogger:    logger,
		Answerer:           chatservice.NewTelemetryAnswerer(providerLabel, answerer, metrics),
	})
	backgroundCtx, backgroundCancel := context.WithCancel(context.Background())
	reindexWorker := reindexer.NewService(
		documentRepo,
		knowledgeRepo,
		documentReindexer{
			documents: documentRepo,
			ingestion: ingestionService,
		},
		knowledgeReindexer{
			knowledge: knowledgeIndexService,
		},
		reindexer.Config{
			RetryInterval: 30 * time.Second,
			BatchSize:     20,
			Observer:      metrics,
			Logger:        logger,
		},
	)
	go reindexWorker.Run(backgroundCtx)

	app := &App{
		Config:           cfg,
		Logger:           logger,
		Metrics:          metrics,
		DocumentHandler:  handler.NewDocumentHandler(ingestionService),
		ChatHandler:      handler.NewChatHandler(orchestrator, chatRepo, guardrailService, metrics, logger),
		KnowledgeHandler: handler.NewKnowledgeHandler(registry, knowledgeGovernanceService, reindexTaskService),
		postgres:         postgresClient,
		redis:            redisClient,
		backgroundCancel: backgroundCancel,
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

func buildEmbedder(cfg config.Config) llm.Embedder {
	local := llm.LocalHasherEmbedder{Dimension: cfg.Model.EmbeddingDimension}
	if cfg.Model.Provider == "local" || cfg.Model.EmbeddingAPIKey == "" {
		return local
	}
	return llm.OpenAICompatibleEmbedder{
		BaseURL:    cfg.Model.EmbeddingBaseURL,
		APIKey:     cfg.Model.EmbeddingAPIKey,
		Model:      cfg.Model.EmbeddingModel,
		Dimensions: cfg.Model.EmbeddingDimension,
	}
}

func buildReranker(cfg config.Config) llm.Reranker {
	local := llm.LocalOverlapReranker{}
	if cfg.Model.Provider == "local" || cfg.Model.RerankAPIKey == "" {
		return local
	}
	return llm.FallbackReranker{
		Primary: llm.DashScopeReranker{
			URL:         cfg.Model.RerankURL,
			APIKey:      cfg.Model.RerankAPIKey,
			Model:       cfg.Model.RerankModel,
			Instruction: cfg.Model.RerankInstruction,
		},
		Fallback: local,
	}
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

func buildKnowledgeExtractor(cfg config.Config) chatservice.KnowledgeExtractor {
	fallback := chatservice.RuleKnowledgeExtractor{}
	if cfg.Model.Provider == "local" || cfg.Model.APIKey == "" {
		return chatservice.FallbackKnowledgeExtractor{
			Fallback: fallback,
		}
	}
	return chatservice.FallbackKnowledgeExtractor{
		Primary: chatservice.LLMKnowledgeExtractor{
			Generator: llm.OpenAICompatibleTextGenerator{
				BaseURL:     cfg.Model.BaseURL,
				APIKey:      cfg.Model.APIKey,
				Model:       cfg.Model.ChatModel,
				Temperature: 0.1,
			},
		},
		Fallback: fallback,
	}
}

func buildSummaryGenerator(cfg config.Config) memory.SummaryGenerator {
	fallback := memory.HeuristicSummaryGenerator{}
	if cfg.Model.Provider == "local" || cfg.Model.APIKey == "" {
		return memory.FallbackSummaryGenerator{
			Fallback: fallback,
		}
	}
	return memory.FallbackSummaryGenerator{
		Primary: memory.LLMSummaryGenerator{
			Generator: llm.OpenAICompatibleTextGenerator{
				BaseURL:     cfg.Model.BaseURL,
				APIKey:      cfg.Model.APIKey,
				Model:       cfg.Model.ChatModel,
				Temperature: 0.1,
			},
		},
		Fallback: fallback,
	}
}

func (a *App) Run() error {
	if err := a.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("listen and serve: %w", err)
	}
	return nil
}

func (a *App) Shutdown(ctx context.Context) error {
	if a.backgroundCancel != nil {
		a.backgroundCancel()
	}
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

func (r documentReindexer) ReindexDocument(ctx context.Context, documentID string) (map[string]any, error) {
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
		"target_type": "document",
		"document_id": result.DocumentID,
		"chunk_count": result.ChunkCount,
		"status":      result.Status,
	}, nil
}

type knowledgeReindexer struct {
	knowledge *knowledgeservice.Service
}

type toolTaskExecutor struct {
	registry *tools.Registry
}

func (e toolTaskExecutor) Execute(ctx context.Context, toolName string, input map[string]any) error {
	if e.registry == nil {
		return fmt.Errorf("tool registry is not configured")
	}
	_, err := e.registry.Execute(ctx, toolName, input)
	return err
}

func (r knowledgeReindexer) ReindexEntry(ctx context.Context, entryID string) (knowledgeservice.IndexResult, error) {
	return r.knowledge.ReindexEntry(ctx, entryID)
}

func (r knowledgeReindexer) ReindexKnowledgeEntry(ctx context.Context, entryID string) (map[string]any, error) {
	result, err := r.ReindexEntry(ctx, entryID)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"target_type":        "knowledge_entry",
		"knowledge_entry_id": result.EntryID,
		"chunk_count":        result.ChunkCount,
		"status":             result.Status,
	}, nil
}
