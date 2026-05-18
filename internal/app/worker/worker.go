package worker

import (
	"context"

	"github.com/sevoniva/nivora/internal/infra/config"
	"github.com/sevoniva/nivora/internal/infra/logging"
)

func Run(ctx context.Context, configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	logger := logging.New(cfg.Log.Level)

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	logger.Info("worker runtime is ready", "event_bus", cfg.EventBus.Type)
	logger.Info("workflow advancement loop placeholder initialized")
	return nil
}
