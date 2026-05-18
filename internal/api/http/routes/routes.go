package routes

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/sevoniva/nivora/internal/api/http/handlers"
	"github.com/sevoniva/nivora/internal/infra/config"
	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
	"github.com/sevoniva/nivora/internal/version"
)

func New(cfg config.Config, info version.Info, logger *slog.Logger, pipelineService *pipelineusecase.Service) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", handlers.Health)
	r.Get("/readyz", handlers.Ready)

	r.Route("/api/v1", func(api chi.Router) {
		api.Get("/version", handlers.Version(info))
		api.Get("/system/info", handlers.SystemInfo(cfg))
		api.Post("/pipeline-runs", handlers.CreatePipelineRun(pipelineService))
		api.Get("/pipeline-runs/{id}", handlers.GetPipelineRun(pipelineService))
		api.Get("/pipeline-runs/{id}/logs", handlers.GetPipelineRunLogs(pipelineService))
		api.Get("/pipeline-runs/{id}/events", handlers.GetPipelineRunEvents(pipelineService))

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
		{"/releases", "releases"},
		{"/deployments", "deployments"},
		{"/runners", "runners"},
		{"/approvals", "approvals"},
		{"/policies", "policies"},
		{"/audit-logs", "audit logs"},
		{"/events", "events"},
		{"/logs", "logs"},
		{"/integrations", "integrations"},
		{"/visualization", "visualization"},
	}
}
