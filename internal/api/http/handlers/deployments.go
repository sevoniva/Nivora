package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sevoniva/nivora/internal/api/http/dto"
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
		result, err := service.CreateAndRun(r.Context(), deploymentusecase.CreateRunInput{Definition: def})
		if err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{
				Code:    "deployment_run_failed",
				Message: err.Error(),
			})
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
		result, err := service.Plan(r.Context(), deploymentusecase.CreateRunInput{Definition: def})
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
		result, err := service.Plan(r.Context(), deploymentusecase.CreateRunInput{Definition: def})
		if err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "gitops_plan_failed", Message: err.Error()})
			return
		}
		RespondJSON(w, http.StatusOK, result.Record.GitOpsPlan)
	}
}

func ListDeploymentRuns(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		records, err := service.List(r.Context())
		respondDeploymentResult(w, r, records, err)
	}
}

func GetDeploymentRun(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		record, err := service.Get(r.Context(), chi.URLParam(r, "id"))
		respondDeploymentResult(w, r, record, err)
	}
}

func GetDeploymentPlan(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		record, err := service.Get(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			respondDeploymentResult(w, r, nil, err)
			return
		}
		RespondJSON(w, http.StatusOK, record.Plan)
	}
}

func GetDeploymentGitOpsPlan(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		record, err := service.Get(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			respondDeploymentResult(w, r, nil, err)
			return
		}
		RespondJSON(w, http.StatusOK, record.GitOpsPlan)
	}
}

func GetDeploymentArgoCDStatus(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		record, err := service.Get(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			respondDeploymentResult(w, r, nil, err)
			return
		}
		RespondJSON(w, http.StatusOK, record.ArgoCD)
	}
}

func SyncDeploymentArgoCD(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		record, err := service.Get(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			respondDeploymentResult(w, r, nil, err)
			return
		}
		RespondJSON(w, http.StatusOK, record.GitOpsDiff)
	}
}

func GetDeploymentLogs(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logs, err := service.Logs(r.Context(), chi.URLParam(r, "id"))
		respondDeploymentResult(w, r, logs, err)
	}
}

func GetDeploymentResources(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resources, err := service.Resources(r.Context(), chi.URLParam(r, "id"))
		respondDeploymentResult(w, r, resources, err)
	}
}

func GetDeploymentHealth(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		health, err := service.Health(r.Context(), chi.URLParam(r, "id"))
		respondDeploymentResult(w, r, health, err)
	}
}

func GetDeploymentRuntimeDiff(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		diff, err := service.Diff(r.Context(), chi.URLParam(r, "id"))
		respondDeploymentResult(w, r, diff, err)
	}
}

func GetDeploymentManifestSnapshot(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		snapshot, err := service.Snapshot(r.Context(), chi.URLParam(r, "id"))
		respondDeploymentResult(w, r, snapshot, err)
	}
}

func GetDeploymentRollbackPlan(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		plan, err := service.RollbackPlan(r.Context(), chi.URLParam(r, "id"))
		respondDeploymentResult(w, r, plan, err)
	}
}

func GetDeploymentEvents(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		events, err := service.Events(r.Context(), chi.URLParam(r, "id"))
		respondDeploymentResult(w, r, events, err)
	}
}

func GetDeploymentTimeline(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		timeline, err := service.Timeline(r.Context(), chi.URLParam(r, "id"))
		respondDeploymentResult(w, r, timeline, err)
	}
}

func CancelDeploymentRun(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		record, err := service.Cancel(r.Context(), chi.URLParam(r, "id"), "")
		respondDeploymentResult(w, r, record, err)
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
	RespondError(w, r, status, dto.ErrorResponse{
		Code:    code,
		Message: err.Error(),
	})
}
