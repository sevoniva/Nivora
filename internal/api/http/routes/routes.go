package routes

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/sevoniva/nivora/internal/api/http/handlers"
	"github.com/sevoniva/nivora/internal/infra/config"
	artifactusecase "github.com/sevoniva/nivora/internal/usecase/artifact"
	credentialusecase "github.com/sevoniva/nivora/internal/usecase/credential"
	deploymentusecase "github.com/sevoniva/nivora/internal/usecase/deployment"
	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
	releaseorchestration "github.com/sevoniva/nivora/internal/usecase/releaseorchestration"
	securityusecase "github.com/sevoniva/nivora/internal/usecase/security"
	"github.com/sevoniva/nivora/internal/version"
)

func New(cfg config.Config, info version.Info, logger *slog.Logger, pipelineService *pipelineusecase.Service, deploymentService *deploymentusecase.Service, artifactService *artifactusecase.Service, releaseService *releaseorchestration.Service, securityService *securityusecase.Service, credentialService *credentialusecase.Service) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", handlers.Health)
	r.Get("/readyz", handlers.Ready)

	r.Route("/api/v1", func(api chi.Router) {
		api.Get("/version", handlers.Version(info))
		api.Get("/system/info", handlers.SystemInfo(cfg))
		api.Get("/pipeline-runs", handlers.ListPipelineRuns(pipelineService))
		api.Post("/pipeline-runs", handlers.CreatePipelineRun(pipelineService))
		api.Get("/pipeline-runs/{id}", handlers.GetPipelineRun(pipelineService))
		api.Get("/pipeline-runs/{id}/logs", handlers.GetPipelineRunLogs(pipelineService))
		api.Get("/pipeline-runs/{id}/events", handlers.GetPipelineRunEvents(pipelineService))
		api.Get("/pipeline-runs/{id}/timeline", handlers.GetPipelineRunTimeline(pipelineService))
		api.Post("/pipeline-runs/{id}/cancel", handlers.CancelPipelineRun(pipelineService))
		api.Get("/runners", handlers.ListRunners(pipelineService))
		api.Post("/runners/register", handlers.RegisterRunner(pipelineService))
		api.Get("/runners/{id}", handlers.GetRunner(pipelineService))
		api.Post("/runners/{id}/heartbeat", handlers.HeartbeatRunner(pipelineService))
		api.Post("/deployments/plan", handlers.PlanDeploymentRun(deploymentService))
		api.Post("/deployments/gitops/plan", handlers.PlanGitOpsDeployment(deploymentService))
		api.Post("/deployments/gitops", handlers.CreateDeploymentRun(deploymentService))
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
		api.Post("/secrets", handlers.CreateSecret(credentialService))
		api.Get("/secrets/refs", handlers.ListSecretRefs(credentialService))
		api.Delete("/secrets/{id}", handlers.DeleteSecret(credentialService))
		api.Post("/credentials", handlers.CreateCredential(credentialService))
		api.Get("/credentials", handlers.ListCredentials(credentialService))
		api.Get("/credentials/{id}", handlers.GetCredential(credentialService))
		api.Delete("/credentials/{id}", handlers.DeleteCredential(credentialService))
		api.Post("/credentials/{id}/validate", handlers.ValidateCredential(credentialService))
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
		{"/approvals", "approvals"},
		{"/policies", "policies"},
		{"/audit-logs", "audit logs"},
		{"/events", "events"},
		{"/logs", "logs"},
		{"/integrations", "integrations"},
		{"/visualization", "visualization"},
	}
}
