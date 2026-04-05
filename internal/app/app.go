package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"knowflow/internal/config"
)

type App struct {
	Config config.Config
	Server *http.Server
}

func New(cfg config.Config) (*App, error) {
	router := NewRouter(nil)
	return &App{
		Config: cfg,
		Server: &http.Server{
			Addr:              ":" + cfg.HTTP.Port,
			Handler:           router,
			ReadHeaderTimeout: 10 * time.Second,
		},
	}, nil
}

func (a *App) Run() error {
	if err := a.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("listen and serve: %w", err)
	}
	return nil
}

func (a *App) Shutdown(ctx context.Context) error {
	return a.Server.Shutdown(ctx)
}
