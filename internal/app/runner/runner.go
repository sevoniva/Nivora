package runner

import (
	"context"

	shellexecutor "github.com/sevoniva/nivora/internal/adapters/executor/shell"
	"github.com/sevoniva/nivora/internal/infra/config"
	"github.com/sevoniva/nivora/internal/infra/logging"
)

func Run(ctx context.Context, configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	logger := logging.New(cfg.Log.Level)
	exec := shellexecutor.New()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	logger.Info("runner registration placeholder ready", "runner", cfg.Runner.Name, "group", cfg.Runner.Group)
	logger.Info("runner heartbeat placeholder ready", "interval", cfg.Runner.HeartbeatInterval)
	logger.Info("runner runtime is ready", "executor", "shell", "executor_initialized", exec != nil)
	return nil
}
