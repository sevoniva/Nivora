package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/sevoniva/nivora/internal/api/http/dto"
	apimiddleware "github.com/sevoniva/nivora/internal/api/http/middleware"
	domainpipeline "github.com/sevoniva/nivora/internal/domain/pipeline"
	domainrunner "github.com/sevoniva/nivora/internal/domain/runner"
	"github.com/sevoniva/nivora/internal/domain/tenant"
	"github.com/sevoniva/nivora/internal/infra/telemetry"
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
		start := time.Now()
		projectID := ""
		subject := apimiddleware.Subject(r.Context())
		if subject.ScopeType == tenant.ScopeProject {
			projectID = subject.ScopeID
		}
		result, err := service.CreateAndRun(r.Context(), pipelineusecase.CreateRunInput{
			Definition:    def,
			ProjectID:     projectID,
			CorrelationID: apimiddleware.CorrelationID(r.Context()),
		})
		if err != nil {
			telemetry.DefaultMetrics().IncFailure()
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{
				Code:    "pipeline_run_failed",
				Message: err.Error(),
			})
			return
		}
		telemetry.DefaultMetrics().IncPipelineRun()
		telemetry.DefaultMetrics().ObservePipelineDuration(time.Since(start))
		if result.Record.Run.Status == domainpipeline.PipelineRunFailed || result.Record.Run.Status == domainpipeline.PipelineRunTimeout {
			telemetry.DefaultMetrics().IncFailure()
		}
		RespondJSON(w, http.StatusCreated, pipelineRunResponse(result.Record))
	}
}

func GetPipelineRun(service *pipelineusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		record, ok := getAuthorizedPipelineRecord(w, r, service)
		if !ok {
			return
		}
		RespondJSON(w, http.StatusOK, pipelineRunResponse(record))
	}
}

func ListPipelineRuns(service *pipelineusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		scopeType, scopeID := TenantScopeFilter(r)
		records, err := service.ListFiltered(r.Context(), scopeType, scopeID)
		if err != nil {
			respondPipelineResult(w, r, nil, err)
			return
		}
		response := make([]map[string]any, 0, len(records))
		for _, record := range records {
			response = append(response, pipelineRunResponse(record))
		}
		if respondPaginated(w, r, response, nil) {
			return
		}
	}
}

func GetPipelineRunLogs(service *pipelineusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := getAuthorizedPipelineRecord(w, r, service); !ok {
			return
		}
		logs, err := service.Logs(r.Context(), chi.URLParam(r, "id"))
		if respondPaginated(w, r, logs, err) {
			return
		}
		respondPipelineResult(w, r, nil, err)
	}
}

func pipelineRunResponse(record pipelineusecase.RunRecord) map[string]any {
	return map[string]any{
		"pipeline": map[string]any{
			"id":        record.Pipeline.ID,
			"projectId": record.Pipeline.ProjectID,
			"name":      record.Pipeline.Name,
		},
		"run": map[string]any{
			"id":                record.Run.ID,
			"pipelineId":        record.Run.PipelineID,
			"pipelineVersionId": record.Run.PipelineVersionID,
			"correlationId":     record.Run.CorrelationID,
			"status":            record.Run.Status,
			"startedAt":         record.Run.StartedAt,
			"finishedAt":        record.Run.FinishedAt,
			"failureReason":     record.Run.FailureReason,
			"createdAt":         record.Run.CreatedAt,
			"updatedAt":         record.Run.UpdatedAt,
		},
		"stages": record.Stages,
		"logs":   record.Logs,
		"events": record.Events,
	}
}

func GetPipelineRunEvents(service *pipelineusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := getAuthorizedPipelineRecord(w, r, service); !ok {
			return
		}
		events, err := service.Events(r.Context(), chi.URLParam(r, "id"))
		if respondPaginated(w, r, events, err) {
			return
		}
		respondPipelineResult(w, r, nil, err)
	}
}

func GetPipelineRunTimeline(service *pipelineusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := getAuthorizedPipelineRecord(w, r, service); !ok {
			return
		}
		timeline, err := service.Timeline(r.Context(), chi.URLParam(r, "id"))
		if respondPaginated(w, r, timeline, err) {
			return
		}
		respondPipelineResult(w, r, nil, err)
	}
}

func CancelPipelineRun(service *pipelineusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := getAuthorizedPipelineRecord(w, r, service); !ok {
			return
		}
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
		result, err := service.RegisterRunnerWithToken(r.Context(), runner)
		if err != nil {
			respondPipelineResult(w, r, nil, err)
			return
		}
		RespondJSON(w, http.StatusCreated, result)
	}
}

func RotateRunnerToken(service *pipelineusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := service.RotateRunnerToken(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			respondPipelineResult(w, r, nil, err)
			return
		}
		RespondJSON(w, http.StatusOK, result)
	}
}

func RevokeRunnerToken(service *pipelineusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		runner, err := service.RevokeRunnerToken(r.Context(), chi.URLParam(r, "id"))
		respondPipelineResult(w, r, runner, err)
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
		id := chi.URLParam(r, "id")
		if err := service.ValidateRunnerToken(r.Context(), id, runnerToken(r)); err != nil {
			respondPipelineResult(w, r, nil, err)
			return
		}
		runner, err := service.HeartbeatRunner(r.Context(), id)
		if err == nil {
			telemetry.DefaultMetrics().IncRunnerHeartbeat()
		}
		respondPipelineResult(w, r, runner, err)
	}
}

