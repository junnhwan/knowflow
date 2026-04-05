package config

import (
	"os"
	"path/filepath"
	"testing"
)

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

func TestLoadConfigFromDotEnv(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	content := []byte("POSTGRES_DSN=postgres://postgres:secret@47.112.180.205:5432/postgres?sslmode=disable\nREDIS_ADDR=47.112.180.205:6379\nREDIS_PASSWORD=secret\n")
	if err := os.WriteFile(filepath.Join(tempDir, ".env"), content, 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	t.Setenv("POSTGRES_DSN", "")
	t.Setenv("REDIS_ADDR", "")
	t.Setenv("REDIS_PASSWORD", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Postgres.DSN != "postgres://postgres:secret@47.112.180.205:5432/postgres?sslmode=disable" {
		t.Fatalf("unexpected postgres dsn: %s", cfg.Postgres.DSN)
	}

	if cfg.Redis.Addr != "47.112.180.205:6379" {
		t.Fatalf("unexpected redis addr: %s", cfg.Redis.Addr)
	}

	if cfg.Redis.Password != "secret" {
		t.Fatalf("unexpected redis password: %s", cfg.Redis.Password)
	}
}

func TestLoadConfigProcessEnvOverridesDotEnv(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	content := []byte("POSTGRES_DSN=postgres://dotenv:dotenv@127.0.0.1:5432/dotenv?sslmode=disable\n")
	if err := os.WriteFile(filepath.Join(tempDir, ".env"), content, 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	t.Setenv("POSTGRES_DSN", "postgres://process:process@127.0.0.1:5432/process?sslmode=disable")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Postgres.DSN != "postgres://process:process@127.0.0.1:5432/process?sslmode=disable" {
		t.Fatalf("unexpected postgres dsn: %s", cfg.Postgres.DSN)
	}
}

func TestLoadConfigRejectsUnsupportedEmbeddingDimension(t *testing.T) {
	t.Setenv("EMBEDDING_DIMENSION", "128")

	_, err := Load()
	if err == nil {
		t.Fatal("expected embedding dimension validation error")
	}
}
