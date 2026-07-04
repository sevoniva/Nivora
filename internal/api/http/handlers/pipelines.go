package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/sevoniva/nivora/internal/api/http/dto"
	apimiddleware "github.com/sevoniva/nivora/internal/api/http/middleware"
	domainpipeline "github.com/sevoniva/nivora/internal/domain/pipeline"
	"github.com/sevoniva/nivora/internal/domain/tenant"
	"github.com/sevoniva/nivora/internal/infra/telemetry"
	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
)

func ListPipelineDefinitions(catalog *pipelineusecase.DefinitionCatalog) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID := r.URL.Query().Get("projectId")
		scopeType, scopeID := TenantScopeFilter(r)
		if scopeType != "" {
			if scopeType != tenant.ScopeProject || scopeID == "" {
				RespondJSON(w, http.StatusOK, map[string]any{"pipelines": []pipelineusecase.DefinitionRecord{}})
				return
			}
			projectID = scopeID
		}
		records, err := catalog.List(r.Context(), projectID)
		if err != nil {
			respondPipelineDefinitionError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, map[string]any{"pipelines": records})
	}
}

func CreatePipelineDefinition(catalog *pipelineusecase.DefinitionCatalog) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input pipelineusecase.DefinitionCreateInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_json", Message: "request body must be valid JSON"})
			return
		}
		if !applyPipelineDefinitionCreateScope(w, r, &input) {
			return
		}
		record, err := catalog.Create(r.Context(), input)
		if err != nil {
			respondPipelineDefinitionError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusCreated, record)
	}
}

func GetPipelineDefinition(catalog *pipelineusecase.DefinitionCatalog) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		record, ok := getAuthorizedPipelineDefinition(w, r, catalog)
		if !ok {
			return
		}
		RespondJSON(w, http.StatusOK, record)
	}
}

func UpdatePipelineDefinition(catalog *pipelineusecase.DefinitionCatalog) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := getAuthorizedPipelineDefinition(w, r, catalog); !ok {
			return
		}
		var input pipelineusecase.DefinitionUpdateInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_json", Message: "request body must be valid JSON"})
			return
		}
		record, err := catalog.Update(r.Context(), chi.URLParam(r, "id"), input)
		if err != nil {
			respondPipelineDefinitionError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, record)
	}
}

func DisablePipelineDefinition(catalog *pipelineusecase.DefinitionCatalog) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := getAuthorizedPipelineDefinition(w, r, catalog); !ok {
			return
		}
		record, err := catalog.Disable(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			respondPipelineDefinitionError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, record)
	}
}

func ListPipelineDefinitionVersions(catalog *pipelineusecase.DefinitionCatalog) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		record, ok := getAuthorizedPipelineDefinition(w, r, catalog)
		if !ok {
			return
		}
		versions, err := catalog.Versions(r.Context(), record.Pipeline.ID)
		if err != nil {
			respondPipelineDefinitionError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, map[string]any{
			"pipelineId":       record.Pipeline.ID,
			"versions":         versions,
			"historyComplete":  true,
			"currentVersionId": record.Version.ID,
		})
	}
}

func RunPipelineDefinition(catalog *pipelineusecase.DefinitionCatalog, service *pipelineusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		record, ok := getAuthorizedPipelineDefinition(w, r, catalog)
		if !ok {
			return
		}
		if !record.Pipeline.Enabled {
			RespondError(w, r, http.StatusConflict, dto.ErrorResponse{Code: "pipeline_definition_disabled", Message: "pipeline definition is disabled"})
			return
		}
		definition := record.Definition
		versionID := record.Version.ID
		if requestedVersion := r.URL.Query().Get("version"); requestedVersion != "" {
			versionNumber, err := strconv.Atoi(requestedVersion)
			if err != nil || versionNumber <= 0 {
				RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_pipeline_version", Message: "version must be a positive integer"})
				return
			}
			versionRecord, err := catalog.Version(r.Context(), record.Pipeline.ID, versionNumber)
			if err != nil {
				respondPipelineDefinitionError(w, r, err)
				return
			}
			definition = versionRecord.Definition
			versionID = versionRecord.Version.ID
		}
		start := time.Now()
		result, err := service.CreateAndRun(r.Context(), pipelineusecase.CreateRunInput{
			Definition:        definition,
			ProjectID:         record.Pipeline.ProjectID,
			PipelineID:        record.Pipeline.ID,
			PipelineVersionID: versionID,
			CorrelationID:     apimiddleware.CorrelationID(r.Context()),
		})
		if err != nil {
			telemetry.DefaultMetrics().IncFailure()
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "pipeline_run_failed", Message: err.Error()})
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

func applyPipelineDefinitionCreateScope(w http.ResponseWriter, r *http.Request, input *pipelineusecase.DefinitionCreateInput) bool {
	scopeType, scopeID := TenantScopeFilter(r)
	if scopeType == "" {
		return true
	}
	if scopeType != tenant.ScopeProject || scopeID == "" {
		RespondError(w, r, http.StatusForbidden, dto.ErrorResponse{Code: "forbidden", Message: "pipeline definitions require project scope"})
		return false
	}
	input.ProjectID = scopeID
	return true
}

func getAuthorizedPipelineDefinition(w http.ResponseWriter, r *http.Request, catalog *pipelineusecase.DefinitionCatalog) (pipelineusecase.DefinitionRecord, bool) {
	record, err := catalog.Get(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		respondPipelineDefinitionError(w, r, err)
		return pipelineusecase.DefinitionRecord{}, false
	}
	scopeType, scopeID := TenantScopeFilter(r)
	if scopeType == "" {
		return record, true
	}
	if scopeType == tenant.ScopeProject && scopeID != "" && record.Pipeline.ProjectID == scopeID {
		return record, true
	}
	RespondError(w, r, http.StatusForbidden, dto.ErrorResponse{Code: "forbidden", Message: "pipeline definition is outside requester scope"})
	return pipelineusecase.DefinitionRecord{}, false
}

func respondPipelineDefinitionError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, pipelineusecase.ErrPipelineDefinitionNotFound):
		RespondError(w, r, http.StatusNotFound, dto.ErrorResponse{Code: "pipeline_definition_not_found", Message: err.Error()})
	case errors.Is(err, pipelineusecase.ErrPipelineDefinitionAlreadyExists):
		RespondError(w, r, http.StatusConflict, dto.ErrorResponse{Code: "pipeline_definition_exists", Message: err.Error()})
	default:
		RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_pipeline_definition", Message: err.Error()})
	}
}
