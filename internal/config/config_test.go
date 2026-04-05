package config

import "testing"

func TestLoadConfigFromEnv(t *testing.T) {
	t.Setenv("HTTP_PORT", "8080")
	t.Setenv("POSTGRES_DSN", "postgres://knowflow:knowflow@localhost:5432/knowflow?sslmode=disable")
	t.Setenv("REDIS_ADDR", "localhost:6379")
	t.Setenv("MODEL_PROVIDER", "local")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.HTTP.Port != "8080" {
		t.Fatalf("unexpected http port: %s", cfg.HTTP.Port)
	}

	if cfg.Redis.Addr != "localhost:6379" {
		t.Fatalf("unexpected redis addr: %s", cfg.Redis.Addr)
	}

	if cfg.Model.Provider != "local" {
		t.Fatalf("unexpected model provider: %s", cfg.Model.Provider)
	}

	if cfg.Retrieval.FinalTopK != 5 {
		t.Fatalf("unexpected final top k: %d", cfg.Retrieval.FinalTopK)
	}
}
