package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/sevoniva/nivora/internal/api/http/dto"
	domainapp "github.com/sevoniva/nivora/internal/domain/application"
	catalogusecase "github.com/sevoniva/nivora/internal/usecase/catalog"
	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
	workflowusecase "github.com/sevoniva/nivora/internal/usecase/workflow"
)

// gitlabPushEvent is the subset of GitLab's push webhook payload Nivora needs
// to trigger a workflow run. See https://docs.gitlab.com/user/project/integrations/webhook_events.html#push-events
type gitlabPushEvent struct {
	Ref     string `json:"ref"`
	Before  string `json:"before"`
	After   string `json:"after"`
	Project struct {
		ID            int    `json:"id"`
		Name          string `json:"name"`
		WebURL        string `json:"web_url"`
		GitHTTPURL    string `json:"git_http_url"`
		GitSSHURL     string `json:"git_ssh_url"`
		DefaultBranch string `json:"default_branch"`
	} `json:"project"`
	Repository struct {
		Name       string `json:"name"`
		URL        string `json:"url"`
		GitHTTPURL string `json:"git_http_url"`
		GitSSHURL  string `json:"git_ssh_url"`
	} `json:"repository"`
}

// GitLabWebhookConfig configures the GitLab webhook receiver.
type GitLabWebhookConfig struct {
	// Secret is the expected value of the X-Gitlab-Token header. If empty,
	// token validation is skipped (dev mode only; production must set a secret).
	Secret string
	// WorkflowContent is the Nivora Workflow YAML to run on each push event.
	// It is executed with Confirm=true and AllowPipelineRun=true. The workflow
	// receives the repository id, ref, and project id from the matched catalog
	// repository.
	WorkflowContent string
}

// GitLabWebhook handles POST /api/v1/webhooks/gitlab. It validates the
// X-Gitlab-Token header, parses the push event, matches the repository URL
// against the catalog, and queues a workflow run. The handler is read-only
// with respect to the webhook itself (it only queues work); the workflow run
// it triggers is governed by the existing workflow.run permission path.
func GitLabWebhook(catalog *catalogusecase.Service, workflows *workflowusecase.Service, pipelines *pipelineusecase.Service, cfg GitLabWebhookConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cfg.Secret != "" {
			if r.Header.Get("X-Gitlab-Token") != cfg.Secret {
				RespondError(w, r, http.StatusUnauthorized, dto.ErrorResponse{
					Code:    "invalid_webhook_token",
					Message: "X-Gitlab-Token header does not match configured secret",
					Path:    r.URL.Path,
				})
				return
			}
		}
		eventType := r.Header.Get("X-Gitlab-Event")
		if eventType != "" && eventType != "Push Hook" {
			RespondJSON(w, http.StatusOK, map[string]any{"status": "ignored", "reason": "event type not handled: " + eventType})
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_request", Message: "read body: " + err.Error(), Path: r.URL.Path})
			return
		}
		var event gitlabPushEvent
		if err := json.Unmarshal(body, &event); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_request", Message: "parse push event: " + err.Error(), Path: r.URL.Path})
			return
		}
		repoURL := firstNonEmpty(event.Repository.GitHTTPURL, event.Project.GitHTTPURL, event.Repository.URL, event.Project.WebURL)
		if repoURL == "" {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_request", Message: "push event missing repository url", Path: r.URL.Path})
			return
		}
		repository, err := findRepositoryByURL(r, catalog, repoURL)
		if err != nil {
			RespondError(w, r, http.StatusNotFound, dto.ErrorResponse{Code: "repository_not_found", Message: err.Error(), Path: r.URL.Path})
			return
		}
		ref := strings.TrimPrefix(event.Ref, "refs/heads/")
		if ref == "" {
			ref = repository.DefaultBranch
		}
		content := cfg.WorkflowContent
		if content == "" {
			RespondError(w, r, http.StatusUnprocessableEntity, dto.ErrorResponse{Code: "webhook_not_configured", Message: "webhook workflow content is not configured", Path: r.URL.Path})
			return
		}
		result, err := workflows.Run(r.Context(), workflowusecase.RunInput{
			Content:          content,
			RepositoryID:     repository.ID,
			ProjectID:        repository.ProjectID,
			Ref:              ref,
			ActorID:          "gitlab-webhook",
			Confirm:          true,
			AllowPipelineRun: true,
		}, pipelines)
		if err != nil {
			respondWorkflowError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusAccepted, map[string]any{
			"status":        "triggered",
			"repositoryId":  repository.ID,
			"ref":           ref,
			"workflowRunId": result.WorkflowRun.ID,
			"pipelineRunId": result.PipelineRun.Run.ID,
			"mutated":       false,
		})
	}
}

// findRepositoryByURL lists catalog repositories and returns the first whose
// URL matches the given GitLab URL. Catalog does not index by URL yet, so a
// linear scan is acceptable for the webhook foundation.
// ponytail: linear scan; add a URL index if catalog size makes this hot.
func findRepositoryByURL(r *http.Request, catalog *catalogusecase.Service, repoURL string) (domainapp.Repository, error) {
	repositories, err := catalog.ListRepositories(r.Context(), "")
	if err != nil {
		return domainapp.Repository{}, err
	}
	normalized := normalizeRepoURL(repoURL)
	for _, repository := range repositories {
		if normalizeRepoURL(repository.URL) == normalized {
			return repository, nil
		}
	}
	return domainapp.Repository{}, errors.New("no catalog repository matches url: " + repoURL)
}

// normalizeRepoURL strips trailing .git and slashes for URL comparison so that
// http://host/group/repo.git and http://host/group/repo match.
func normalizeRepoURL(raw string) string {
	s := strings.TrimSpace(raw)
	s = strings.TrimSuffix(s, ".git")
	s = strings.TrimSuffix(s, "/")
	return strings.ToLower(s)
}
