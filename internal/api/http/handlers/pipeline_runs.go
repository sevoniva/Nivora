package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sevoniva/nivora/internal/api/http/dto"
	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
)

func CreatePipelineRun(service *pipelineusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var def pipelineusecase.Definition
		if err := json.NewDecoder(r.Body).Decode(&def); err != nil {
			RespondJSON(w, http.StatusBadRequest, dto.ErrorResponse{
				Code:    "invalid_request",
				Message: "request body must be a pipeline definition",
				Path:    r.URL.Path,
			})
			return
		}
		result, err := service.CreateAndRun(r.Context(), pipelineusecase.CreateRunInput{Definition: def})
		if err != nil {
			RespondJSON(w, http.StatusBadRequest, dto.ErrorResponse{
				Code:    "pipeline_run_failed",
				Message: err.Error(),
				Path:    r.URL.Path,
			})
			return
		}
		RespondJSON(w, http.StatusCreated, pipelineRunResponse(result.Record))
	}
}

func GetPipelineRun(service *pipelineusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		record, err := service.Get(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			respondPipelineResult(w, r, nil, err)
			return
		}
		RespondJSON(w, http.StatusOK, pipelineRunResponse(record))
	}
}

func GetPipelineRunLogs(service *pipelineusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logs, err := service.Logs(r.Context(), chi.URLParam(r, "id"))
		respondPipelineResult(w, r, logs, err)
	}
}

func pipelineRunResponse(record pipelineusecase.RunRecord) map[string]any {
	return map[string]any{
		"pipeline": map[string]any{
			"id":   record.Pipeline.ID,
			"name": record.Pipeline.Name,
		},
		"run": map[string]any{
			"id":            record.Run.ID,
			"pipelineId":    record.Run.PipelineID,
			"status":        record.Run.Status,
			"startedAt":     record.Run.StartedAt,
			"finishedAt":    record.Run.FinishedAt,
			"failureReason": record.Run.FailureReason,
			"createdAt":     record.Run.CreatedAt,
			"updatedAt":     record.Run.UpdatedAt,
		},
		"stages": record.Stages,
		"logs":   record.Logs,
		"events": record.Events,
	}
}

func GetPipelineRunEvents(service *pipelineusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		events, err := service.Events(r.Context(), chi.URLParam(r, "id"))
		respondPipelineResult(w, r, events, err)
	}
}

func respondPipelineResult(w http.ResponseWriter, r *http.Request, payload any, err error) {
	if err == nil {
		RespondJSON(w, http.StatusOK, payload)
		return
	}
	status := http.StatusInternalServerError
	code := "internal_error"
	if errors.Is(err, pipelineusecase.ErrRunNotFound) {
		status = http.StatusNotFound
		code = "pipeline_run_not_found"
	}
	RespondJSON(w, status, dto.ErrorResponse{
		Code:    code,
		Message: err.Error(),
		Path:    r.URL.Path,
	})
}
