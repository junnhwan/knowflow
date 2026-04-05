package main

import (
	"testing"

	"knowflow/internal/config"
)

func TestBuildStartupSummaryRedactsSecretsAndExtractsTargets(t *testing.T) {
	summary := buildStartupSummary(config.Config{
		HTTP: config.HTTPConfig{
			Port: "8080",
		},
		Postgres: config.PostgresConfig{
			DSN: "postgres://postgres:secret@47.112.180.205:5432/knowflow?sslmode=disable",
		},
		Redis: config.RedisConfig{
			Addr: "47.112.180.205:6379",
			DB:   0,
		},
		Model: config.ModelConfig{
			Provider:       "dashscope",
			BaseURL:        "https://dashscope.aliyuncs.com/compatible-mode/v1",
			APIKey:         "secret-key",
			ChatModel:      "qwen-plus",
			EmbeddingModel: "text-embedding-v4",
		},
	})

	if summary.HTTPPort != "8080" {
		t.Fatalf("unexpected http port: %s", summary.HTTPPort)
	}

	if summary.PlaygroundURL != "http://localhost:8080/playground" {
		t.Fatalf("unexpected playground url: %s", summary.PlaygroundURL)
	}

	if summary.PostgresTarget != "47.112.180.205:5432/knowflow" {
		t.Fatalf("unexpected postgres target: %s", summary.PostgresTarget)
	}

	if summary.RedisTarget != "47.112.180.205:6379/0" {
		t.Fatalf("unexpected redis target: %s", summary.RedisTarget)
	}

	if summary.ModelMode != "remote" {
		t.Fatalf("unexpected model mode: %s", summary.ModelMode)
	}

	if summary.ChatModel != "qwen-plus" {
		t.Fatalf("unexpected chat model: %s", summary.ChatModel)
	}

	if summary.ModelBaseURL != "https://dashscope.aliyuncs.com/compatible-mode/v1" {
		t.Fatalf("unexpected model base url: %s", summary.ModelBaseURL)
	}

	if summary.ContainsSecret("secret") {
		t.Fatal("expected startup summary to redact credentials")
	}
}

func TestBuildStartupSummaryMarksLocalModeWhenProviderIsLocal(t *testing.T) {
	summary := buildStartupSummary(config.Config{
		HTTP: config.HTTPConfig{Port: "8080"},
		Postgres: config.PostgresConfig{
			DSN: "postgres://postgres:secret@localhost:5432/knowflow?sslmode=disable",
		},
		Redis: config.RedisConfig{
			Addr: "localhost:6379",
			DB:   1,
		},
		Model: config.ModelConfig{
			Provider:  "local",
			ChatModel: "qwen-plus",
		},
	})

	if summary.ModelMode != "local" {
		t.Fatalf("unexpected model mode: %s", summary.ModelMode)
	}

	if summary.ModelBaseURL != "" {
		t.Fatalf("expected empty base url for local mode, got %s", summary.ModelBaseURL)
	}
}
