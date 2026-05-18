package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sevoniva/nivora/internal/api/http/dto"
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
