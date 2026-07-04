package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/sevoniva/nivora/internal/api/http/dto"
	apimiddleware "github.com/sevoniva/nivora/internal/api/http/middleware"
	domainapproval "github.com/sevoniva/nivora/internal/domain/approval"
	domaindeployment "github.com/sevoniva/nivora/internal/domain/deployment"
	"github.com/sevoniva/nivora/internal/domain/tenant"
	"github.com/sevoniva/nivora/internal/infra/telemetry"
	portargocd "github.com/sevoniva/nivora/internal/ports/argocd"
	deploymentusecase "github.com/sevoniva/nivora/internal/usecase/deployment"
)

func CreateDeploymentRun(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var def deploymentusecase.Definition
		if err := json.NewDecoder(r.Body).Decode(&def); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{
				Code:    "invalid_request",
				Message: "request body must be a deployment definition",
			})
			return
		}
		start := time.Now()
		result, err := service.CreateAndRun(r.Context(), deploymentusecase.CreateRunInput{
			Definition:    def,
			ProjectID:     projectIDFromRequest(r),
			CorrelationID: apimiddleware.CorrelationID(r.Context()),
		})
		if err != nil {
			telemetry.DefaultMetrics().IncFailure()
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{
				Code:    "deployment_run_failed",
				Message: err.Error(),
			})
			return
		}
		telemetry.DefaultMetrics().IncDeploymentRun()
		telemetry.DefaultMetrics().ObserveDeploymentDuration(time.Since(start))
		if result.Record.Run.Status == domaindeployment.DeploymentRunFailed {
			telemetry.DefaultMetrics().IncFailure()
		}
		RespondJSON(w, http.StatusCreated, result.Record)
	}
}

type deploymentApplyRequest struct {
	Definition deploymentusecase.Definition `json:"definition"`
	Confirm    bool                         `json:"confirm"`
}

func ApplyDeploymentRun(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req deploymentApplyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{
				Code:    "invalid_request",
				Message: "request body must include definition and confirm",
			})
			return
		}
		if !req.Confirm {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{
				Code:    "deployment_apply_unconfirmed",
				Message: "deployment apply requires confirm=true",
			})
			return
		}
		req.Definition.Spec.Options.Apply = true
		req.Definition.Spec.Options.DryRun = false
		result, err := service.CreateAndRun(r.Context(), deploymentusecase.CreateRunInput{
			Definition:    req.Definition,
			ProjectID:     projectIDFromRequest(r),
			AllowApply:    true,
			Confirm:       true,
			CorrelationID: apimiddleware.CorrelationID(r.Context()),
		})
		if err != nil {
			respondDeploymentResult(w, r, nil, err)
			return
		}
		RespondJSON(w, http.StatusCreated, result.Record)
	}
}

func PlanDeploymentRun(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var def deploymentusecase.Definition
		if err := json.NewDecoder(r.Body).Decode(&def); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{
				Code:    "invalid_request",
				Message: "request body must be a deployment definition",
			})
			return
		}
		result, err := service.Plan(r.Context(), deploymentusecase.CreateRunInput{
			Definition:    def,
			ProjectID:     projectIDFromRequest(r),
			CorrelationID: apimiddleware.CorrelationID(r.Context()),
		})
		if err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{
				Code:    "deployment_plan_failed",
				Message: err.Error(),
			})
			return
		}
		RespondJSON(w, http.StatusOK, result.Record.Plan)
	}
}

func PlanGitOpsDeployment(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var def deploymentusecase.Definition
		if err := json.NewDecoder(r.Body).Decode(&def); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_request", Message: "request body must be a deployment definition"})
			return
		}
		result, err := service.Plan(r.Context(), deploymentusecase.CreateRunInput{
			Definition:    def,
			ProjectID:     projectIDFromRequest(r),
			CorrelationID: apimiddleware.CorrelationID(r.Context()),
		})
		if err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "gitops_plan_failed", Message: err.Error()})
			return
		}
		RespondJSON(w, http.StatusOK, result.Record.GitOpsPlan)
	}
}

type gitOpsExecutionRequest struct {
	Definition deploymentusecase.Definition `json:"definition"`
	Confirm    bool                         `json:"confirm"`
	AllowPush  bool                         `json:"allowPush"`
}

