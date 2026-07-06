package worker

import (
	"context"
	"log/slog"
	"time"

	appruntime "github.com/sevoniva/nivora/internal/app/runtime"
	"github.com/sevoniva/nivora/internal/infra/config"
	"github.com/sevoniva/nivora/internal/infra/logging"
	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
)

// reconcileInterval is the delay between worker reconciliation passes. Each
// pass processes queued PipelineRuns, recovers stale runs, reclaims expired
// job leases, and publishes pending outbox events.
// ponytail: fixed interval is enough for the foundation; make configurable
// when multi-worker throughput tuning matters.
const reconcileInterval = 5 * time.Second

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

	logger.Info("worker runtime is ready", "event_bus", cfg.EventBus.Type, "runtime_store", cfg.Database.RuntimeStore, "reconcile_interval", reconcileInterval)
	service, closePipeline, err := appruntime.NewPipelineServiceWithConfig(ctx, cfg)
	if err != nil {
		return err
	}
	defer closePipeline()

	ticker := time.NewTicker(reconcileInterval)
	defer ticker.Stop()

	// Run one pass immediately so a single short-lived invocation still
	// processes queued work without waiting for the first tick.
	if err := reconcileOnce(ctx, service, cfg.Runner.Name, logger); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := reconcileOnce(ctx, service, cfg.Runner.Name, logger); err != nil {
				return err
			}
		}
	}
}

func reconcileOnce(ctx context.Context, service *pipelineusecase.Service, workerID string, logger *slog.Logger) error {
	summary, err := service.ReconcileRuntime(ctx, pipelineRecoveryOptions(workerID))
	if err != nil {
		logger.Error("workflow recovery pass failed", "error", err)
		return err
	}
	if summary.ProcessedPipelineRuns > 0 || summary.RecoveredPipelineRuns > 0 || summary.QueuedPipelineRuns > 0 {
		logger.Info(
			"workflow recovery pass completed",
			"worker_id", summary.WorkerID,
			"queued_pipeline_runs", summary.QueuedPipelineRuns,
			"processed_pipeline_runs", summary.ProcessedPipelineRuns,
			"recovered_pipeline_runs", summary.RecoveredPipelineRuns,
			"expired_job_claims", summary.ExpiredJobClaims,
			"published_outbox_events", summary.PublishedOutboxEvents,
			"failed_outbox_events", summary.FailedOutboxEvents,
		)
	}
	return nil
}

func pipelineRecoveryOptions(workerID string) pipelineusecase.RuntimeRecoveryOptions {
	if workerID == "" {
		workerID = "nivora-worker"
	}
	return pipelineusecase.RuntimeRecoveryOptions{WorkerID: workerID}
}
