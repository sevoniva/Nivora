package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sevoniva/nivora/internal/api/http/dto"
	apimiddleware "github.com/sevoniva/nivora/internal/api/http/middleware"
	domainapproval "github.com/sevoniva/nivora/internal/domain/approval"
	"github.com/sevoniva/nivora/internal/domain/tenant"
	releaseorchestration "github.com/sevoniva/nivora/internal/usecase/releaseorchestration"
)

func PlanRelease(service *releaseorchestration.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var def releaseorchestration.Definition
		if err := json.NewDecoder(r.Body).Decode(&def); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_request", Message: "request body must be a release orchestration definition"})
			return
		}
		if releaseID := chi.URLParam(r, "id"); releaseID != "" && releaseID != "local" && def.Spec.ReleaseID == "" {
			def.Spec.ReleaseID = releaseID
		}
		record, err := service.Plan(r.Context(), releaseorchestration.PlanInput{
			Definition:    def,
			ProjectID:     releaseProjectIDFromRequest(r),
			CorrelationID: apimiddleware.CorrelationID(r.Context()),
		})
		respondReleaseOrchestrationResult(w, r, record, err)
	}
}

func DeployRelease(service *releaseorchestration.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var def releaseorchestration.Definition
		if err := json.NewDecoder(r.Body).Decode(&def); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_request", Message: "request body must be a release orchestration definition"})
			return
		}
		if releaseID := chi.URLParam(r, "id"); releaseID != "" && releaseID != "local" && def.Spec.ReleaseID == "" {
			def.Spec.ReleaseID = releaseID
		}
		record, err := service.Deploy(r.Context(), releaseorchestration.DeployInput{
			Definition:    def,
			ProjectID:     releaseProjectIDFromRequest(r),
			CorrelationID: apimiddleware.CorrelationID(r.Context()),
		})
		if err == nil {
			RespondJSON(w, http.StatusCreated, record)
			return
		}
		respondReleaseOrchestrationResult(w, r, nil, err)
	}
}

func releaseProjectIDFromRequest(r *http.Request) string {
	subject := apimiddleware.Subject(r.Context())
	if subject.ScopeType == tenant.ScopeProject {
		return subject.ScopeID
	}
	return ""
}

func GetReleasePlan(service *releaseorchestration.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		record, ok := getAuthorizedReleasePlan(w, r, service)
		if !ok {
			return
		}
		RespondJSON(w, http.StatusOK, record)
	}
}

func ListReleaseExecutions(service *releaseorchestration.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		records, err := service.ListExecutions(r.Context(), chi.URLParam(r, "id"))
		if err == nil {
			records = filterReleaseExecutionsForRequest(r, records)
		}
		respondReleaseOrchestrationResult(w, r, records, err)
	}
}

func GetReleaseExecution(service *releaseorchestration.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		record, ok := getAuthorizedReleaseExecution(w, r, service)
		if !ok {
			return
		}
		RespondJSON(w, http.StatusOK, record)
	}
}

func GetReleaseExecutionTimeline(service *releaseorchestration.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := getAuthorizedReleaseExecution(w, r, service); !ok {
			return
		}
		timeline, err := service.Timeline(r.Context(), chi.URLParam(r, "execution_id"))
		respondReleaseOrchestrationResult(w, r, timeline, err)
	}
}

func GetReleaseExecutionTargets(service *releaseorchestration.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := getAuthorizedReleaseExecution(w, r, service); !ok {
			return
		}
		targets, err := service.Targets(r.Context(), chi.URLParam(r, "execution_id"))
		respondReleaseOrchestrationResult(w, r, targets, err)
	}
}

func CancelReleaseExecution(service *releaseorchestration.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := getAuthorizedReleaseExecution(w, r, service); !ok {
			return
		}
		record, err := service.Cancel(r.Context(), chi.URLParam(r, "execution_id"), "")
		respondReleaseOrchestrationResult(w, r, record, err)
	}
}

func ResumeReleaseExecutionAfterApproval(service *releaseorchestration.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := getAuthorizedReleaseExecution(w, r, service); !ok {
			return
		}
		var approval domainapproval.ApprovalRequest
		if err := json.NewDecoder(r.Body).Decode(&approval); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_request", Message: "request body must be an approval request"})
			return
		}
		record, err := service.ApplyApprovalDecision(r.Context(), chi.URLParam(r, "execution_id"), approval, "")
		respondReleaseOrchestrationResult(w, r, record, err)
	}
}

