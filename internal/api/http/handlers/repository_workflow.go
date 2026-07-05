package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/sevoniva/nivora/internal/api/http/dto"
	domainapp "github.com/sevoniva/nivora/internal/domain/application"
	catalogusecase "github.com/sevoniva/nivora/internal/usecase/catalog"
	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
	repositoryusecase "github.com/sevoniva/nivora/internal/usecase/repository"
	workflowusecase "github.com/sevoniva/nivora/internal/usecase/workflow"
)

type repositorySnapshotRequest struct {
	Ref       string `json:"ref,omitempty"`
	LocalPath string `json:"localPath,omitempty"`
}

type workflowDefinitionRequest struct {
	Content              string `json:"content"`
	RepositoryID         string `json:"repositoryId,omitempty"`
	RepositorySnapshotID string `json:"repositorySnapshotId,omitempty"`
	Path                 string `json:"path,omitempty"`
	Ref                  string `json:"ref,omitempty"`
}

type workflowDefinitionPayload struct {
	Content              string
	Definition           workflowusecase.Definition
	RepositoryID         string
	RepositorySnapshotID string
	Path                 string
	Ref                  string
}

type workflowRunRequest struct {
	Content              string `json:"content,omitempty"`
	PlanID               string `json:"planId,omitempty"`
	RepositoryID         string `json:"repositoryId,omitempty"`
	RepositorySnapshotID string `json:"repositorySnapshotId,omitempty"`
	Path                 string `json:"path,omitempty"`
	Ref                  string `json:"ref,omitempty"`
	ProjectID            string `json:"projectId,omitempty"`
	EnvironmentID        string `json:"environmentId,omitempty"`
	CorrelationID        string `json:"correlationId,omitempty"`
	Confirm              bool   `json:"confirm"`
	AllowPipelineRun     bool   `json:"allowPipelineRun"`
}

type workflowReconcileRequest struct {
	RepositoryID string `json:"repositoryId,omitempty"`
	WorkflowID   string `json:"workflowId,omitempty"`
	ProjectID    string `json:"projectId,omitempty"`
	Status       string `json:"status,omitempty"`
	Limit        int    `json:"limit,omitempty"`
	Offset       int    `json:"offset,omitempty"`
}

type workflowRetryRequest struct {
	CorrelationID    string `json:"correlationId,omitempty"`
	Confirm          bool   `json:"confirm"`
	AllowPipelineRun bool   `json:"allowPipelineRun"`
}

type devOpsPlanRequest struct {
	RepositoryID string `json:"repositoryId"`
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

func PlanRepositoryDevOps(repositories *repositoryusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input devOpsPlanRequest
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil && !errors.Is(err, io.EOF) {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_devops_plan_request", Message: err.Error(), Path: r.URL.Path})
			return
		}
		repositoryID := strings.TrimSpace(input.RepositoryID)
		if repositoryID == "" {
			repositoryID = strings.TrimSpace(r.URL.Query().Get("repositoryId"))
		}
		if repositoryID == "" {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_devops_plan_request", Message: "repositoryId is required", Path: r.URL.Path})
			return
		}
		plan, err := repositories.DevOpsPlan(r.Context(), repositoryID)
		if err != nil {
			respondRepositoryError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, map[string]any{"plan": plan, "mutated": false})
	}
}

func ReviewRepositoryDevOpsReadiness(repositories *repositoryusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input devOpsPlanRequest
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil && !errors.Is(err, io.EOF) {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_devops_readiness_review_request", Message: err.Error(), Path: r.URL.Path})
			return
		}
		repositoryID := strings.TrimSpace(input.RepositoryID)
		if repositoryID == "" {
			repositoryID = strings.TrimSpace(r.URL.Query().Get("repositoryId"))
		}
		if repositoryID == "" {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_devops_readiness_review_request", Message: "repositoryId is required", Path: r.URL.Path})
			return
		}
		review, err := repositories.DevOpsReadinessReview(r.Context(), repositoryID)
		if err != nil {
			respondRepositoryError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, map[string]any{"review": review, "mutated": false})
	}
}

func ListWorkflows(service *workflowusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		workflows, err := service.ListWorkflows(r.Context(), workflowusecase.PlanListFilter{
			RepositoryID: r.URL.Query().Get("repositoryId"),
			WorkflowID:   r.URL.Query().Get("workflowId"),
		})
		if err != nil {
			respondWorkflowError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, map[string]any{"workflows": workflows})
	}
}