func CommitGitOpsDeployment(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req gitOpsExecutionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_request", Message: "request body must include definition and confirm"})
			return
		}
		if !req.Confirm {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "gitops_commit_unconfirmed", Message: "gitops commit requires confirm=true"})
			return
		}
		req.Definition.Spec.GitOps.WriteToWorkingTree = true
		req.Definition.Spec.GitOps.Commit = true
		req.Definition.Spec.GitOps.AllowPush = req.Definition.Spec.GitOps.AllowPush || req.AllowPush
		result, err := service.CreateAndRun(r.Context(), deploymentusecase.CreateRunInput{
			Definition:    req.Definition,
			ProjectID:     projectIDFromRequest(r),
			Confirm:       true,
			CorrelationID: apimiddleware.CorrelationID(r.Context()),
		})
		if err != nil {
			respondDeploymentResult(w, r, nil, err)
			return
		}
		RespondJSON(w, http.StatusCreated, result.Record)
	}
}

func RollbackGitOpsDeployment(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req gitOpsExecutionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_request", Message: "request body must include definition, rollbackRevision, and confirm"})
			return
		}
		if !req.Confirm {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "gitops_rollback_unconfirmed", Message: "gitops rollback requires confirm=true"})
			return
		}
		req.Definition.Spec.GitOps.Rollback = true
		result, err := service.CreateAndRun(r.Context(), deploymentusecase.CreateRunInput{
			Definition:    req.Definition,
			ProjectID:     projectIDFromRequest(r),
			Confirm:       true,
			CorrelationID: apimiddleware.CorrelationID(r.Context()),
		})
		if err != nil {
			respondDeploymentResult(w, r, nil, err)
			return
		}
		RespondJSON(w, http.StatusCreated, result.Record)
	}
}

func PlanHostDeployment(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var def deploymentusecase.Definition
		if err := json.NewDecoder(r.Body).Decode(&def); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_request", Message: "request body must be a deployment definition"})
			return
		}
		result, err := service.Plan(r.Context(), deploymentusecase.CreateRunInput{
			Definition:    def,
			ProjectID:     projectIDFromRequest(r),
			CorrelationID: apimiddleware.CorrelationID(r.Context()),
		})
		if err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "host_deployment_plan_failed", Message: err.Error()})
			return
		}
		RespondJSON(w, http.StatusOK, result.Record.HostPlan)
	}
}

func CreateHostGroup(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var group deploymentusecase.HostGroup
		if err := json.NewDecoder(r.Body).Decode(&group); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_request", Message: "request body must be a host group"})
			return
		}
		created, err := service.CreateHostGroup(r.Context(), group)
		respondDeploymentResult(w, r, created, err)
	}
}

func ListHostGroups(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		groups, err := service.ListHostGroups(r.Context())
		respondDeploymentResult(w, r, groups, err)
	}
}

func GetHostGroup(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		group, err := service.GetHostGroup(r.Context(), chi.URLParam(r, "id"))
		respondDeploymentResult(w, r, group, err)
	}
}

func ListDeploymentRuns(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		scopeType, scopeID := TenantScopeFilter(r)
		records, err := service.ListFiltered(r.Context(), scopeType, scopeID)
		if respondPaginated(w, r, records, err) {
			return
		}
		respondDeploymentResult(w, r, nil, err)
	}
}

func projectIDFromRequest(r *http.Request) string {
	subject := apimiddleware.Subject(r.Context())
	if subject.ScopeType == tenant.ScopeProject {
		return subject.ScopeID
	}
	return ""
}

func GetDeploymentRun(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		record, ok := getAuthorizedDeploymentRecord(w, r, service)
		if !ok {
			return
		}
		RespondJSON(w, http.StatusOK, record)
	}
}

func GetDeploymentPlan(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		record, ok := getAuthorizedDeploymentRecord(w, r, service)
		if !ok {
			return
		}
		RespondJSON(w, http.StatusOK, record.Plan)
	}
}

func GetDeploymentHosts(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := getAuthorizedDeploymentRecord(w, r, service); !ok {
			return
		}
		hosts, err := service.Hosts(r.Context(), chi.URLParam(r, "id"))
		respondDeploymentResult(w, r, hosts, err)
	}
}

