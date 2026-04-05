package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"knowflow/internal/app"
	"knowflow/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config failed", "error", err)
		os.Exit(1)
	}

	application, err := app.New(cfg)
	if err != nil {
		slog.Error("create application failed", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := application.Shutdown(shutdownCtx); err != nil {
			slog.Error("shutdown failed", "error", err)
		}
	}()

	if err := application.Run(); err != nil {
		slog.Error("run failed", "error", err)
		os.Exit(1)
	}
}