func GetWorkflow(service *workflowusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		workflow, err := service.GetWorkflow(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			respondWorkflowError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, workflow)
	}
}

func GetWorkflowLatestPlan(service *workflowusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		record, err := service.GetLatestPlan(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			respondWorkflowError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, record)
	}
}

func ValidateWorkflowDefinition(service *workflowusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		payload, err := workflowDefinitionFromRequest(r)
		if err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_workflow_request", Message: err.Error(), Path: r.URL.Path})
			return
		}
		plan, err := service.Validate(r.Context(), workflowusecase.PlanInput{
			Content:              payload.Content,
			RepositoryID:         payload.RepositoryID,
			RepositorySnapshotID: payload.RepositorySnapshotID,
			Path:                 payload.Path,
			Ref:                  payload.Ref,
		})
		if err != nil {
			RespondJSON(w, http.StatusBadRequest, map[string]any{"valid": false, "error": err.Error()})
			return
		}
		RespondJSON(w, http.StatusOK, map[string]any{"valid": true, "plan": plan})
	}
}

func PlanWorkflowDefinition(service *workflowusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		payload, err := workflowDefinitionFromRequest(r)
		if err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_workflow_request", Message: err.Error(), Path: r.URL.Path})
			return
		}
		record, err := service.Plan(r.Context(), workflowusecase.PlanInput{
			Content:              payload.Content,
			RepositoryID:         payload.RepositoryID,
			RepositorySnapshotID: payload.RepositorySnapshotID,
			Path:                 payload.Path,
			Ref:                  payload.Ref,
		})
		if err != nil {
			respondWorkflowError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, record.Plan)
	}
}

func ListWorkflowPlans(service *workflowusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		plans, err := service.ListPlans(r.Context(), workflowusecase.PlanListFilter{
			RepositoryID: r.URL.Query().Get("repositoryId"),
			WorkflowID:   r.URL.Query().Get("workflowId"),
			Limit:        limit,
			Offset:       offset,
		})
		if err != nil {
			respondWorkflowError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, map[string]any{"plans": plans})
	}
}

func GetWorkflowPlan(service *workflowusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		record, err := service.GetPlan(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			respondWorkflowError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, record)
	}
}

func ListWorkflowRuns(service *workflowusecase.Service, pipelines *pipelineusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		runs, err := service.RefreshRuns(r.Context(), workflowusecase.RunListFilter{
			RepositoryID: r.URL.Query().Get("repositoryId"),
			WorkflowID:   r.URL.Query().Get("workflowId"),
			ProjectID:    r.URL.Query().Get("projectId"),
			Status:       workflowusecase.RunStatus(r.URL.Query().Get("status")),
			Limit:        limit,
			Offset:       offset,
		}, pipelines)
		if err != nil {
			respondWorkflowError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, map[string]any{"workflowRuns": runs})
	}
}

func GetWorkflowRun(service *workflowusecase.Service, pipelines *pipelineusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		record, err := service.RefreshRunStatus(r.Context(), chi.URLParam(r, "id"), pipelines)
		if err != nil {
			respondWorkflowError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, record)
	}
}

func CancelWorkflowRun(service *workflowusecase.Service, pipelines *pipelineusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		record, err := service.CancelRun(r.Context(), chi.URLParam(r, "id"), actorIDFromRequest(r), pipelines)
		if err != nil {
			respondWorkflowError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, record)
	}
}

func ReconcileWorkflowRuns(service *workflowusecase.Service, pipelines *pipelineusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input workflowReconcileRequest
		if r.Body != nil {
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil && !errors.Is(err, io.EOF) {
				RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_workflow_reconcile_request", Message: err.Error(), Path: r.URL.Path})
				return
			}
		}
		if input.RepositoryID == "" {
			input.RepositoryID = r.URL.Query().Get("repositoryId")
		}
		if input.WorkflowID == "" {
			input.WorkflowID = r.URL.Query().Get("workflowId")
		}
		if input.ProjectID == "" {
			input.ProjectID = r.URL.Query().Get("projectId")
		}
		if input.Status == "" {
			input.Status = r.URL.Query().Get("status")
		}
		if input.Limit == 0 {
			input.Limit, _ = strconv.Atoi(r.URL.Query().Get("limit"))
		}
		if input.Offset == 0 {
			input.Offset, _ = strconv.Atoi(r.URL.Query().Get("offset"))
		}
		result, err := service.ReconcileRuns(r.Context(), workflowusecase.RunListFilter{
			RepositoryID: input.RepositoryID,
			WorkflowID:   input.WorkflowID,
			ProjectID:    input.ProjectID,
			Status:       workflowusecase.RunStatus(input.Status),
			Limit:        input.Limit,
			Offset:       input.Offset,
		}, pipelines)
		if err != nil {
			respondWorkflowError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, result)
	}
}

