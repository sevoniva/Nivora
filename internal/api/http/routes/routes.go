package routes

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	scmgeneric "github.com/sevoniva/nivora/internal/adapters/scm/generic"
	"github.com/sevoniva/nivora/internal/api/http/dto"
	"github.com/sevoniva/nivora/internal/api/http/handlers"
	apimiddleware "github.com/sevoniva/nivora/internal/api/http/middleware"
	apimcp "github.com/sevoniva/nivora/internal/api/mcp"
	"github.com/sevoniva/nivora/internal/infra/config"
	approvalusecase "github.com/sevoniva/nivora/internal/usecase/approval"
	artifactusecase "github.com/sevoniva/nivora/internal/usecase/artifact"
	authusecase "github.com/sevoniva/nivora/internal/usecase/auth"
	catalogusecase "github.com/sevoniva/nivora/internal/usecase/catalog"
	cloudusecase "github.com/sevoniva/nivora/internal/usecase/cloud"
	complianceusecase "github.com/sevoniva/nivora/internal/usecase/compliance"
	credentialusecase "github.com/sevoniva/nivora/internal/usecase/credential"
	deploymentusecase "github.com/sevoniva/nivora/internal/usecase/deployment"
	integrationusecase "github.com/sevoniva/nivora/internal/usecase/integration"
	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
	pluginusecase "github.com/sevoniva/nivora/internal/usecase/plugin"
	policyusecase "github.com/sevoniva/nivora/internal/usecase/policy"
	releaseorchestration "github.com/sevoniva/nivora/internal/usecase/releaseorchestration"
	repositoryusecase "github.com/sevoniva/nivora/internal/usecase/repository"
	runtimecenter "github.com/sevoniva/nivora/internal/usecase/runtimecenter"
	securityusecase "github.com/sevoniva/nivora/internal/usecase/security"
	tenancyusecase "github.com/sevoniva/nivora/internal/usecase/tenancy"
	workflowusecase "github.com/sevoniva/nivora/internal/usecase/workflow"
	"github.com/sevoniva/nivora/internal/version"
)

type Option func(*routeOptions)

type routeOptions struct {
	catalogService          *catalogusecase.Service
	pipelineCatalog         *pipelineusecase.DefinitionCatalog
	policyCatalog           *policyusecase.Service
	artifactRegistryCatalog *artifactusecase.RegistryService
	repositoryService       *repositoryusecase.Service
	workflowService         *workflowusecase.Service
}

func WithCatalogService(service *catalogusecase.Service) Option {
	return func(options *routeOptions) {
		if service != nil {
			options.catalogService = service
		}
	}
}

func WithPipelineDefinitionCatalog(catalog *pipelineusecase.DefinitionCatalog) Option {
	return func(options *routeOptions) {
		if catalog != nil {
			options.pipelineCatalog = catalog
		}
	}
}

func WithPolicyCatalog(service *policyusecase.Service) Option {
	return func(options *routeOptions) {
		if service != nil {
			options.policyCatalog = service
		}
	}
}

func WithArtifactRegistryCatalog(service *artifactusecase.RegistryService) Option {
	return func(options *routeOptions) {
		if service != nil {
			options.artifactRegistryCatalog = service
		}
	}
}

func WithRepositoryService(service *repositoryusecase.Service) Option {
	return func(options *routeOptions) {
		if service != nil {
			options.repositoryService = service
		}
	}
}

func WithWorkflowService(service *workflowusecase.Service) Option {
	return func(options *routeOptions) {
		if service != nil {
			options.workflowService = service
		}
	}
}