func GetDeploymentGitOpsPlan(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		record, ok := getAuthorizedDeploymentRecord(w, r, service)
		if !ok {
			return
		}
		RespondJSON(w, http.StatusOK, record.GitOpsPlan)
	}
}

func GetDeploymentArgoCDStatus(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		record, ok := getAuthorizedDeploymentRecord(w, r, service)
		if !ok {
			return
		}
		RespondJSON(w, http.StatusOK, record.ArgoCD)
	}
}

func SyncDeploymentArgoCD(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := getAuthorizedDeploymentRecord(w, r, service); !ok {
			return
		}
		var req portargocd.SyncRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_request", Message: "request body must include allowSync and confirmed"})
			return
		}
		if !req.AllowSync || !req.Confirmed {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "argocd_sync_disabled", Message: "sync requires allowSync=true and confirmed=true"})
			return
		}
		record, err := service.SyncDeployment(r.Context(), chi.URLParam(r, "id"), "", req.AllowSync, req.Confirmed)
		respondDeploymentResult(w, r, record, err)
	}
}

func GetArgoCDApplicationStatus(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status, err := service.ArgoCDStatus(r.Context(), chi.URLParam(r, "name"))
		respondDeploymentResult(w, r, status, err)
	}
}

func GetArgoCDApplicationResources(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resources, err := service.ArgoCDResources(r.Context(), chi.URLParam(r, "name"))
		respondDeploymentResult(w, r, resources, err)
	}
}

func SyncArgoCDApplication(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req portargocd.SyncRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_request", Message: "request body must include applicationName, allowSync, and confirmed"})
			return
		}
		if req.ApplicationName == "" {
			req.ApplicationName = chi.URLParam(r, "name")
		}
		if !req.AllowSync || !req.Confirmed {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "argocd_sync_disabled", Message: "sync requires allowSync=true and confirmed=true"})
			return
		}
		result, err := service.SyncArgoCDApplication(r.Context(), req)
		respondDeploymentResult(w, r, result, err)
	}
}

func GetDeploymentDiff(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		record, ok := getAuthorizedDeploymentRecord(w, r, service)
		if !ok {
			return
		}
		RespondJSON(w, http.StatusOK, record.GitOpsDiff)
	}
}

func GetDeploymentLogs(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := getAuthorizedDeploymentRecord(w, r, service); !ok {
			return
		}
		logs, err := service.Logs(r.Context(), chi.URLParam(r, "id"))
		if respondPaginated(w, r, logs, err) {
			return
		}
		respondDeploymentResult(w, r, nil, err)
	}
}

func GetDeploymentResources(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := getAuthorizedDeploymentRecord(w, r, service); !ok {
			return
		}
		resources, err := service.Resources(r.Context(), chi.URLParam(r, "id"))
		respondDeploymentResult(w, r, resources, err)
	}
}

func GetDeploymentHealth(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := getAuthorizedDeploymentRecord(w, r, service); !ok {
			return
		}
		health, err := service.Health(r.Context(), chi.URLParam(r, "id"))
		respondDeploymentResult(w, r, health, err)
	}
}

func GetDeploymentRuntimeDiff(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := getAuthorizedDeploymentRecord(w, r, service); !ok {
			return
		}
		diff, err := service.Diff(r.Context(), chi.URLParam(r, "id"))
		respondDeploymentResult(w, r, diff, err)
	}
}

func GetDeploymentManifestSnapshot(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := getAuthorizedDeploymentRecord(w, r, service); !ok {
			return
		}
		snapshot, err := service.Snapshot(r.Context(), chi.URLParam(r, "id"))
		respondDeploymentResult(w, r, snapshot, err)
	}
}

func GetDeploymentRollbackPlan(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := getAuthorizedDeploymentRecord(w, r, service); !ok {
			return
		}
		plan, err := service.RollbackPlan(r.Context(), chi.URLParam(r, "id"))
		respondDeploymentResult(w, r, plan, err)
	}
}

type deploymentRollbackRequest struct {
	Confirm bool `json:"confirm"`
}

