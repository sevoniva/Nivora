package worker

import (
	"context"

	appruntime "github.com/sevoniva/nivora/internal/app/runtime"
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
	service := appruntime.NewPipelineService()
	processed, err := service.ProcessQueued(ctx, 10)
	if err != nil {
		return err
	}
	published, err := service.PublishPendingOutbox(ctx, 100)
	if err != nil {
		return err
	}
	logger.Info("workflow advancement loop completed", "processed_pipeline_runs", len(processed), "published_outbox_events", published)
	return nil
}
