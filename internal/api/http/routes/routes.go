package routes

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/sevoniva/nivora/internal/api/http/handlers"
	apimiddleware "github.com/sevoniva/nivora/internal/api/http/middleware"
	"github.com/sevoniva/nivora/internal/infra/config"
	approvalusecase "github.com/sevoniva/nivora/internal/usecase/approval"
	artifactusecase "github.com/sevoniva/nivora/internal/usecase/artifact"
	authusecase "github.com/sevoniva/nivora/internal/usecase/auth"
	cloudusecase "github.com/sevoniva/nivora/internal/usecase/cloud"
	credentialusecase "github.com/sevoniva/nivora/internal/usecase/credential"
	deploymentusecase "github.com/sevoniva/nivora/internal/usecase/deployment"
	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
	releaseorchestration "github.com/sevoniva/nivora/internal/usecase/releaseorchestration"
	securityusecase "github.com/sevoniva/nivora/internal/usecase/security"
	"github.com/sevoniva/nivora/internal/version"
)

func New(cfg config.Config, info version.Info, logger *slog.Logger, pipelineService *pipelineusecase.Service, deploymentService *deploymentusecase.Service, artifactService *artifactusecase.Service, releaseService *releaseorchestration.Service, securityService *securityusecase.Service, credentialService *credentialusecase.Service, authService *authusecase.Service, approvalService *approvalusecase.Service, cloudService *cloudusecase.Service) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(apimiddleware.RequestContext())
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(apimiddleware.StructuredAccessLog(logger))

	r.Get("/healthz", handlers.Health)
	r.Get("/readyz", handlers.Ready)
	r.Get("/metrics", handlers.Metrics())

	r.Route("/api/v1", func(api chi.Router) {
		api.Use(apimiddleware.Authenticate(cfg.Auth, authService, handlers.RespondError))
		api.Get("/version", handlers.Version(info))
		api.Get("/system/info", handlers.SystemInfo(cfg))
		api.Get("/system/runtime", handlers.SystemRuntime(cfg))
		api.Get("/system/diagnostics", handlers.SystemDiagnostics(cfg))
		api.Get("/auth/whoami", handlers.WhoAmI())
		api.Get("/auth/permissions", handlers.AuthPermissions(authService))
		api.Get("/auth/token-info", handlers.AuthTokenInfo())
		api.Get("/users", handlers.ListUsers(authService))
		api.Get("/roles", handlers.ListRoles(authService))
		api.Get("/permissions", handlers.ListPermissions(authService))
		api.Get("/orgs/{id}/members", handlers.ListOrgMembers(authService))
		api.Post("/orgs/{id}/members", apimiddleware.RequirePermission(authService, "project.write", handlers.RespondError, handlers.AddOrgMember(authService)))
		api.Get("/projects/{id}/members", handlers.ListProjectMembers(authService))
		api.Post("/projects/{id}/members", apimiddleware.RequirePermission(authService, "project.write", handlers.RespondError, handlers.AddProjectMember(authService)))
		api.Get("/pipeline-runs", handlers.ListPipelineRuns(pipelineService))
		api.Post("/pipeline-runs", handlers.CreatePipelineRun(pipelineService))
		api.Get("/pipeline-runs/{id}", handlers.GetPipelineRun(pipelineService))
		api.Get("/pipeline-runs/{id}/logs", handlers.GetPipelineRunLogs(pipelineService))
		api.Get("/pipeline-runs/{id}/events", handlers.GetPipelineRunEvents(pipelineService))
		api.Get("/pipeline-runs/{id}/timeline", handlers.GetPipelineRunTimeline(pipelineService))
		api.Post("/pipeline-runs/{id}/cancel", handlers.CancelPipelineRun(pipelineService))
		api.Post("/pipeline-runs/{id}/cancel-request", handlers.RequestPipelineRunCancel(pipelineService))
		api.Get("/runners", handlers.ListRunners(pipelineService))
		api.Post("/runners/register", handlers.RegisterRunner(pipelineService))
		api.Get("/runners/{id}", handlers.GetRunner(pipelineService))
		api.Post("/runners/{id}/heartbeat", handlers.HeartbeatRunner(pipelineService))
		api.Post("/runners/{id}/jobs/claim", handlers.ClaimRunnerJob(pipelineService))
		api.Post("/jobs/{id}/logs", handlers.AppendJobLogs(pipelineService))
		api.Post("/jobs/{id}/status", handlers.UpdateJobStatus(pipelineService))
		api.Post("/deployments/plan", handlers.PlanDeploymentRun(deploymentService))
		api.Post("/deployments/gitops/plan", handlers.PlanGitOpsDeployment(deploymentService))
		api.Post("/deployments/gitops", handlers.CreateDeploymentRun(deploymentService))
		api.Get("/host-groups", handlers.ListHostGroups(deploymentService))
		api.Post("/host-groups", handlers.CreateHostGroup(deploymentService))
		api.Get("/host-groups/{id}", handlers.GetHostGroup(deploymentService))
		api.Post("/deployments/host/plan", handlers.PlanHostDeployment(deploymentService))
		api.Get("/integrations/argocd/applications/{name}/status", handlers.GetArgoCDApplicationStatus(deploymentService))
		api.Get("/integrations/argocd/applications/{name}/resources", handlers.GetArgoCDApplicationResources(deploymentService))
		api.Post("/integrations/argocd/applications/{name}/sync", handlers.SyncArgoCDApplication(deploymentService))
		api.Post("/artifacts/inspect", handlers.InspectArtifact(artifactService))
		api.Post("/artifacts/resolve", handlers.ResolveArtifact(artifactService))
		api.Post("/artifact-registries/validate", handlers.ValidateArtifactRegistry())
		api.Post("/security/scans", handlers.CreateSecurityScan(securityService))
		api.Get("/security/scans/{id}", handlers.GetSecurityScan(securityService))
		api.Get("/security/scans/{id}/findings", handlers.GetSecurityFindings(securityService))
		api.Post("/policies/evaluate", handlers.EvaluatePolicy(securityService))
		api.Post("/approvals", handlers.CreateApproval(approvalService))
		api.Get("/approvals", handlers.ListApprovals(approvalService))
		api.Get("/approvals/{id}", handlers.GetApproval(approvalService))
		api.Post("/approvals/{id}/approve", handlers.ApproveApproval(approvalService))
		api.Post("/approvals/{id}/reject", handlers.RejectApproval(approvalService))
		api.Post("/approvals/{id}/cancel", handlers.CancelApproval(approvalService))
		api.Get("/change-windows", handlers.ListChangeWindows(approvalService))
		api.Post("/change-windows", handlers.CreateChangeWindow(approvalService))
		api.Get("/change-windows/{id}", handlers.GetChangeWindow(approvalService))
		api.Post("/change-windows/evaluate", handlers.EvaluateChangeWindow(approvalService))
		api.Get("/notifications", handlers.ListNotifications(approvalService))
		api.Post("/notifications/test", handlers.TestNotification(approvalService))
		api.Get("/cloud/providers", handlers.ListCloudProviders(cloudService))
		api.Post("/cloud/accounts", handlers.CreateCloudAccount(cloudService))
		api.Get("/cloud/accounts", handlers.ListCloudAccounts(cloudService))
		api.Get("/cloud/accounts/{id}", handlers.GetCloudAccount(cloudService))
		api.Post("/cloud/accounts/{id}/validate", handlers.ValidateCloudAccount(cloudService))
		api.Get("/cloud/accounts/{id}/regions", handlers.ListCloudRegions(cloudService))
		api.Get("/cloud/accounts/{id}/clusters", handlers.ListCloudClusters(cloudService))
		api.Get("/cloud/accounts/{id}/hosts", handlers.ListCloudHosts(cloudService))
		api.Get("/cloud/accounts/{id}/registries", handlers.ListCloudRegistries(cloudService))
		api.Get("/cloud/accounts/{id}/inventory", handlers.GetCloudInventory(cloudService))
		api.Post("/secrets", apimiddleware.RequirePermission(authService, "credential.manage", handlers.RespondError, handlers.CreateSecret(credentialService)))
		api.Get("/secrets/refs", apimiddleware.RequirePermission(authService, "credential.manage", handlers.RespondError, handlers.ListSecretRefs(credentialService)))
		api.Delete("/secrets/{id}", apimiddleware.RequirePermission(authService, "credential.manage", handlers.RespondError, handlers.DeleteSecret(credentialService)))
		api.Post("/credentials", apimiddleware.RequirePermission(authService, "credential.manage", handlers.RespondError, handlers.CreateCredential(credentialService)))
		api.Get("/credentials", apimiddleware.RequirePermission(authService, "credential.manage", handlers.RespondError, handlers.ListCredentials(credentialService)))
		api.Get("/credentials/{id}", apimiddleware.RequirePermission(authService, "credential.manage", handlers.RespondError, handlers.GetCredential(credentialService)))
		api.Delete("/credentials/{id}", apimiddleware.RequirePermission(authService, "credential.manage", handlers.RespondError, handlers.DeleteCredential(credentialService)))
		api.Post("/credentials/{id}/validate", apimiddleware.RequirePermission(authService, "credential.manage", handlers.RespondError, handlers.ValidateCredential(credentialService)))
		api.Get("/releases", handlers.ListReleases(artifactService))
		api.Post("/releases", handlers.CreateRelease(artifactService))
		api.Post("/releases/{id}/plan", handlers.PlanRelease(releaseService))
		api.Post("/releases/{id}/deploy", handlers.DeployRelease(releaseService))
		api.Get("/releases/{id}/plan", handlers.GetReleasePlan(releaseService))
		api.Get("/releases/{id}/executions", handlers.ListReleaseExecutions(releaseService))
		api.Get("/releases/{id}/security", handlers.GetReleaseSecurity(releaseService))
		api.Get("/releases/{id}", handlers.GetRelease(artifactService))
		api.Get("/releases/{id}/artifacts", handlers.GetReleaseArtifacts(artifactService))
		api.Get("/releases/executions/{execution_id}", handlers.GetReleaseExecution(releaseService))
		api.Get("/releases/executions/{execution_id}/timeline", handlers.GetReleaseExecutionTimeline(releaseService))
		api.Get("/releases/executions/{execution_id}/targets", handlers.GetReleaseExecutionTargets(releaseService))
		api.Post("/releases/executions/{execution_id}/cancel", handlers.CancelReleaseExecution(releaseService))
		api.Get("/deployments", handlers.ListDeploymentRuns(deploymentService))
		api.Post("/deployments", handlers.CreateDeploymentRun(deploymentService))
		api.Get("/deployments/{id}", handlers.GetDeploymentRun(deploymentService))
		api.Get("/deployments/{id}/plan", handlers.GetDeploymentPlan(deploymentService))
		api.Get("/deployments/{id}/gitops-plan", handlers.GetDeploymentGitOpsPlan(deploymentService))
		api.Get("/deployments/{id}/argocd-status", handlers.GetDeploymentArgoCDStatus(deploymentService))
		api.Get("/deployments/{id}/resources", handlers.GetDeploymentResources(deploymentService))
		api.Get("/deployments/{id}/hosts", handlers.GetDeploymentHosts(deploymentService))
		api.Get("/deployments/{id}/health", handlers.GetDeploymentHealth(deploymentService))
		api.Get("/deployments/{id}/diff", handlers.GetDeploymentRuntimeDiff(deploymentService))
		api.Get("/deployments/{id}/manifest-snapshot", handlers.GetDeploymentManifestSnapshot(deploymentService))
		api.Get("/deployments/{id}/rollback-plan", handlers.GetDeploymentRollbackPlan(deploymentService))
		api.Get("/deployments/{id}/logs", handlers.GetDeploymentLogs(deploymentService))
		api.Get("/deployments/{id}/events", handlers.GetDeploymentEvents(deploymentService))
		api.Get("/deployments/{id}/timeline", handlers.GetDeploymentTimeline(deploymentService))
		api.Get("/deployments/{id}/security", handlers.GetDeploymentSecurity(deploymentService))
		api.Post("/deployments/{id}/cancel", handlers.CancelDeploymentRun(deploymentService))
		api.Post("/deployments/{id}/sync", handlers.SyncDeploymentArgoCD(deploymentService))
		api.Get("/visualization/pipeline-runs/{id}/dag", handlers.GetVisualizationPipelineDAG(pipelineService))
		api.Get("/visualization/pipeline-runs/{id}/timeline", handlers.GetVisualizationPipelineTimeline(pipelineService))
		api.Get("/visualization/pipeline-runs/{id}/summary", handlers.GetVisualizationPipelineSummary(pipelineService))
		api.Get("/visualization/deployments/{id}/timeline", handlers.GetVisualizationDeploymentTimeline(deploymentService))
		api.Get("/visualization/deployments/{id}/resources", handlers.GetVisualizationDeploymentResources(deploymentService))
		api.Get("/visualization/deployments/{id}/diff", handlers.GetVisualizationDeploymentDiff(deploymentService))
		api.Get("/visualization/deployments/{id}/health", handlers.GetVisualizationDeploymentHealth(deploymentService))
		api.Get("/visualization/releases/{id}/overview", handlers.GetVisualizationReleaseOverview(releaseService))
		api.Get("/visualization/releases/executions/{id}/timeline", handlers.GetVisualizationReleaseExecutionTimeline(releaseService))
		api.Get("/visualization/releases/executions/{id}/targets", handlers.GetVisualizationReleaseExecutionTargets(releaseService))
		api.Get("/visualization/environments/{id}/topology", handlers.GetVisualizationEnvironmentTopology(deploymentService, releaseService))
		api.Get("/visualization/runners/summary", handlers.GetVisualizationRunnerSummary(pipelineService))
		api.Get("/visualization/security/summary", handlers.GetVisualizationSecuritySummary(securityService))
		api.Get("/visualization/audit/timeline", handlers.GetVisualizationAuditTimeline(pipelineService, deploymentService, releaseService, securityService))

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

type routeGroup struct {
	path string
	name string
}

func placeholderGroups() []routeGroup {
	return []routeGroup{
		{"/orgs", "orgs"},
		{"/projects", "projects"},
		{"/applications", "applications"},
		{"/environments", "environments"},
		{"/repositories", "repositories"},
		{"/artifact-registries", "artifact registries"},
		{"/pipelines", "pipelines"},
		{"/policies", "policies"},
		{"/audit-logs", "audit logs"},
		{"/events", "events"},
		{"/logs", "logs"},
		{"/integrations", "integrations"},
		{"/visualization", "visualization"},
	}
}