func RollbackDeploymentRun(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := getAuthorizedDeploymentRecord(w, r, service); !ok {
			return
		}
		var req deploymentRollbackRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_request", Message: "request body must include confirm"})
			return
		}
		if !req.Confirm {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "deployment_rollback_unconfirmed", Message: "deployment rollback requires confirm=true"})
			return
		}
		record, err := service.Rollback(r.Context(), deploymentusecase.RollbackInput{
			DeploymentRunID: chi.URLParam(r, "id"),
			Confirm:         req.Confirm,
		})
		respondDeploymentResult(w, r, record, err)
	}
}

func GetDeploymentEvents(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := getAuthorizedDeploymentRecord(w, r, service); !ok {
			return
		}
		events, err := service.Events(r.Context(), chi.URLParam(r, "id"))
		if respondPaginated(w, r, events, err) {
			return
		}
		respondDeploymentResult(w, r, nil, err)
	}
}

func GetDeploymentTimeline(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := getAuthorizedDeploymentRecord(w, r, service); !ok {
			return
		}
		timeline, err := service.Timeline(r.Context(), chi.URLParam(r, "id"))
		if respondPaginated(w, r, timeline, err) {
			return
		}
		respondDeploymentResult(w, r, nil, err)
	}
}

func GetDeploymentSecurity(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		record, ok := getAuthorizedDeploymentRecord(w, r, service)
		if !ok {
			return
		}
		RespondJSON(w, http.StatusOK, record.Security)
	}
}

func CancelDeploymentRun(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := getAuthorizedDeploymentRecord(w, r, service); !ok {
			return
		}
		record, err := service.Cancel(r.Context(), chi.URLParam(r, "id"), "")
		respondDeploymentResult(w, r, record, err)
	}
}

func ResumeDeploymentRunAfterApproval(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := getAuthorizedDeploymentRecord(w, r, service); !ok {
			return
		}
		var approval domainapproval.ApprovalRequest
		if err := json.NewDecoder(r.Body).Decode(&approval); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_request", Message: "request body must be an approval request"})
			return
		}
		record, err := service.ApplyApprovalDecision(r.Context(), chi.URLParam(r, "id"), approval, "")
		respondDeploymentResult(w, r, record, err)
	}
}

func getAuthorizedDeploymentRecord(w http.ResponseWriter, r *http.Request, service *deploymentusecase.Service) (deploymentusecase.RunRecord, bool) {
	record, err := service.Get(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		respondDeploymentResult(w, r, nil, err)
		return deploymentusecase.RunRecord{}, false
	}
	if !deploymentRecordInRequestScope(r, record) {
		RespondError(w, r, http.StatusForbidden, dto.ErrorResponse{
			Code:    "forbidden",
			Message: "deployment run is outside requester scope",
			Path:    r.URL.Path,
		})
		return deploymentusecase.RunRecord{}, false
	}
	return record, true
}

func deploymentRecordInRequestScope(r *http.Request, record deploymentusecase.RunRecord) bool {
	scopeType, scopeID := TenantScopeFilter(r)
	if scopeType == "" {
		return true
	}
	if scopeID == "" {
		return false
	}
	switch scopeType {
	case tenant.ScopeProject:
		return record.Environment.ProjectID == scopeID || record.Target.ProjectID == scopeID
	case tenant.ScopeEnvironment:
		return record.Environment.ID == scopeID || record.Target.EnvironmentID == scopeID || record.Run.EnvironmentID == scopeID
	default:
		return false
	}
}

func respondDeploymentResult(w http.ResponseWriter, r *http.Request, payload any, err error) {
	if err == nil {
		RespondJSON(w, http.StatusOK, payload)
		return
	}
	status := http.StatusInternalServerError
	code := "internal_error"
	if errors.Is(err, deploymentusecase.ErrRunNotFound) {
		status = http.StatusNotFound
		code = "deployment_run_not_found"
	}
	if errors.Is(err, deploymentusecase.ErrRunTerminal) {
		status = http.StatusConflict
		code = "deployment_run_terminal"
	}
	if errors.Is(err, deploymentusecase.ErrHostGroupNotFound) {
		status = http.StatusNotFound
		code = "host_group_not_found"
	}
	RespondError(w, r, status, dto.ErrorResponse{
		Code:    code,
		Message: err.Error(),
	})
}
