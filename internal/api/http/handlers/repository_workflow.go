package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/sevoniva/nivora/internal/api/http/dto"
	domainapp "github.com/sevoniva/nivora/internal/domain/application"
	catalogusecase "github.com/sevoniva/nivora/internal/usecase/catalog"
	repositoryusecase "github.com/sevoniva/nivora/internal/usecase/repository"
	workflowusecase "github.com/sevoniva/nivora/internal/usecase/workflow"
)

type repositorySnapshotRequest struct {
	Ref       string `json:"ref,omitempty"`
	LocalPath string `json:"localPath,omitempty"`
}

type workflowDefinitionRequest struct {
	Content string `json:"content"`
}

func CreateRepositorySnapshot(catalog *catalogusecase.Service, repositories *repositoryusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input repositorySnapshotRequest
		if r.Body != nil {
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil && !errors.Is(err, io.EOF) {
				RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_repository_snapshot_request", Message: err.Error(), Path: r.URL.Path})
				return
			}
		}
		repository, err := catalog.GetRepository(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			respondCatalogError(w, r, err)
			return
		}
		snapshot, err := repositories.CreateSnapshot(r.Context(), repositoryusecase.SnapshotInput{
			Repository: toRepositoryUsecase(repository),
			Ref:        input.Ref,
			LocalPath:  input.LocalPath,
		})
		if err != nil {
			respondRepositoryError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusCreated, snapshot)
	}
}

func ListRepositorySnapshots(repositories *repositoryusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		snapshots, err := repositories.ListSnapshots(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			respondRepositoryError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, map[string]any{"snapshots": snapshots})
	}
}

func GetRepositoryIntelligence(repositories *repositoryusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		snapshot, err := repositories.GetLatestSnapshot(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			respondRepositoryError(w, r, err)
			return
		}
		intelligence, err := repositories.GetIntelligence(r.Context(), snapshot.RepositoryID, snapshot.ID)
		if err != nil {
			respondRepositoryError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, intelligence)
	}
}

func AnalyzeRepository(repositories *repositoryusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		intelligence, err := repositories.AnalyzeLatest(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			respondRepositoryError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, intelligence)
	}
}

func ValidateWorkflowDefinition() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		def, err := workflowDefinitionFromRequest(r)
		if err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_workflow_request", Message: err.Error(), Path: r.URL.Path})
			return
		}
		plan, err := workflowusecase.PlanDefinition(def, workflowusecase.PlanOptions{})
		if err != nil {
			RespondJSON(w, http.StatusBadRequest, map[string]any{"valid": false, "error": err.Error()})
			return
		}
		RespondJSON(w, http.StatusOK, map[string]any{"valid": true, "plan": plan})
	}
}

func PlanWorkflowDefinition() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		def, err := workflowDefinitionFromRequest(r)
		if err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_workflow_request", Message: err.Error(), Path: r.URL.Path})
			return
		}
		plan, err := workflowusecase.PlanDefinition(def, workflowusecase.PlanOptions{})
		if err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_workflow_definition", Message: err.Error(), Path: r.URL.Path})
			return
		}
		RespondJSON(w, http.StatusOK, plan)
	}
}

func WorkflowRunGuardedPlaceholder() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		RespondError(w, r, http.StatusNotImplemented, dto.ErrorResponse{
			Code:    "not_implemented",
			Message: "workflow run is a guarded future capability; use /api/v1/workflows/validate or /api/v1/workflows/plan in this phase",
			Path:    r.URL.Path,
		})
	}
}

func workflowDefinitionFromRequest(r *http.Request) (workflowusecase.Definition, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return workflowusecase.Definition{}, err
	}
	content := strings.TrimSpace(string(body))
	if strings.HasPrefix(content, "{") {
		var input workflowDefinitionRequest
		if err := json.Unmarshal(body, &input); err != nil {
			return workflowusecase.Definition{}, err
		}
		content = strings.TrimSpace(input.Content)
	}
	if content == "" {
		return workflowusecase.Definition{}, errors.New("workflow content is required")
	}
	return workflowusecase.ParseDefinition([]byte(content))
}

func toRepositoryUsecase(repository domainapp.Repository) repositoryusecase.Repository {
	status := repositoryusecase.RepositoryStatusActive
	if !repository.Enabled {
		status = repositoryusecase.RepositoryStatusDisabled
	}
	provider := repositoryusecase.Provider(repository.Provider)
	if provider == "" || provider == "generic" {
		provider = repositoryusecase.ProviderGenericGit
	}
	return repositoryusecase.Repository{
		ID:            repository.ID,
		Name:          repository.Name,
		Provider:      provider,
		URL:           repository.URL,
		DefaultBranch: repository.DefaultBranch,
		CredentialRef: repository.CredentialRef,
		ProjectID:     repository.ProjectID,
		Labels:        repository.Labels,
		Metadata:      repository.Metadata,
		Status:        status,
		CreatedAt:     repository.CreatedAt,
		UpdatedAt:     repository.UpdatedAt,
	}
}

func respondRepositoryError(w http.ResponseWriter, r *http.Request, err error) {
	status := http.StatusBadRequest
	code := "repository_error"
	if errors.Is(err, repositoryusecase.ErrNotFound) {
		status = http.StatusNotFound
		code = "repository_record_not_found"
	}
	if errors.Is(err, repositoryusecase.ErrInvalid) {
		code = "invalid_repository_request"
	}
	RespondError(w, r, status, dto.ErrorResponse{Code: code, Message: err.Error(), Path: r.URL.Path})
}
