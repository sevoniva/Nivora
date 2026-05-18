package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/sevoniva/nivora/internal/api/http/routes"
	appruntime "github.com/sevoniva/nivora/internal/app/runtime"
	"github.com/sevoniva/nivora/internal/infra/config"
	"github.com/sevoniva/nivora/internal/infra/logging"
	artifactusecase "github.com/sevoniva/nivora/internal/usecase/artifact"
	deploymentusecase "github.com/sevoniva/nivora/internal/usecase/deployment"
	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
	releaseorchestration "github.com/sevoniva/nivora/internal/usecase/releaseorchestration"
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
	pipelineService := NewPipelineService()
	deploymentService := NewDeploymentService()
	artifactService := NewArtifactService()
	releaseService := NewReleaseOrchestrationServiceWith(artifactService, deploymentService)
	handler := routes.New(cfg, version.Current(), logger, pipelineService, deploymentService, artifactService, releaseService)
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

func NewPipelineService() *pipelineusecase.Service {
	return appruntime.NewPipelineService()
}

func NewDeploymentService() *deploymentusecase.Service {
	return appruntime.NewDeploymentService()
}

func NewArtifactService() *artifactusecase.Service {
	return appruntime.NewArtifactService()
}

func NewReleaseOrchestrationService() *releaseorchestration.Service {
	return appruntime.NewReleaseOrchestrationService()
}

func NewReleaseOrchestrationServiceWith(artifactService *artifactusecase.Service, deploymentService *deploymentusecase.Service) *releaseorchestration.Service {
	return appruntime.NewReleaseOrchestrationServiceWith(artifactService, deploymentService)
}
