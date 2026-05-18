package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sevoniva/nivora/internal/api/http/dto"
	domainartifact "github.com/sevoniva/nivora/internal/domain/artifact"
	artifactusecase "github.com/sevoniva/nivora/internal/usecase/artifact"
)

type artifactRequest struct {
	Reference string `json:"reference"`
	Type      string `json:"type,omitempty"`
}

func InspectArtifact(service *artifactusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req artifactRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_request", Message: "request body must include reference"})
			return
		}
		result, err := service.Inspect(r.Context(), req.Reference, domainartifact.ArtifactType(req.Type))
		respondArtifactResult(w, r, result, err)
	}
}

func ResolveArtifact(service *artifactusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req artifactRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_request", Message: "request body must include reference"})
			return
		}
		result, err := service.Resolve(r.Context(), req.Reference, domainartifact.ArtifactType(req.Type))
		respondArtifactResult(w, r, result, err)
	}
}

type artifactRegistryValidateRequest struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Endpoint string `json:"endpoint"`
	Insecure bool   `json:"insecure,omitempty"`
}

func ValidateArtifactRegistry() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req artifactRegistryValidateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_request", Message: "request body must be an artifact registry config"})
			return
		}
		if req.Name == "" || req.Type == "" || req.Endpoint == "" {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_artifact_registry", Message: "name, type, and endpoint are required"})
			return
		}
		if req.Type != "oci" {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "unsupported_artifact_registry", Message: "only generic OCI registry configuration is supported in this phase"})
			return
		}
		RespondJSON(w, http.StatusOK, map[string]any{
			"valid":    true,
			"name":     req.Name,
			"type":     req.Type,
			"endpoint": req.Endpoint,
			"insecure": req.Insecure,
			"warnings": registryWarnings(req),
		})
	}
}

func registryWarnings(req artifactRegistryValidateRequest) []string {
	if req.Insecure {
		return []string{"insecure OCI registry configuration is for local development only"}
	}
	return nil
}

func CreateRelease(service *artifactusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var def artifactusecase.ReleaseDefinition
		if err := json.NewDecoder(r.Body).Decode(&def); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_request", Message: "request body must be a release definition"})
			return
		}
		record, err := service.CreateRelease(r.Context(), artifactusecase.CreateReleaseInput{Definition: def})
		if err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "release_create_failed", Message: err.Error()})
			return
		}
		RespondJSON(w, http.StatusCreated, record)
	}
}

func ListReleases(service *artifactusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		records, err := service.ListReleases(r.Context())
		respondArtifactResult(w, r, records, err)
	}
}

func GetRelease(service *artifactusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		record, err := service.GetRelease(r.Context(), chi.URLParam(r, "id"))
		respondArtifactResult(w, r, record, err)
	}
}

func GetReleaseArtifacts(service *artifactusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		artifacts, err := service.ReleaseArtifacts(r.Context(), chi.URLParam(r, "id"))
		respondArtifactResult(w, r, artifacts, err)
	}
}

func respondArtifactResult(w http.ResponseWriter, r *http.Request, payload any, err error) {
	if err == nil {
		RespondJSON(w, http.StatusOK, payload)
		return
	}
	status := http.StatusBadRequest
	code := "artifact_error"
	if errors.Is(err, artifactusecase.ErrReleaseNotFound) {
		status = http.StatusNotFound
		code = "release_not_found"
	}
	RespondError(w, r, status, dto.ErrorResponse{Code: code, Message: err.Error()})
}
