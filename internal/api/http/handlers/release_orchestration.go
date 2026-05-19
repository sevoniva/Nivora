package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sevoniva/nivora/internal/api/http/dto"
	apimiddleware "github.com/sevoniva/nivora/internal/api/http/middleware"
	domainapproval "github.com/sevoniva/nivora/internal/domain/approval"
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
			CorrelationID: apimiddleware.CorrelationID(r.Context()),
		})
		if err == nil {
			RespondJSON(w, http.StatusCreated, record)
			return
		}
		respondReleaseOrchestrationResult(w, r, nil, err)
	}
}

func GetReleasePlan(service *releaseorchestration.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		record, err := service.GetPlan(r.Context(), chi.URLParam(r, "id"))
		respondReleaseOrchestrationResult(w, r, record, err)
	}
}

func ListReleaseExecutions(service *releaseorchestration.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		records, err := service.ListExecutions(r.Context(), chi.URLParam(r, "id"))
		respondReleaseOrchestrationResult(w, r, records, err)
	}
}

func GetReleaseExecution(service *releaseorchestration.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		record, err := service.GetExecution(r.Context(), chi.URLParam(r, "execution_id"))
		respondReleaseOrchestrationResult(w, r, record, err)
	}
}

func GetReleaseExecutionTimeline(service *releaseorchestration.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		timeline, err := service.Timeline(r.Context(), chi.URLParam(r, "execution_id"))
		respondReleaseOrchestrationResult(w, r, timeline, err)
	}
}

func GetReleaseExecutionTargets(service *releaseorchestration.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		targets, err := service.Targets(r.Context(), chi.URLParam(r, "execution_id"))
		respondReleaseOrchestrationResult(w, r, targets, err)
	}
}

func CancelReleaseExecution(service *releaseorchestration.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		record, err := service.Cancel(r.Context(), chi.URLParam(r, "execution_id"), "")
		respondReleaseOrchestrationResult(w, r, record, err)
	}
}

func ResumeReleaseExecutionAfterApproval(service *releaseorchestration.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		record, err := service.GetPlan(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			respondReleaseOrchestrationResult(w, r, nil, err)
			return
		}
		RespondJSON(w, http.StatusOK, record.Security)
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
