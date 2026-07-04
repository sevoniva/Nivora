package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sevoniva/nivora/internal/api/http/dto"
	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
)

func ListPipelineDefinitions(catalog *pipelineusecase.DefinitionCatalog) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		records, err := catalog.List(r.Context(), r.URL.Query().Get("projectId"))
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
		record, err := catalog.Get(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			respondPipelineDefinitionError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, record)
	}
}

func UpdatePipelineDefinition(catalog *pipelineusecase.DefinitionCatalog) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		record, err := catalog.Disable(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			respondPipelineDefinitionError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, record)
	}
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
