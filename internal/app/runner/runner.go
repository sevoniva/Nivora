package runner

import (
	"context"

	shellexecutor "github.com/sevoniva/nivora/internal/adapters/executor/shell"
	appruntime "github.com/sevoniva/nivora/internal/app/runtime"
	domainrunner "github.com/sevoniva/nivora/internal/domain/runner"
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
	service := appruntime.NewPipelineService()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	runnerID := cfg.Runner.Name
	if runnerID == "" {
		runnerID = "local-runner"
	}
	if err := service.RegisterRunner(ctx, domainrunner.Runner{
		ID:        runnerID,
		Name:      cfg.Runner.Name,
		GroupID:   cfg.Runner.Group,
		Status:    "online",
		Labels:    map[string]string{"group": cfg.Runner.Group},
		Executors: []string{"shell"},
	}); err != nil {
		return err
	}
	if _, err := service.HeartbeatRunner(ctx, runnerID); err != nil {
		return err
	}

	logger.Info("runner registration ready", "runner", runnerID, "group", cfg.Runner.Group)
	logger.Info("runner heartbeat recorded", "interval", cfg.Runner.HeartbeatInterval)
	logger.Info("runner runtime is ready", "executor", "shell", "executor_initialized", exec != nil)
	return nil
}
