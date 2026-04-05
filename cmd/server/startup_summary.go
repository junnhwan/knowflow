package main

import (
	"fmt"
	"net/url"
	"strings"

	"log/slog"

	"knowflow/internal/config"
)

type startupSummary struct {
	HTTPPort       string
	PlaygroundURL  string
	PostgresTarget string
	RedisTarget    string
	ModelMode      string
	ModelProvider  string
	ChatModel      string
	ModelBaseURL   string
}

func buildStartupSummary(cfg config.Config) startupSummary {
	summary := startupSummary{
		HTTPPort:       cfg.HTTP.Port,
		PlaygroundURL:  fmt.Sprintf("http://localhost:%s/playground", cfg.HTTP.Port),
		PostgresTarget: postgresTarget(cfg.Postgres.DSN),
		RedisTarget:    fmt.Sprintf("%s/%d", cfg.Redis.Addr, cfg.Redis.DB),
		ModelProvider:  cfg.Model.Provider,
		ChatModel:      cfg.Model.ChatModel,
	}

	if cfg.Model.Provider != "local" && strings.TrimSpace(cfg.Model.APIKey) != "" {
		summary.ModelMode = "remote"
		summary.ModelBaseURL = cfg.Model.BaseURL
	} else {
		summary.ModelMode = "local"
	}
	return summary
}

func postgresTarget(dsn string) string {
	parsed, err := url.Parse(dsn)
	if err != nil {
		return "unparsed"
	}
	database := strings.TrimPrefix(parsed.Path, "/")
	if database == "" {
		database = "unknown"
	}
	host := parsed.Host
	if host == "" {
		host = "unknown"
	}
	return fmt.Sprintf("%s/%s", host, database)
}

func (s startupSummary) ContainsSecret(secret string) bool {
	if secret == "" {
		return false
	}
	fields := []string{
		s.HTTPPort,
		s.PlaygroundURL,
		s.PostgresTarget,
		s.RedisTarget,
		s.ModelMode,
		s.ModelProvider,
		s.ChatModel,
		s.ModelBaseURL,
	}
	for _, field := range fields {
		if strings.Contains(field, secret) {
			return true
		}
	}
	return false
}

func (s startupSummary) slogAttrs() []any {
	return []any{
		"http_port", s.HTTPPort,
		"playground_url", s.PlaygroundURL,
		"postgres_target", s.PostgresTarget,
		"redis_target", s.RedisTarget,
		"model_mode", s.ModelMode,
		"model_provider", s.ModelProvider,
		"chat_model", s.ChatModel,
		"model_base_url", s.ModelBaseURL,
		"stream_mode", "sse",
		"playground_enabled", true,
	}
}

func logStartupSummary(cfg config.Config) {
	summary := buildStartupSummary(cfg)
	slog.Info("application startup ready", summary.slogAttrs()...)
}
