package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	HTTP          HTTPConfig
	Postgres      PostgresConfig
	Redis         RedisConfig
	Model         ModelConfig
	Retrieval     RetrievalConfig
	Memory        MemoryConfig
	Observability ObservabilityConfig
}

type HTTPConfig struct {
	Port string
}

type PostgresConfig struct {
	DSN string
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type ModelConfig struct {
	Provider           string
	BaseURL            string
	APIKey             string
	ChatModel          string
	EmbeddingModel     string
	RerankModel        string
	EmbeddingDimension int
}

type RetrievalConfig struct {
	VectorTopK int
	KeywordTopK int
	FinalTopK  int
}

type MemoryConfig struct {
	RecentMessages int
	TokenThreshold int
	RecentRounds   int
	TTLSeconds     int
}

type ObservabilityConfig struct {
	LogLevel string
}

func Load() (Config, error) {
	cfg := Config{
		HTTP: HTTPConfig{
			Port: getEnv("HTTP_PORT", "8080"),
		},
		Postgres: PostgresConfig{
			DSN: getEnv("POSTGRES_DSN", "postgres://knowflow:knowflow@localhost:5432/knowflow?sslmode=disable"),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		Model: ModelConfig{
			Provider:           strings.ToLower(getEnv("MODEL_PROVIDER", "local")),
			BaseURL:            getEnv("MODEL_BASE_URL", "https://dashscope.aliyuncs.com/compatible-mode/v1"),
			APIKey:             firstNonEmpty(os.Getenv("MODEL_API_KEY"), os.Getenv("DASHSCOPE_API_KEY")),
			ChatModel:          getEnv("MODEL_CHAT_NAME", "qwen-turbo"),
			EmbeddingModel:     getEnv("MODEL_EMBEDDING_NAME", "text-embedding-v4"),
			RerankModel:        getEnv("MODEL_RERANK_NAME", "gpt-rerank"),
			EmbeddingDimension: getEnvInt("EMBEDDING_DIMENSION", 64),
		},
		Retrieval: RetrievalConfig{
			VectorTopK:  getEnvInt("RETRIEVAL_VECTOR_TOP_K", 8),
			KeywordTopK: getEnvInt("RETRIEVAL_KEYWORD_TOP_K", 8),
			FinalTopK:   getEnvInt("RETRIEVAL_FINAL_TOP_K", 5),
		},
		Memory: MemoryConfig{
			RecentMessages: getEnvInt("MEMORY_RECENT_MESSAGES", 10),
			TokenThreshold: getEnvInt("MEMORY_TOKEN_THRESHOLD", 6000),
			RecentRounds:   getEnvInt("MEMORY_RECENT_ROUNDS", 5),
			TTLSeconds:     getEnvInt("MEMORY_TTL_SECONDS", 3600),
		},
		Observability: ObservabilityConfig{
			LogLevel: strings.ToUpper(getEnv("LOG_LEVEL", "INFO")),
		},
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.HTTP.Port) == "" {
		return errors.New("http port is required")
	}
	if strings.TrimSpace(c.Postgres.DSN) == "" {
		return errors.New("postgres dsn is required")
	}
	if strings.TrimSpace(c.Redis.Addr) == "" {
		return errors.New("redis addr is required")
	}
	if c.Retrieval.FinalTopK <= 0 {
		return fmt.Errorf("retrieval final top k must be positive")
	}
	if c.Model.EmbeddingDimension <= 0 {
		return fmt.Errorf("embedding dimension must be positive")
	}
	if c.Memory.TTLSeconds <= 0 {
		return fmt.Errorf("memory ttl must be positive")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