func ClaimRunnerJob(service *pipelineusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		id := chi.URLParam(r, "id")
		if err := service.ValidateRunnerToken(r.Context(), id, runnerToken(r)); err != nil {
			respondPipelineResult(w, r, nil, err)
			return
		}
		lease := 30 * time.Second
		if value := r.URL.Query().Get("leaseSeconds"); value != "" {
			parsed, err := time.ParseDuration(value + "s")
			if err != nil {
				RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_request", Message: "leaseSeconds must be an integer number of seconds"})
				return
			}
			lease = parsed
		}
		claim, err := service.ClaimJob(r.Context(), id, lease)
		telemetry.DefaultMetrics().IncJobClaim()
		telemetry.DefaultMetrics().ObserveJobClaimLatency(time.Since(start))
		respondPipelineResult(w, r, claim, err)
	}
}

func AppendJobLogs(service *pipelineusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		runnerID := ""
		jobID := chi.URLParam(r, "job_id")
		if jobID != "" {
			runnerID = chi.URLParam(r, "id")
		} else {
			jobID = chi.URLParam(r, "id")
		}
		if runnerID != "" {
			if err := service.ValidateRunnerToken(r.Context(), runnerID, runnerToken(r)); err != nil {
				respondPipelineResult(w, r, nil, err)
				return
			}
			if err := service.ValidateRunnerJob(r.Context(), runnerID, jobID); err != nil {
				respondPipelineResult(w, r, nil, err)
				return
			}
		}
		var input pipelineusecase.AppendJobLogInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_request", Message: "request body must be a job log append request"})
			return
		}
		if len(input.Content) > MaxLogChunkBytes {
			RespondError(w, r, http.StatusRequestEntityTooLarge, dto.ErrorResponse{Code: "log_chunk_too_large", Message: "log chunk content exceeds 64 KiB"})
			return
		}
		logs, err := service.AppendJobLog(r.Context(), jobID, input)
		respondPipelineResult(w, r, logs, err)
	}
}

func UpdateJobStatus(service *pipelineusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		runnerID := ""
		jobID := chi.URLParam(r, "job_id")
		if jobID != "" {
			runnerID = chi.URLParam(r, "id")
		} else {
			jobID = chi.URLParam(r, "id")
		}
		if runnerID != "" {
			if err := service.ValidateRunnerToken(r.Context(), runnerID, runnerToken(r)); err != nil {
				respondPipelineResult(w, r, nil, err)
				return
			}
			if err := service.ValidateRunnerJob(r.Context(), runnerID, jobID); err != nil {
				respondPipelineResult(w, r, nil, err)
				return
			}
		}
		var input pipelineusecase.UpdateJobStatusInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_request", Message: "request body must be a job status update"})
			return
		}
		record, err := service.UpdateJobStatus(r.Context(), jobID, input)
		if err != nil {
			respondPipelineResult(w, r, nil, err)
			return
		}
		RespondJSON(w, http.StatusOK, pipelineRunResponse(record))
	}
}

func MarkOfflineRunners(service *pipelineusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		timeout := time.Minute
		if value := r.URL.Query().Get("timeoutSeconds"); value != "" {
			parsed, err := time.ParseDuration(value + "s")
			if err != nil {
				RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_request", Message: "timeoutSeconds must be an integer number of seconds"})
				return
			}
			timeout = parsed
		}
		runners, err := service.MarkOfflineRunners(r.Context(), timeout)
		respondPipelineResult(w, r, runners, err)
	}
}

func RequestPipelineRunCancel(service *pipelineusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := getAuthorizedPipelineRecord(w, r, service); !ok {
			return
		}
		record, err := service.RequestCancel(r.Context(), chi.URLParam(r, "id"), "")
		if err != nil {
			respondPipelineResult(w, r, nil, err)
			return
		}
		RespondJSON(w, http.StatusOK, pipelineRunResponse(record))
	}
}

func getAuthorizedPipelineRecord(w http.ResponseWriter, r *http.Request, service *pipelineusecase.Service) (pipelineusecase.RunRecord, bool) {
	record, err := service.Get(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		respondPipelineResult(w, r, nil, err)
		return pipelineusecase.RunRecord{}, false
	}
	if !pipelineRecordInRequestScope(r, record) {
		RespondError(w, r, http.StatusForbidden, dto.ErrorResponse{
			Code:    "forbidden",
			Message: "pipeline run is outside requester scope",
			Path:    r.URL.Path,
		})
		return pipelineusecase.RunRecord{}, false
	}
	return record, true
}

func pipelineRecordInRequestScope(r *http.Request, record pipelineusecase.RunRecord) bool {
	scopeType, scopeID := TenantScopeFilter(r)
	if scopeType == "" {
		return true
	}
	if scopeID == "" {
		return false
	}
	return scopeType == tenant.ScopeProject && record.Pipeline.ProjectID == scopeID
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
	if errors.Is(err, pipelineusecase.ErrJobNotFound) {
		status = http.StatusNotFound
		code = "job_run_not_found"
	}
	if errors.Is(err, pipelineusecase.ErrNoClaimableJob) {
		status = http.StatusConflict
		code = "no_claimable_job"
	}
	if errors.Is(err, pipelineusecase.ErrRunnerUnauthorized) || errors.Is(err, pipelineusecase.ErrRunnerTokenRevoked) {
		status = http.StatusUnauthorized
		code = "runner_unauthorized"
	}
	if errors.Is(err, pipelineusecase.ErrRunnerConcurrencyLimit) {
		status = http.StatusConflict
		code = "runner_concurrency_limit"
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

func runnerToken(r *http.Request) string {
	if token := r.Header.Get("X-Nivora-Runner-Token"); token != "" {
		return token
	}
	header := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(header) > len(prefix) && strings.EqualFold(header[:len(prefix)], prefix) {
		return header[len(prefix):]
	}
	return ""
}
