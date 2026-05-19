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
	approvalusecase "github.com/sevoniva/nivora/internal/usecase/approval"
	artifactusecase "github.com/sevoniva/nivora/internal/usecase/artifact"
	authusecase "github.com/sevoniva/nivora/internal/usecase/auth"
	cloudusecase "github.com/sevoniva/nivora/internal/usecase/cloud"
	credentialusecase "github.com/sevoniva/nivora/internal/usecase/credential"
	deploymentusecase "github.com/sevoniva/nivora/internal/usecase/deployment"
	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
	pluginusecase "github.com/sevoniva/nivora/internal/usecase/plugin"
	releaseorchestration "github.com/sevoniva/nivora/internal/usecase/releaseorchestration"
	securityusecase "github.com/sevoniva/nivora/internal/usecase/security"
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
	pipelineService, closePipeline, err := appruntime.NewPipelineServiceWithConfig(ctx, cfg)
	if err != nil {
		return err
	}
	defer closePipeline()
	deploymentService := NewDeploymentService()
	artifactService := NewArtifactService()
	securityService := NewSecurityService()
	credentialService := NewCredentialService()
	authService := NewAuthService()
	approvalService := NewApprovalService()
	cloudService := NewCloudService()
	pluginRegistry := NewPluginRegistry()
	releaseService := NewReleaseOrchestrationServiceWith(artifactService, deploymentService)
	handler := routes.New(cfg, version.Current(), logger, pipelineService, deploymentService, artifactService, releaseService, securityService, credentialService, authService, approvalService, cloudService, pluginRegistry)
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
	err = srv.ListenAndServe()
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

func NewSecurityService() *securityusecase.Service {
	return appruntime.NewSecurityService()
}

func NewCredentialService() *credentialusecase.Service {
	return appruntime.NewCredentialService()
}

func NewAuthService() *authusecase.Service {
	return appruntime.NewAuthService()
}

func NewApprovalService() *approvalusecase.Service {
	return appruntime.NewApprovalService()
}

func NewCloudService() *cloudusecase.Service {
	return appruntime.NewCloudService()
}

func NewPluginRegistry() *pluginusecase.Registry {
	return appruntime.NewPluginRegistry()
}