func New(cfg config.Config, info version.Info, logger *slog.Logger, pipelineService *pipelineusecase.Service, deploymentService *deploymentusecase.Service, artifactService *artifactusecase.Service, releaseService *releaseorchestration.Service, securityService *securityusecase.Service, credentialService *credentialusecase.Service, authService *authusecase.Service, approvalService *approvalusecase.Service, cloudService *cloudusecase.Service, tenancyService *tenancyusecase.Service, complianceService *complianceusecase.Service, pluginRegistry *pluginusecase.Registry, opts ...Option) http.Handler {
	r := chi.NewRouter()
	runtimeCenter := runtimecenter.NewService(pipelineService, deploymentService, releaseService)
	routeConfig := routeOptions{
		catalogService:          catalogusecase.NewService(catalogusecase.NewMemoryStore()),
		pipelineCatalog:         pipelineusecase.NewDefinitionCatalog(pipelineusecase.NewDefinitionMemoryStore()),
		policyCatalog:           policyusecase.NewService(policyusecase.NewMemoryStore()),
		artifactRegistryCatalog: artifactusecase.NewRegistryService(artifactusecase.NewRegistryMemoryStore()),
		repositoryService:       repositoryusecase.NewService(repositoryusecase.NewMemoryStore(), scmgeneric.New()),
		workflowService:         workflowusecase.NewService(workflowusecase.NewMemoryStore()),
	}
	for _, opt := range opts {
		opt(&routeConfig)
	}
	catalogService := routeConfig.catalogService
	pipelineCatalog := routeConfig.pipelineCatalog
	policyCatalog := routeConfig.policyCatalog
	artifactRegistryCatalog := routeConfig.artifactRegistryCatalog
	repositoryService := routeConfig.repositoryService
	workflowService := routeConfig.workflowService
	deploymentService.WithRepositoryCatalog(catalogService)
	deploymentService.WithPolicyCatalog(policyCatalog)
	releaseService.WithPolicyCatalog(policyCatalog)
	releaseService.WithReleaseTargetCatalog(catalogService)
	integrationService := integrationusecase.NewService(pluginRegistry)
	mcpServer := apimcp.NewServer(apimcp.Services{
		Config:       cfg,
		Auth:         authService,
		Pipelines:    pipelineService,
		PipelineDefs: pipelineCatalog,
		Deployments:  deploymentService,
		Catalog:      catalogService,
		Artifacts:    artifactService,
		Workflows:    workflowService,
		Releases:     releaseService,
		Security:     securityService,
		Compliance:   complianceService,
		Plugins:      pluginRegistry,
		Audit:        apimcp.NewComplianceAuditRecorder(complianceService),
	}, logger)
	r.Use(middleware.RequestID)
	r.Use(apimiddleware.RequestContext())
	r.Use(middleware.RealIP)
	r.Use(rejectOversizedRequestBody)
	r.Use(middleware.RequestSize(handlers.MaxRequestBodyBytes))
	r.Use(middleware.Recoverer)
	r.Use(apimiddleware.StructuredAccessLog(logger))

	r.Get("/healthz", handlers.Health)
	r.Get("/readyz", handlers.ReadyWithConfig(cfg))
	r.Get("/metrics", handlers.Metrics())

	r.Route("/api/v1", func(api chi.Router) {
		api.Use(apimiddleware.Authenticate(cfg.Auth, authService, handlers.RespondError))
		api.Get("/version", handlers.Version(info))
		api.Get("/system/info", handlers.SystemInfo(cfg))
		api.Get("/system/runtime", handlers.SystemRuntime(cfg))
		api.Get("/system/diagnostics", handlers.SystemDiagnostics(cfg))
		api.Get("/system/runtime/recovery", handlers.RuntimeRecoveryStatus(runtimeCenter))
		api.Post("/system/runtime/reconcile", handlers.ReconcileRuntime(runtimeCenter))
		api.Post("/mcp/rpc", handlers.RemoteMCPJSONRPC(cfg, mcpServer))
		api.Get("/tenancy/quota", handlers.GetQuota(tenancyService))
		api.Post("/tenancy/quota", apimiddleware.RequirePermission(authService, "project.write", handlers.RespondError, handlers.SetQuota(tenancyService)))
		api.Get("/tenancy/usage", handlers.GetUsageSummary(tenancyService))
		api.Get("/audit/verify", apimiddleware.RequirePermission(authService, "audit.read", handlers.RespondError, handlers.VerifyAuditChain(complianceService)))
		api.Get("/audit/search", apimiddleware.RequirePermission(authService, "audit.read", handlers.RespondError, handlers.SearchAudit(complianceService)))
		api.Get("/audit-logs", apimiddleware.RequirePermission(authService, "audit.read", handlers.RespondError, handlers.ListAuditLogs(complianceService)))
		api.Get("/events", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.ListEvents(pipelineService, deploymentService, releaseService, artifactService, securityService)))
		api.Get("/logs", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.ListLogs(pipelineService, deploymentService)))
		api.Get("/timeline", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.ListTimeline(pipelineService, deploymentService, releaseService, artifactService, securityService)))
		api.Get("/evidence/bundles", apimiddleware.RequirePermission(authService, "audit.read", handlers.RespondError, handlers.ListEvidenceBundles(complianceService)))
		api.Post("/evidence/bundles", apimiddleware.RequirePermission(authService, "audit.read", handlers.RespondError, handlers.GenerateEvidenceBundle(complianceService)))
		api.Get("/evidence/bundles/{id}", apimiddleware.RequirePermission(authService, "audit.read", handlers.RespondError, handlers.GetEvidenceBundleByID(complianceService)))
		api.Get("/evidence/bundles/{id}/export", apimiddleware.RequirePermission(authService, "audit.read", handlers.RespondError, handlers.ExportEvidenceBundleByID(complianceService)))
		api.Get("/evidence/{subject_type}/{id}", apimiddleware.RequirePermission(authService, "audit.read", handlers.RespondError, handlers.GetEvidenceBundle(complianceService)))
		api.Get("/retention-policy", apimiddleware.RequirePermission(authService, "audit.read", handlers.RespondError, handlers.GetRetentionPolicy(complianceService)))
		api.Post("/retention-policy", apimiddleware.RequirePermission(authService, "policy.manage", handlers.RespondError, handlers.SetRetentionPolicy(complianceService)))
		api.Post("/retention-policy/run", apimiddleware.RequirePermission(authService, "policy.manage", handlers.RespondError, handlers.RunRetentionPolicy(complianceService)))
		api.Get("/plugins", handlers.ListPlugins(pluginRegistry))
		api.Get("/plugins/{name}", handlers.GetPlugin(pluginRegistry))
		api.Get("/plugins/{name}/capabilities", handlers.GetPluginCapabilities(pluginRegistry))
		api.Post("/plugins/validate", handlers.ValidatePlugin(pluginRegistry))
		api.Get("/auth/whoami", handlers.WhoAmI())
		api.Get("/auth/permissions", handlers.AuthPermissions(authService))
		api.Get("/auth/token-info", handlers.AuthTokenInfo())
		api.Get("/users", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.ListUsers(authService)))
		api.Get("/service-accounts", apimiddleware.RequirePermission(authService, "credential.manage", handlers.RespondError, handlers.ListServiceAccounts(authService)))
		api.Post("/service-accounts", apimiddleware.RequirePermission(authService, "credential.manage", handlers.RespondError, handlers.CreateServiceAccount(authService)))
		api.Get("/auth/tokens", apimiddleware.RequirePermission(authService, "credential.manage", handlers.RespondError, handlers.ListAPITokens(authService)))
		api.Post("/auth/tokens", apimiddleware.RequirePermission(authService, "credential.manage", handlers.RespondError, handlers.CreateAPIToken(authService)))
		api.Post("/auth/tokens/{id}/rotate", apimiddleware.RequirePermission(authService, "credential.manage", handlers.RespondError, handlers.RotateAPIToken(authService)))
		api.Post("/auth/tokens/{id}/revoke", apimiddleware.RequirePermission(authService, "credential.manage", handlers.RespondError, handlers.RevokeAPIToken(authService)))
		api.Get("/roles", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.ListRoles(authService)))
		api.Get("/permissions", handlers.ListPermissions(authService))
		api.Get("/orgs/{id}/members", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.ListOrgMembers(authService)))
		api.Post("/orgs/{id}/members", apimiddleware.RequirePermission(authService, "project.write", handlers.RespondError, handlers.AddOrgMember(authService)))
		api.Get("/orgs", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.ListOrgs(catalogService)))
		api.Post("/orgs", apimiddleware.RequirePermission(authService, "project.write", handlers.RespondError, handlers.CreateOrg(catalogService)))
		api.Get("/orgs/{id}", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetOrg(catalogService)))
		api.Patch("/orgs/{id}", apimiddleware.RequirePermission(authService, "project.write", handlers.RespondError, handlers.UpdateOrg(catalogService)))
		api.Delete("/orgs/{id}", apimiddleware.RequirePermission(authService, "project.write", handlers.RespondError, handlers.DisableOrg(catalogService)))
		api.Get("/projects/{id}/members", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.ListProjectMembers(authService)))
		api.Post("/projects/{id}/members", apimiddleware.RequirePermission(authService, "project.write", handlers.RespondError, handlers.AddProjectMember(authService)))
		api.Get("/projects", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.ListProjects(catalogService)))
		api.Post("/projects", apimiddleware.RequirePermission(authService, "project.write", handlers.RespondError, handlers.CreateProject(catalogService)))
		api.Get("/projects/{id}", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetProject(catalogService)))
		api.Patch("/projects/{id}", apimiddleware.RequirePermission(authService, "project.write", handlers.RespondError, handlers.UpdateProject(catalogService)))
		api.Delete("/projects/{id}", apimiddleware.RequirePermission(authService, "project.write", handlers.RespondError, handlers.DisableProject(catalogService)))
		api.Get("/applications", apimiddleware.RequirePermission(authService, "application.read", handlers.RespondError, handlers.ListApplications(catalogService)))
		api.Post("/applications", apimiddleware.RequirePermission(authService, "application.write", handlers.RespondError, handlers.CreateApplication(catalogService)))
		api.Get("/applications/{id}", apimiddleware.RequirePermission(authService, "application.read", handlers.RespondError, handlers.GetApplication(catalogService)))
		api.Patch("/applications/{id}", apimiddleware.RequirePermission(authService, "application.write", handlers.RespondError, handlers.UpdateApplication(catalogService)))
		api.Delete("/applications/{id}", apimiddleware.RequirePermission(authService, "application.write", handlers.RespondError, handlers.DisableApplication(catalogService)))
		api.Get("/environments/{id}/members", apimiddleware.RequirePermission(authService, "environment.read", handlers.RespondError, handlers.ListEnvironmentMembers(authService)))
		api.Post("/environments/{id}/members", apimiddleware.RequirePermission(authService, "environment.write", handlers.RespondError, handlers.AddEnvironmentMember(authService)))
		api.Get("/environments", apimiddleware.RequirePermission(authService, "environment.read", handlers.RespondError, handlers.ListEnvironments(catalogService)))
		api.Post("/environments", apimiddleware.RequirePermission(authService, "environment.write", handlers.RespondError, handlers.CreateEnvironment(catalogService)))
		api.Get("/environments/{id}", apimiddleware.RequirePermission(authService, "environment.read", handlers.RespondError, handlers.GetEnvironment(catalogService)))
		api.Patch("/environments/{id}", apimiddleware.RequirePermission(authService, "environment.write", handlers.RespondError, handlers.UpdateEnvironment(catalogService)))
		api.Delete("/environments/{id}", apimiddleware.RequirePermission(authService, "environment.write", handlers.RespondError, handlers.DisableEnvironment(catalogService)))
		api.Get("/repositories", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.ListRepositories(catalogService)))
		api.Post("/repositories", apimiddleware.RequirePermission(authService, "project.write", handlers.RespondError, handlers.CreateRepository(catalogService)))
		api.Get("/repositories/{id}", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetRepository(catalogService)))
		api.Patch("/repositories/{id}", apimiddleware.RequirePermission(authService, "project.write", handlers.RespondError, handlers.UpdateRepository(catalogService)))
		api.Delete("/repositories/{id}", apimiddleware.RequirePermission(authService, "project.write", handlers.RespondError, handlers.DisableRepository(catalogService)))
		api.Post("/repositories/{id}/validate", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.ValidateRepository(catalogService)))
		api.Post("/repositories/{id}/snapshot", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.CreateRepositorySnapshot(catalogService, repositoryService)))
		api.Get("/repositories/{id}/snapshots", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.ListRepositorySnapshots(repositoryService)))
		api.Get("/repositories/{id}/intelligence", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetRepositoryIntelligence(repositoryService)))
		api.Post("/repositories/{id}/analyze", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.AnalyzeRepository(repositoryService)))
		api.Post("/devops/plan", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.PlanRepositoryDevOps(repositoryService)))
		api.Get("/workflows", apimiddleware.RequirePermission(authService, "workflow.plan", handlers.RespondError, handlers.ListWorkflows(workflowService)))
		api.Get("/workflows/plans", apimiddleware.RequirePermission(authService, "workflow.plan", handlers.RespondError, handlers.ListWorkflowPlans(workflowService)))
		api.Get("/workflows/plans/{id}", apimiddleware.RequirePermission(authService, "workflow.plan", handlers.RespondError, handlers.GetWorkflowPlan(workflowService)))
		api.Get("/workflows/runs", apimiddleware.RequirePermission(authService, "workflow.run", handlers.RespondError, handlers.ListWorkflowRuns(workflowService, pipelineService)))
		api.Get("/workflows/runs/{id}", apimiddleware.RequirePermission(authService, "workflow.run", handlers.RespondError, handlers.GetWorkflowRun(workflowService, pipelineService)))
		api.Post("/workflows/runs/reconcile", apimiddleware.RequirePermission(authService, "workflow.run", handlers.RespondError, handlers.ReconcileWorkflowRuns(workflowService, pipelineService)))
		api.Post("/workflows/runs/{id}/cancel", apimiddleware.RequirePermission(authService, "workflow.run", handlers.RespondError, handlers.CancelWorkflowRun(workflowService, pipelineService)))
		api.Post("/workflows/runs/{id}/retry", apimiddleware.RequirePermission(authService, "workflow.run", handlers.RespondError, handlers.RetryWorkflowRun(workflowService, pipelineService)))
		api.Get("/workflows/{id}/plan", apimiddleware.RequirePermission(authService, "workflow.plan", handlers.RespondError, handlers.GetWorkflowLatestPlan(workflowService)))
		api.Post("/workflows/validate", apimiddleware.RequirePermission(authService, "workflow.plan", handlers.RespondError, handlers.ValidateWorkflowDefinition()))
		api.Post("/workflows/plan", apimiddleware.RequirePermission(authService, "workflow.plan", handlers.RespondError, handlers.PlanWorkflowDefinition(workflowService)))
		api.Post("/workflows/run", apimiddleware.RequirePermission(authService, "workflow.run", handlers.RespondError, handlers.RunWorkflowDefinition(workflowService, pipelineService)))
		api.Get("/release-targets", apimiddleware.RequirePermission(authService, "environment.read", handlers.RespondError, handlers.ListReleaseTargets(catalogService)))
		api.Post("/release-targets", apimiddleware.RequirePermission(authService, "environment.write", handlers.RespondError, handlers.CreateReleaseTarget(catalogService)))
		api.Get("/release-targets/{id}", apimiddleware.RequirePermission(authService, "environment.read", handlers.RespondError, handlers.GetReleaseTarget(catalogService)))
		api.Patch("/release-targets/{id}", apimiddleware.RequirePermission(authService, "environment.write", handlers.RespondError, handlers.UpdateReleaseTarget(catalogService)))
		api.Delete("/release-targets/{id}", apimiddleware.RequirePermission(authService, "environment.write", handlers.RespondError, handlers.DisableReleaseTarget(catalogService)))
		api.Post("/release-targets/{id}/validate", apimiddleware.RequirePermission(authService, "deployment.create", handlers.RespondError, handlers.ValidateReleaseTarget(catalogService)))
		api.Get("/pipelines", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.ListPipelineDefinitions(pipelineCatalog)))
		api.Post("/pipelines", apimiddleware.RequirePermission(authService, "project.write", handlers.RespondError, handlers.CreatePipelineDefinition(pipelineCatalog)))
		api.Get("/pipelines/{id}", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetPipelineDefinition(pipelineCatalog)))
		api.Patch("/pipelines/{id}", apimiddleware.RequirePermission(authService, "project.write", handlers.RespondError, handlers.UpdatePipelineDefinition(pipelineCatalog)))
		api.Delete("/pipelines/{id}", apimiddleware.RequirePermission(authService, "project.write", handlers.RespondError, handlers.DisablePipelineDefinition(pipelineCatalog)))
		api.Get("/pipelines/{id}/versions", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.ListPipelineDefinitionVersions(pipelineCatalog)))
		api.Post("/pipelines/{id}/rollback", apimiddleware.RequirePermission(authService, "project.write", handlers.RespondError, handlers.RollbackPipelineDefinition(pipelineCatalog)))
		api.Post("/pipelines/{id}/runs", apimiddleware.RequirePermission(authService, "pipeline.run", handlers.RespondError, handlers.RunPipelineDefinition(pipelineCatalog, pipelineService)))
		api.Get("/pipeline-runs", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.ListPipelineRuns(pipelineService)))
		api.Post("/pipeline-runs", apimiddleware.RequirePermission(authService, "pipeline.run", handlers.RespondError, handlers.CreatePipelineRun(pipelineService)))
		api.Get("/pipeline-runs/{id}", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetPipelineRun(pipelineService)))
		api.Get("/pipeline-runs/{id}/dag", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetPipelineRunDAG(pipelineService)))
		api.Get("/pipeline-runs/{id}/jobs", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetPipelineRunJobs(pipelineService)))
		api.Get("/pipeline-runs/{id}/steps", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetPipelineRunSteps(pipelineService)))
		api.Get("/pipeline-runs/{id}/logs", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetPipelineRunLogs(pipelineService)))
		api.Get("/pipeline-runs/{id}/artifacts", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetPipelineRunArtifacts(pipelineService)))
		api.Get("/pipeline-runs/{id}/caches", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetPipelineRunCaches(pipelineService)))
		api.Get("/pipeline-runs/{id}/annotations", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetPipelineRunAnnotations(pipelineService)))
		api.Get("/pipeline-runs/{id}/summary", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetPipelineRunSummary(pipelineService)))
		api.Get("/pipeline-runs/{id}/events", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetPipelineRunEvents(pipelineService)))
		api.Get("/pipeline-runs/{id}/timeline", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetPipelineRunTimeline(pipelineService)))
		api.Post("/pipeline-runs/{id}/cancel", apimiddleware.RequirePermission(authService, "pipeline.run", handlers.RespondError, handlers.CancelPipelineRun(pipelineService)))
		api.Post("/pipeline-runs/{id}/cancel-request", apimiddleware.RequirePermission(authService, "pipeline.run", handlers.RespondError, handlers.RequestPipelineRunCancel(pipelineService)))
		api.Get("/runner-groups", apimiddleware.RequirePermission(authService, "runner.manage", handlers.RespondError, handlers.ListRunnerGroups(pipelineService)))
		api.Post("/runner-groups", apimiddleware.RequirePermission(authService, "runner.manage", handlers.RespondError, handlers.CreateRunnerGroup(pipelineService)))
		api.Get("/runner-groups/{id}", apimiddleware.RequirePermission(authService, "runner.manage", handlers.RespondError, handlers.GetRunnerGroup(pipelineService)))
		api.Get("/runners", apimiddleware.RequirePermission(authService, "runner.manage", handlers.RespondError, handlers.ListRunners(pipelineService)))
		api.Post("/runners/register", apimiddleware.RequirePermission(authService, "runner.manage", handlers.RespondError, handlers.RegisterRunner(pipelineService)))
		api.Get("/runners/{id}", apimiddleware.RequirePermission(authService, "runner.manage", handlers.RespondError, handlers.GetRunner(pipelineService)))
		api.Post("/runners/{id}/token/rotate", apimiddleware.RequirePermission(authService, "runner.manage", handlers.RespondError, handlers.RotateRunnerToken(pipelineService)))
		api.Post("/runners/{id}/token/revoke", apimiddleware.RequirePermission(authService, "runner.manage", handlers.RespondError, handlers.RevokeRunnerToken(pipelineService)))
		api.Post("/runners/{id}/heartbeat", handlers.HeartbeatRunner(pipelineService))
		api.Post("/runners/{id}/jobs/claim", handlers.ClaimRunnerJob(pipelineService))
		api.Post("/runners/{id}/jobs/{job_id}/logs", handlers.AppendJobLogs(pipelineService))
		api.Post("/runners/{id}/jobs/{job_id}/status", handlers.UpdateJobStatus(pipelineService))
		api.Post("/runners/offline-detect", apimiddleware.RequirePermission(authService, "runner.manage", handlers.RespondError, handlers.MarkOfflineRunners(pipelineService)))
		api.Post("/jobs/{id}/logs", apimiddleware.RequirePermission(authService, "runner.manage", handlers.RespondError, handlers.AppendJobLogs(pipelineService)))
		api.Post("/jobs/{id}/status", apimiddleware.RequirePermission(authService, "runner.manage", handlers.RespondError, handlers.UpdateJobStatus(pipelineService)))
		api.Post("/deployments/plan", apimiddleware.RequirePermission(authService, "deployment.create", handlers.RespondError, handlers.PlanDeploymentRun(deploymentService)))
		api.Post("/deployments/gitops/plan", apimiddleware.RequirePermission(authService, "deployment.create", handlers.RespondError, handlers.PlanGitOpsDeployment(deploymentService)))
		api.Post("/deployments/gitops/commit", apimiddleware.RequirePermission(authService, "deployment.create", handlers.RespondError, handlers.CommitGitOpsDeployment(deploymentService)))
		api.Post("/deployments/gitops/rollback", apimiddleware.RequirePermission(authService, "deployment.create", handlers.RespondError, handlers.RollbackGitOpsDeployment(deploymentService)))
		// Alias for GitOps deployment convenience; canonical path is POST /deployments
		api.Post("/deployments/gitops", apimiddleware.RequirePermission(authService, "deployment.create", handlers.RespondError, handlers.CreateDeploymentRun(deploymentService)))
		api.Get("/host-groups", apimiddleware.RequirePermission(authService, "environment.read", handlers.RespondError, handlers.ListHostGroups(deploymentService)))
		api.Post("/host-groups", apimiddleware.RequirePermission(authService, "environment.write", handlers.RespondError, handlers.CreateHostGroup(deploymentService)))
		api.Get("/host-groups/{id}", apimiddleware.RequirePermission(authService, "environment.read", handlers.RespondError, handlers.GetHostGroup(deploymentService)))
		api.Post("/deployments/host/plan", apimiddleware.RequirePermission(authService, "deployment.create", handlers.RespondError, handlers.PlanHostDeployment(deploymentService)))
		api.Get("/integrations", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.ListIntegrations(integrationService)))
		api.Get("/integrations/argocd/applications/{name}/status", apimiddleware.RequirePermission(authService, "deployment.create", handlers.RespondError, handlers.GetArgoCDApplicationStatus(deploymentService)))
		api.Get("/integrations/argocd/applications/{name}/resources", apimiddleware.RequirePermission(authService, "deployment.create", handlers.RespondError, handlers.GetArgoCDApplicationResources(deploymentService)))
		api.Post("/integrations/argocd/applications/{name}/sync", apimiddleware.RequirePermission(authService, "deployment.create", handlers.RespondError, handlers.SyncArgoCDApplication(deploymentService)))
		api.Get("/artifacts", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.ListArtifacts(artifactService)))
		api.Post("/artifacts", apimiddleware.RequirePermission(authService, "release.create", handlers.RespondError, handlers.CreateArtifact(artifactService)))
		api.Post("/artifacts/inspect", apimiddleware.RequirePermission(authService, "release.create", handlers.RespondError, handlers.InspectArtifact(artifactService)))
		api.Post("/artifacts/resolve", apimiddleware.RequirePermission(authService, "release.create", handlers.RespondError, handlers.ResolveArtifact(artifactService)))
		api.Get("/artifacts/{id}/releases", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetArtifactReleases(artifactService)))
		api.Get("/artifacts/{id}", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetArtifact(artifactService)))
		api.Get("/artifact-registries", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.ListArtifactRegistries(artifactRegistryCatalog)))
		api.Post("/artifact-registries", apimiddleware.RequirePermission(authService, "credential.manage", handlers.RespondError, handlers.CreateArtifactRegistry(artifactRegistryCatalog)))
		api.Get("/artifact-registries/{id}", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetArtifactRegistry(artifactRegistryCatalog)))
		api.Patch("/artifact-registries/{id}", apimiddleware.RequirePermission(authService, "credential.manage", handlers.RespondError, handlers.UpdateArtifactRegistry(artifactRegistryCatalog)))
		api.Delete("/artifact-registries/{id}", apimiddleware.RequirePermission(authService, "credential.manage", handlers.RespondError, handlers.DisableArtifactRegistry(artifactRegistryCatalog)))
		api.Get("/artifact-registries/{id}/artifacts", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.ListArtifactRegistryArtifacts(artifactRegistryCatalog)))
		api.Post("/artifact-registries/{id}/validate", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.ValidateSavedArtifactRegistry(artifactRegistryCatalog)))
		api.Post("/artifact-registries/validate", handlers.ValidateArtifactRegistry())
		api.Get("/security/scans", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.ListSecurityScans(securityService)))
		api.Post("/security/scans", apimiddleware.RequirePermission(authService, "policy.manage", handlers.RespondError, handlers.CreateSecurityScan(securityService, policyCatalog)))
		api.Get("/security/findings", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.ListSecurityFindings(securityService)))
		api.Get("/security/findings/{id}", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetSecurityFinding(securityService)))
		api.Get("/security/scans/{id}", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetSecurityScan(securityService)))
		api.Get("/security/scans/{id}/findings", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetSecurityFindings(securityService)))
		api.Get("/policies", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.ListPolicies(policyCatalog)))
		api.Post("/policies", apimiddleware.RequirePermission(authService, "policy.manage", handlers.RespondError, handlers.CreatePolicy(policyCatalog)))
		api.Get("/policies/results", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.ListPolicyResults(securityService)))
		api.Get("/policies/results/{id}", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetPolicyResult(securityService)))
		api.Get("/policies/{id}/attachments", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.ListPolicyAttachments(policyCatalog)))
		api.Post("/policies/{id}/attachments", apimiddleware.RequirePermission(authService, "policy.manage", handlers.RespondError, handlers.AttachPolicy(policyCatalog)))
		api.Post("/policies/{id}/evaluate", apimiddleware.RequirePermission(authService, "policy.manage", handlers.RespondError, handlers.EvaluatePolicyDefinition(policyCatalog, securityService)))
		api.Get("/policies/{id}", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetPolicy(policyCatalog)))
		api.Patch("/policies/{id}", apimiddleware.RequirePermission(authService, "policy.manage", handlers.RespondError, handlers.UpdatePolicy(policyCatalog)))
		api.Delete("/policies/{id}", apimiddleware.RequirePermission(authService, "policy.manage", handlers.RespondError, handlers.DisablePolicy(policyCatalog)))
		api.Post("/policies/evaluate", apimiddleware.RequirePermission(authService, "policy.manage", handlers.RespondError, handlers.EvaluatePolicy(securityService)))
		api.Post("/approvals", apimiddleware.RequirePermission(authService, "deployment.approve", handlers.RespondError, handlers.CreateApproval(approvalService)))
		api.Get("/approvals", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.ListApprovals(approvalService)))
		api.Get("/approvals/{id}", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetApproval(approvalService)))
		api.Post("/approvals/{id}/approve", apimiddleware.RequirePermission(authService, "deployment.approve", handlers.RespondError, handlers.ApproveApproval(approvalService)))
		api.Post("/approvals/{id}/reject", apimiddleware.RequirePermission(authService, "deployment.approve", handlers.RespondError, handlers.RejectApproval(approvalService)))
		api.Post("/approvals/{id}/cancel", apimiddleware.RequirePermission(authService, "deployment.approve", handlers.RespondError, handlers.CancelApproval(approvalService)))
		api.Post("/approvals/{id}/expire", apimiddleware.RequirePermission(authService, "deployment.approve", handlers.RespondError, handlers.ExpireApproval(approvalService)))
		api.Post("/approvals/{id}/resume-subject", apimiddleware.RequirePermission(authService, "deployment.approve", handlers.RespondError, handlers.ResumeApprovalSubject(approvalService, deploymentService, releaseService, pipelineService)))
		api.Get("/change-windows", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.ListChangeWindows(approvalService)))
		api.Post("/change-windows", apimiddleware.RequirePermission(authService, "environment.write", handlers.RespondError, handlers.CreateChangeWindow(approvalService)))
		api.Get("/change-windows/{id}", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetChangeWindow(approvalService)))
		api.Post("/change-windows/evaluate", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.EvaluateChangeWindow(approvalService)))
		api.Get("/notifications", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.ListNotifications(approvalService)))
		api.Post("/notifications/test", apimiddleware.RequirePermission(authService, "environment.write", handlers.RespondError, handlers.TestNotification(approvalService)))
		api.Get("/cloud/providers", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.ListCloudProviders(cloudService)))
		api.Post("/cloud/accounts", apimiddleware.RequirePermission(authService, "credential.manage", handlers.RespondError, handlers.CreateCloudAccount(cloudService)))
		api.Get("/cloud/accounts", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.ListCloudAccounts(cloudService)))
		api.Get("/cloud/accounts/{id}", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetCloudAccount(cloudService)))
		api.Post("/cloud/accounts/{id}/validate", apimiddleware.RequirePermission(authService, "credential.manage", handlers.RespondError, handlers.ValidateCloudAccount(cloudService)))
		api.Get("/cloud/accounts/{id}/regions", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.ListCloudRegions(cloudService)))
		api.Get("/cloud/accounts/{id}/clusters", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.ListCloudClusters(cloudService)))
		api.Get("/cloud/accounts/{id}/hosts", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.ListCloudHosts(cloudService)))
		api.Get("/cloud/accounts/{id}/registries", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.ListCloudRegistries(cloudService)))
		api.Get("/cloud/accounts/{id}/inventory", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetCloudInventory(cloudService)))
		api.Post("/secrets", apimiddleware.RequirePermission(authService, "credential.manage", handlers.RespondError, handlers.CreateSecret(credentialService)))
		api.Get("/secrets/refs", apimiddleware.RequirePermission(authService, "credential.manage", handlers.RespondError, handlers.ListSecretRefs(credentialService)))
		api.Post("/secrets/provider/validate", apimiddleware.RequirePermission(authService, "credential.manage", handlers.RespondError, handlers.ValidateSecretProvider(credentialService)))
		api.Post("/secrets/{id}/rotate", apimiddleware.RequirePermission(authService, "credential.manage", handlers.RespondError, handlers.RotateSecret(credentialService)))
		api.Delete("/secrets/{id}", apimiddleware.RequirePermission(authService, "credential.manage", handlers.RespondError, handlers.DeleteSecret(credentialService)))
		api.Post("/credentials", apimiddleware.RequirePermission(authService, "credential.manage", handlers.RespondError, handlers.CreateCredential(credentialService)))
		api.Get("/credentials", apimiddleware.RequirePermission(authService, "credential.manage", handlers.RespondError, handlers.ListCredentials(credentialService)))
		api.Get("/credentials/{id}", apimiddleware.RequirePermission(authService, "credential.manage", handlers.RespondError, handlers.GetCredential(credentialService)))
		api.Delete("/credentials/{id}", apimiddleware.RequirePermission(authService, "credential.manage", handlers.RespondError, handlers.DeleteCredential(credentialService)))
		api.Post("/credentials/{id}/validate", apimiddleware.RequirePermission(authService, "credential.manage", handlers.RespondError, handlers.ValidateCredential(credentialService)))
		api.Get("/releases", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.ListReleases(artifactService)))
		api.Post("/releases", apimiddleware.RequirePermission(authService, "release.create", handlers.RespondError, handlers.CreateRelease(artifactService)))
		api.Post("/releases/{id}/cancel", apimiddleware.RequirePermission(authService, "deployment.cancel", handlers.RespondError, handlers.CancelRelease(artifactService, releaseService)))
		api.Post("/releases/{id}/plan", apimiddleware.RequirePermission(authService, "release.create", handlers.RespondError, handlers.PlanRelease(releaseService)))
		api.Post("/releases/{id}/deploy", apimiddleware.RequirePermission(authService, "release.create", handlers.RespondError, handlers.DeployRelease(releaseService)))
		api.Post("/releases/{id}/evidence", apimiddleware.RequirePermission(authService, "audit.read", handlers.RespondError, handlers.GenerateReleaseEvidenceBundle(complianceService)))
		api.Get("/releases/{id}/plan", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetReleasePlan(releaseService)))
		api.Get("/releases/{id}/executions", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.ListReleaseExecutions(releaseService)))
		api.Get("/releases/{id}/security", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetReleaseSecurity(releaseService)))
		api.Get("/releases/{id}", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetRelease(artifactService)))
		api.Get("/releases/{id}/artifacts", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetReleaseArtifacts(artifactService)))
		api.Get("/releases/executions/{execution_id}", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetReleaseExecution(releaseService)))
		api.Get("/releases/executions/{execution_id}/timeline", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetReleaseExecutionTimeline(releaseService)))
		api.Get("/releases/executions/{execution_id}/targets", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetReleaseExecutionTargets(releaseService)))
		api.Post("/releases/executions/{execution_id}/cancel", apimiddleware.RequirePermission(authService, "deployment.cancel", handlers.RespondError, handlers.CancelReleaseExecution(releaseService)))
		api.Post("/releases/executions/{execution_id}/resume", apimiddleware.RequirePermission(authService, "deployment.approve", handlers.RespondError, handlers.ResumeReleaseExecutionAfterApproval(releaseService)))
		api.Get("/deployments", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.ListDeploymentRuns(deploymentService)))
		api.Post("/deployments", apimiddleware.RequirePermission(authService, "deployment.create", handlers.RespondError, handlers.CreateDeploymentRun(deploymentService)))
		api.Post("/deployments/apply", apimiddleware.RequirePermission(authService, "deployment.create", handlers.RespondError, handlers.ApplyDeploymentRun(deploymentService)))
		api.Get("/deployments/{id}", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetDeploymentRun(deploymentService)))
		api.Get("/deployments/{id}/plan", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetDeploymentPlan(deploymentService)))
		api.Get("/deployments/{id}/gitops-plan", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetDeploymentGitOpsPlan(deploymentService)))
		api.Get("/deployments/{id}/argocd-status", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetDeploymentArgoCDStatus(deploymentService)))
		api.Get("/deployments/{id}/resources", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetDeploymentResources(deploymentService)))
		api.Get("/deployments/{id}/hosts", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetDeploymentHosts(deploymentService)))
		api.Get("/deployments/{id}/health", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetDeploymentHealth(deploymentService)))
		api.Get("/deployments/{id}/diff", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetDeploymentRuntimeDiff(deploymentService)))
		api.Get("/deployments/{id}/manifest-snapshot", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetDeploymentManifestSnapshot(deploymentService)))
		api.Get("/deployments/{id}/rollback-plan", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetDeploymentRollbackPlan(deploymentService)))
		api.Post("/deployments/{id}/rollback", apimiddleware.RequirePermission(authService, "deployment.create", handlers.RespondError, handlers.RollbackDeploymentRun(deploymentService)))
		api.Get("/deployments/{id}/logs", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetDeploymentLogs(deploymentService)))
		api.Get("/deployments/{id}/events", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetDeploymentEvents(deploymentService)))
		api.Get("/deployments/{id}/timeline", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetDeploymentTimeline(deploymentService)))
		api.Get("/deployments/{id}/security", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetDeploymentSecurity(deploymentService)))
		api.Post("/deployments/{id}/cancel", apimiddleware.RequirePermission(authService, "deployment.cancel", handlers.RespondError, handlers.CancelDeploymentRun(deploymentService)))
		api.Post("/deployments/{id}/resume", apimiddleware.RequirePermission(authService, "deployment.approve", handlers.RespondError, handlers.ResumeDeploymentRunAfterApproval(deploymentService)))
		api.Post("/deployments/{id}/sync", apimiddleware.RequirePermission(authService, "deployment.create", handlers.RespondError, handlers.SyncDeploymentArgoCD(deploymentService)))
		api.Get("/visualization", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.ListVisualizationSurfaces()))
		api.Get("/visualization/pipeline-runs/{id}/dag", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetVisualizationPipelineDAG(pipelineService)))
		api.Get("/visualization/pipeline-runs/{id}/timeline", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetVisualizationPipelineTimeline(pipelineService)))
		api.Get("/visualization/pipeline-runs/{id}/summary", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetVisualizationPipelineSummary(pipelineService)))
		api.Get("/visualization/deployments/{id}/timeline", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetVisualizationDeploymentTimeline(deploymentService)))
		api.Get("/visualization/deployments/{id}/resources", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetVisualizationDeploymentResources(deploymentService)))
		api.Get("/visualization/deployments/{id}/diff", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetVisualizationDeploymentDiff(deploymentService)))
		api.Get("/visualization/deployments/{id}/health", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetVisualizationDeploymentHealth(deploymentService)))
		api.Get("/visualization/releases/{id}/overview", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetVisualizationReleaseOverview(releaseService)))
		api.Get("/visualization/releases/executions/{id}/timeline", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetVisualizationReleaseExecutionTimeline(releaseService)))
		api.Get("/visualization/releases/executions/{id}/targets", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetVisualizationReleaseExecutionTargets(releaseService)))
		api.Get("/visualization/environments/{id}/topology", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetVisualizationEnvironmentTopology(deploymentService, releaseService)))
		api.Get("/visualization/runners/summary", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetVisualizationRunnerSummary(pipelineService)))
		api.Get("/visualization/security/summary", apimiddleware.RequirePermission(authService, "project.read", handlers.RespondError, handlers.GetVisualizationSecuritySummary(securityService)))
		api.Get("/visualization/audit/timeline", apimiddleware.RequirePermission(authService, "audit.read", handlers.RespondError, handlers.GetVisualizationAuditTimeline(pipelineService, deploymentService, releaseService, securityService)))

		for _, group := range placeholderGroups() {
			group := group
			api.Get(group.path, handlers.NotImplemented(group.name))
			api.Route(group.path, func(r chi.Router) {
				r.Get("/", handlers.NotImplemented(group.name))
			})
		}
	})

	return r
}

func rejectOversizedRequestBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength > handlers.MaxRequestBodyBytes {
			handlers.RespondError(w, r, http.StatusRequestEntityTooLarge, dto.ErrorResponse{
				Code:    "request_body_too_large",
				Message: "request body exceeds 4 MiB",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

type routeGroup struct {
	path string
	name string
}

func placeholderGroups() []routeGroup {
	return []routeGroup{}
}

func placeholderOperations() map[string]bool {
	return map[string]bool{}
}
