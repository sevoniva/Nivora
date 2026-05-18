package routes

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/sevoniva/nivora/internal/api/http/handlers"
	"github.com/sevoniva/nivora/internal/infra/config"
	artifactusecase "github.com/sevoniva/nivora/internal/usecase/artifact"
	deploymentusecase "github.com/sevoniva/nivora/internal/usecase/deployment"
	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
	"github.com/sevoniva/nivora/internal/version"
)

func New(cfg config.Config, info version.Info, logger *slog.Logger, pipelineService *pipelineusecase.Service, deploymentService *deploymentusecase.Service, artifactService *artifactusecase.Service) http.Handler {
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
		api.Post("/artifacts/inspect", handlers.InspectArtifact(artifactService))
		api.Post("/artifacts/resolve", handlers.ResolveArtifact(artifactService))
		api.Post("/artifact-registries/validate", handlers.ValidateArtifactRegistry())
		api.Get("/releases", handlers.ListReleases(artifactService))
		api.Post("/releases", handlers.CreateRelease(artifactService))
		api.Get("/releases/{id}", handlers.GetRelease(artifactService))
		api.Get("/releases/{id}/artifacts", handlers.GetReleaseArtifacts(artifactService))
		api.Get("/deployments", handlers.ListDeploymentRuns(deploymentService))
		api.Post("/deployments", handlers.CreateDeploymentRun(deploymentService))
		api.Get("/deployments/{id}", handlers.GetDeploymentRun(deploymentService))
		api.Get("/deployments/{id}/plan", handlers.GetDeploymentPlan(deploymentService))
		api.Get("/deployments/{id}/gitops-plan", handlers.GetDeploymentGitOpsPlan(deploymentService))
		api.Get("/deployments/{id}/resources", handlers.GetDeploymentResources(deploymentService))
		api.Get("/deployments/{id}/health", handlers.GetDeploymentHealth(deploymentService))
		api.Get("/deployments/{id}/diff", handlers.GetDeploymentRuntimeDiff(deploymentService))
		api.Get("/deployments/{id}/manifest-snapshot", handlers.GetDeploymentManifestSnapshot(deploymentService))
		api.Get("/deployments/{id}/rollback-plan", handlers.GetDeploymentRollbackPlan(deploymentService))
		api.Get("/deployments/{id}/logs", handlers.GetDeploymentLogs(deploymentService))
		api.Get("/deployments/{id}/events", handlers.GetDeploymentEvents(deploymentService))
		api.Get("/deployments/{id}/timeline", handlers.GetDeploymentTimeline(deploymentService))
		api.Post("/deployments/{id}/cancel", handlers.CancelDeploymentRun(deploymentService))

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
