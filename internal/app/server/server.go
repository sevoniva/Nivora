package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/sevoniva/nivora/internal/api/http/routes"
	"github.com/sevoniva/nivora/internal/infra/config"
	"github.com/sevoniva/nivora/internal/infra/logging"
	"github.com/sevoniva/nivora/internal/version"
)

func Run(ctx context.Context, configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	logger := logging.New(cfg.Log.Level)
	return RunWithConfig(ctx, cfg, logger)
}

func RunWithConfig(ctx context.Context, cfg config.Config, logger *slog.Logger) error {
	handler := routes.New(cfg, version.Current(), logger)
	srv := &http.Server{
		Addr:              cfg.HTTP.BindAddress,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	logger.Info("nivora server starting", "address", cfg.HTTP.BindAddress)
	err := srv.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