func RetryWorkflowRun(service *workflowusecase.Service, pipelines *pipelineusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input workflowRetryRequest
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil && !errors.Is(err, io.EOF) {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_workflow_retry_request", Message: err.Error(), Path: r.URL.Path})
			return
		}
		result, err := service.RetryRun(r.Context(), chi.URLParam(r, "id"), workflowusecase.RetryInput{
			ActorID:          actorIDFromRequest(r),
			CorrelationID:    input.CorrelationID,
			Confirm:          input.Confirm,
			AllowPipelineRun: input.AllowPipelineRun,
		}, pipelines)
		if err != nil {
			respondWorkflowError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusAccepted, result)
	}
}

func RunWorkflowDefinition(service *workflowusecase.Service, pipelines *pipelineusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input workflowRunRequest
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil && !errors.Is(err, io.EOF) {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_workflow_run_request", Message: err.Error(), Path: r.URL.Path})
			return
		}
		result, err := service.Run(r.Context(), workflowusecase.RunInput{
			Content:              input.Content,
			PlanID:               input.PlanID,
			RepositoryID:         input.RepositoryID,
			RepositorySnapshotID: input.RepositorySnapshotID,
			Path:                 input.Path,
			Ref:                  input.Ref,
			ProjectID:            input.ProjectID,
			EnvironmentID:        input.EnvironmentID,
			ActorID:              actorIDFromRequest(r),
			CorrelationID:        input.CorrelationID,
			Confirm:              input.Confirm,
			AllowPipelineRun:     input.AllowPipelineRun,
		}, pipelines)
		if err != nil {
			respondWorkflowError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusAccepted, result)
	}
}

func actorIDFromRequest(r *http.Request) string {
	if value := strings.TrimSpace(r.Header.Get("X-Nivora-Actor")); value != "" {
		return value
	}
	return "api"
}

func workflowDefinitionFromRequest(r *http.Request) (workflowDefinitionPayload, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return workflowDefinitionPayload{}, err
	}
	content := strings.TrimSpace(string(body))
	payload := workflowDefinitionPayload{Content: content}
	if strings.HasPrefix(content, "{") {
		var input workflowDefinitionRequest
		if err := json.Unmarshal(body, &input); err != nil {
			return workflowDefinitionPayload{}, err
		}
		content = strings.TrimSpace(input.Content)
		payload = workflowDefinitionPayload{
			Content:              content,
			RepositoryID:         strings.TrimSpace(input.RepositoryID),
			RepositorySnapshotID: strings.TrimSpace(input.RepositorySnapshotID),
			Path:                 strings.TrimSpace(input.Path),
			Ref:                  strings.TrimSpace(input.Ref),
		}
	}
	if content == "" {
		return workflowDefinitionPayload{}, errors.New("workflow content is required")
	}
	def, err := workflowusecase.ParseDefinition([]byte(content))
	if err != nil {
		return workflowDefinitionPayload{}, err
	}
	payload.Definition = def
	return payload, nil
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

func respondWorkflowError(w http.ResponseWriter, r *http.Request, err error) {
	status := http.StatusBadRequest
	code := "invalid_workflow_definition"
	if errors.Is(err, workflowusecase.ErrNotFound) {
		status = http.StatusNotFound
		code = "workflow_plan_not_found"
	}
	if errors.Is(err, workflowusecase.ErrRunTerminal) {
		status = http.StatusConflict
		code = "workflow_run_terminal"
	}
	if errors.Is(err, workflowusecase.ErrRunNotRetryable) {
		status = http.StatusConflict
		code = "workflow_run_not_retryable"
	}
	if errors.Is(err, workflowusecase.ErrInvalid) {
		code = "invalid_workflow_definition"
	}
	RespondError(w, r, status, dto.ErrorResponse{Code: code, Message: err.Error(), Path: r.URL.Path})
}