func GetReleaseSecurity(service *releaseorchestration.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		record, ok := getAuthorizedReleasePlan(w, r, service)
		if !ok {
			return
		}
		RespondJSON(w, http.StatusOK, record.Security)
	}
}

func getAuthorizedReleasePlan(w http.ResponseWriter, r *http.Request, service *releaseorchestration.Service) (releaseorchestration.PlanRecord, bool) {
	record, err := service.GetPlan(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		respondReleaseOrchestrationResult(w, r, nil, err)
		return releaseorchestration.PlanRecord{}, false
	}
	if !releasePlanInRequestScope(r, record.Plan) {
		RespondError(w, r, http.StatusForbidden, dto.ErrorResponse{
			Code:    "forbidden",
			Message: "release plan is outside requester scope",
			Path:    r.URL.Path,
		})
		return releaseorchestration.PlanRecord{}, false
	}
	return record, true
}

func getAuthorizedReleaseExecution(w http.ResponseWriter, r *http.Request, service *releaseorchestration.Service) (releaseorchestration.ExecutionRecord, bool) {
	return getAuthorizedReleaseExecutionByIDParam(w, r, service, "execution_id")
}

func getAuthorizedReleaseExecutionByIDParam(w http.ResponseWriter, r *http.Request, service *releaseorchestration.Service, param string) (releaseorchestration.ExecutionRecord, bool) {
	record, err := service.GetExecution(r.Context(), chi.URLParam(r, param))
	if err != nil {
		respondReleaseOrchestrationResult(w, r, nil, err)
		return releaseorchestration.ExecutionRecord{}, false
	}
	if !releaseExecutionInRequestScope(r, record) {
		RespondError(w, r, http.StatusForbidden, dto.ErrorResponse{
			Code:    "forbidden",
			Message: "release execution is outside requester scope",
			Path:    r.URL.Path,
		})
		return releaseorchestration.ExecutionRecord{}, false
	}
	return record, true
}

func filterReleaseExecutionsForRequest(r *http.Request, records []releaseorchestration.ExecutionRecord) []releaseorchestration.ExecutionRecord {
	scopeType, _ := TenantScopeFilter(r)
	if scopeType == "" {
		return records
	}
	filtered := make([]releaseorchestration.ExecutionRecord, 0, len(records))
	for _, record := range records {
		if releaseExecutionInRequestScope(r, record) {
			filtered = append(filtered, record)
		}
	}
	return filtered
}

func releaseExecutionInRequestScope(r *http.Request, record releaseorchestration.ExecutionRecord) bool {
	if releasePlanInRequestScope(r, record.Plan) {
		return true
	}
	scopeType, scopeID := TenantScopeFilter(r)
	if scopeType == "" {
		return true
	}
	return scopeType == tenant.ScopeEnvironment && scopeID != "" && record.Execution.EnvironmentID == scopeID
}

func releasePlanInRequestScope(r *http.Request, plan releaseorchestration.ReleasePlan) bool {
	scopeType, scopeID := TenantScopeFilter(r)
	if scopeType == "" {
		return true
	}
	if scopeID == "" {
		return false
	}
	switch scopeType {
	case tenant.ScopeProject:
		for _, target := range plan.Targets {
			if target.ProjectID == scopeID {
				return true
			}
		}
		return false
	case tenant.ScopeEnvironment:
		if plan.EnvironmentID == scopeID {
			return true
		}
		for _, target := range plan.Targets {
			if target.EnvironmentID == scopeID {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func respondReleaseOrchestrationResult(w http.ResponseWriter, r *http.Request, payload any, err error) {
	if err == nil {
		RespondJSON(w, http.StatusOK, payload)
		return
	}
	status := http.StatusBadRequest
	code := "release_orchestration_error"
	if errors.Is(err, releaseorchestration.ErrPlanNotFound) {
		status = http.StatusNotFound
		code = "release_plan_not_found"
	}
	if errors.Is(err, releaseorchestration.ErrExecutionNotFound) {
		status = http.StatusNotFound
		code = "release_execution_not_found"
	}
	if errors.Is(err, releaseorchestration.ErrExecutionTerminal) {
		status = http.StatusConflict
		code = "release_execution_terminal"
	}
	RespondError(w, r, status, dto.ErrorResponse{Code: code, Message: err.Error()})
}
