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
	complianceusecase "github.com/sevoniva/nivora/internal/usecase/compliance"
	credentialusecase "github.com/sevoniva/nivora/internal/usecase/credential"
	deploymentusecase "github.com/sevoniva/nivora/internal/usecase/deployment"
	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
	pluginusecase "github.com/sevoniva/nivora/internal/usecase/plugin"
	releaseorchestration "github.com/sevoniva/nivora/internal/usecase/releaseorchestration"
	securityusecase "github.com/sevoniva/nivora/internal/usecase/security"
	tenancyusecase "github.com/sevoniva/nivora/internal/usecase/tenancy"
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
	secretProvider := appruntime.NewSecretProvider()
	pipelineService, closePipeline, err := appruntime.NewPipelineServiceWithConfig(ctx, cfg)
	if err != nil {
		return err
	}
	defer closePipeline()
	artifactService, closeArtifact, err := appruntime.NewArtifactServiceWithConfigAndSecretProvider(ctx, cfg, secretProvider)
	if err != nil {
		return err
	}
	defer closeArtifact()
	securityService, closeSecurity, err := appruntime.NewSecurityServiceWithConfig(ctx, cfg)
	if err != nil {
		return err
	}
	defer closeSecurity()
	credentialService, closeCredential, err := appruntime.NewCredentialServiceWithConfigAndSecretProvider(ctx, cfg, secretProvider)
	if err != nil {
		return err
	}
	defer closeCredential()
	authService, closeAuth, err := appruntime.NewAuthServiceWithConfig(ctx, cfg)
	if err != nil {
		return err
	}
	defer closeAuth()
	approvalService, closeApproval, err := appruntime.NewApprovalServiceWithConfig(ctx, cfg)
	if err != nil {
		return err
	}
	defer closeApproval()
	cloudService, closeCloud, err := appruntime.NewCloudServiceWithConfig(ctx, cfg)
	if err != nil {
		return err
	}
	defer closeCloud()
	tenancyService, closeTenancy, err := appruntime.NewTenancyServiceWithConfig(ctx, cfg)
	if err != nil {
		return err
	}
	defer closeTenancy()
	deploymentService, closeDeployment, err := appruntime.NewDeploymentServiceWithConfigDependencies(ctx, cfg, securityService, approvalService)
	if err != nil {
		return err
	}
	defer closeDeployment()
	pluginRegistry := NewPluginRegistry()
	catalogService, closeCatalog, err := appruntime.NewCatalogServiceWithConfig(ctx, cfg)
	if err != nil {
		return err
	}
	defer closeCatalog()
	pipelineCatalog, closePipelineCatalog, err := appruntime.NewPipelineDefinitionCatalogWithConfig(ctx, cfg)
	if err != nil {
		return err
	}
	defer closePipelineCatalog()
	artifactRegistryCatalog, closeArtifactRegistryCatalog, err := appruntime.NewArtifactRegistryServiceWithConfigAndSecretProvider(ctx, cfg, secretProvider)
	if err != nil {
		return err
	}
	defer closeArtifactRegistryCatalog()
	policyCatalog, closePolicyCatalog, err := appruntime.NewPolicyCatalogServiceWithConfig(ctx, cfg)
	if err != nil {
		return err
	}
	defer closePolicyCatalog()
	repositoryService, closeRepository, err := appruntime.NewRepositoryServiceWithConfig(ctx, cfg)
	if err != nil {
		return err
	}
	defer closeRepository()
	deploymentService.WithPolicyCatalog(policyCatalog)
	releaseService, closeRelease, err := appruntime.NewReleaseOrchestrationServiceWithConfigDependencies(ctx, cfg, artifactService, deploymentService, securityService, approvalService)
	if err != nil {
		return err
	}
	defer closeRelease()
	releaseService.WithPolicyCatalog(policyCatalog)
	complianceService, closeCompliance, err := appruntime.NewComplianceServiceWithConfig(ctx, cfg, pipelineService, deploymentService, artifactService, releaseService, securityService, approvalService)
	if err != nil {
		return err
	}
	defer closeCompliance()
	handler := routes.New(
		cfg,
		version.Current(),
		logger,
		pipelineService,
		deploymentService,
		artifactService,
		releaseService,
		securityService,
		credentialService,
		authService,
		approvalService,
		cloudService,
		tenancyService,
		complianceService,
		pluginRegistry,
		routes.WithCatalogService(catalogService),
		routes.WithPipelineDefinitionCatalog(pipelineCatalog),
		routes.WithArtifactRegistryCatalog(artifactRegistryCatalog),
		routes.WithPolicyCatalog(policyCatalog),
		routes.WithRepositoryService(repositoryService),
	)
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

func NewTenancyService() *tenancyusecase.Service {
	return appruntime.NewTenancyService()
}

func NewComplianceService(pipelineService *pipelineusecase.Service, deploymentService *deploymentusecase.Service, artifactService *artifactusecase.Service, releaseService *releaseorchestration.Service, securityService *securityusecase.Service, approvalService *approvalusecase.Service) *complianceusecase.Service {
	return appruntime.NewComplianceService(pipelineService, deploymentService, artifactService, releaseService, securityService, approvalService)
}

func NewPluginRegistry() *pluginusecase.Registry {
	return appruntime.NewPluginRegistry()
}
