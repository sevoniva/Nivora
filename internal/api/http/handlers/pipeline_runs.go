package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sevoniva/nivora/internal/api/http/dto"
	domainrunner "github.com/sevoniva/nivora/internal/domain/runner"
	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
)

func CreatePipelineRun(service *pipelineusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var def pipelineusecase.Definition
		if err := json.NewDecoder(r.Body).Decode(&def); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{
				Code:    "invalid_request",
				Message: "request body must be a pipeline definition",
			})
			return
		}
		result, err := service.CreateAndRun(r.Context(), pipelineusecase.CreateRunInput{Definition: def})
		if err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{
				Code:    "pipeline_run_failed",
				Message: err.Error(),
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

func ListPipelineRuns(service *pipelineusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		records, err := service.List(r.Context())
		if err != nil {
			respondPipelineResult(w, r, nil, err)
			return
		}
		response := make([]map[string]any, 0, len(records))
		for _, record := range records {
			response = append(response, pipelineRunResponse(record))
		}
		RespondJSON(w, http.StatusOK, response)
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

func GetPipelineRunTimeline(service *pipelineusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		timeline, err := service.Timeline(r.Context(), chi.URLParam(r, "id"))
		respondPipelineResult(w, r, timeline, err)
	}
}

func CancelPipelineRun(service *pipelineusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		record, err := service.Cancel(r.Context(), chi.URLParam(r, "id"), "")
		if err != nil {
			respondPipelineResult(w, r, nil, err)
			return
		}
		RespondJSON(w, http.StatusOK, pipelineRunResponse(record))
	}
}

func RegisterRunner(service *pipelineusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var runner domainrunner.Runner
		if err := json.NewDecoder(r.Body).Decode(&runner); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{
				Code:    "invalid_request",
				Message: "request body must be a runner",
			})
			return
		}
		if runner.ID == "" {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{
				Code:    "invalid_request",
				Message: "runner id is required",
			})
			return
		}
		if err := service.RegisterRunner(r.Context(), runner); err != nil {
			respondPipelineResult(w, r, nil, err)
			return
		}
		saved, err := service.GetRunner(r.Context(), runner.ID)
		if err != nil {
			respondPipelineResult(w, r, nil, err)
			return
		}
		RespondJSON(w, http.StatusCreated, saved)
	}
}

func ListRunners(service *pipelineusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		runners, err := service.ListRunners(r.Context())
		respondPipelineResult(w, r, runners, err)
	}
}

func GetRunner(service *pipelineusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		runner, err := service.GetRunner(r.Context(), chi.URLParam(r, "id"))
		respondPipelineResult(w, r, runner, err)
	}
}

func HeartbeatRunner(service *pipelineusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		runner, err := service.HeartbeatRunner(r.Context(), chi.URLParam(r, "id"))
		respondPipelineResult(w, r, runner, err)
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
	if errors.Is(err, pipelineusecase.ErrRunnerNotFound) {
		status = http.StatusNotFound
		code = "runner_not_found"
	}
	if errors.Is(err, pipelineusecase.ErrRunTerminal) {
		status = http.StatusConflict
		code = "pipeline_run_terminal"
	}
	RespondError(w, r, status, dto.ErrorResponse{
		Code:    code,
		Message: err.Error(),
	})
}
