package worker

import (
	"context"

	appruntime "github.com/sevoniva/nivora/internal/app/runtime"
	"github.com/sevoniva/nivora/internal/infra/config"
	"github.com/sevoniva/nivora/internal/infra/logging"
	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
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

	logger.Info("worker runtime is ready", "event_bus", cfg.EventBus.Type, "runtime_store", cfg.Database.RuntimeStore)
	service, closePipeline, err := appruntime.NewPipelineServiceWithConfig(ctx, cfg)
	if err != nil {
		return err
	}
	defer closePipeline()
	summary, err := service.ReconcileRuntime(ctx, pipelineRecoveryOptions(cfg.Runner.Name))
	if err != nil {
		return err
	}
	logger.Info(
		"workflow recovery loop completed",
		"worker_id", summary.WorkerID,
		"queued_pipeline_runs", summary.QueuedPipelineRuns,
		"processed_pipeline_runs", summary.ProcessedPipelineRuns,
		"recovered_pipeline_runs", summary.RecoveredPipelineRuns,
		"expired_job_claims", summary.ExpiredJobClaims,
		"published_outbox_events", summary.PublishedOutboxEvents,
		"failed_outbox_events", summary.FailedOutboxEvents,
	)
	return nil
}

func pipelineRecoveryOptions(workerID string) pipelineusecase.RuntimeRecoveryOptions {
	if workerID == "" {
		workerID = "nivora-worker"
	}
	return pipelineusecase.RuntimeRecoveryOptions{WorkerID: workerID}
}
