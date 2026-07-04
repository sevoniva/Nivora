package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sevoniva/nivora/internal/api/http/dto"
	apimiddleware "github.com/sevoniva/nivora/internal/api/http/middleware"
	domainartifact "github.com/sevoniva/nivora/internal/domain/artifact"
	"github.com/sevoniva/nivora/internal/domain/tenant"
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

func ListArtifacts(service *artifactusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		artifacts, err := service.ListArtifacts(r.Context(), artifactusecase.ListArtifactsInput{
			Type:       r.URL.Query().Get("type"),
			Name:       r.URL.Query().Get("name"),
			Registry:   r.URL.Query().Get("registry"),
			Repository: r.URL.Query().Get("repository"),
			Digest:     r.URL.Query().Get("digest"),
			Reference:  r.URL.Query().Get("reference"),
		})
		respondArtifactResult(w, r, map[string]any{"artifacts": artifacts}, err)
	}
}

func GetArtifact(service *artifactusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		artifact, err := service.GetArtifact(r.Context(), chi.URLParam(r, "id"))
		respondArtifactResult(w, r, artifact, err)
	}
}

func GetArtifactReleases(service *artifactusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bindings, err := service.ArtifactReleases(r.Context(), chi.URLParam(r, "id"))
		respondArtifactResult(w, r, map[string]any{"releases": bindings}, err)
	}
}

type artifactRegistryValidateRequest struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Endpoint string `json:"endpoint"`
	Insecure bool   `json:"insecure,omitempty"`
}

func ListArtifactRegistries(service *artifactusecase.RegistryService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		registries, err := service.List(r.Context(), r.URL.Query().Get("projectId"))
		respondArtifactResult(w, r, map[string]any{"registries": registries}, err)
	}
}

func CreateArtifactRegistry(service *artifactusecase.RegistryService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input artifactusecase.RegistryCreateInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_request", Message: "request body must be an artifact registry config"})
			return
		}
		registry, err := service.Create(r.Context(), input)
		if err != nil {
			respondArtifactResult(w, r, registry, err)
			return
		}
		RespondJSON(w, http.StatusCreated, registry)
	}
}

func GetArtifactRegistry(service *artifactusecase.RegistryService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		registry, err := service.Get(r.Context(), chi.URLParam(r, "id"))
		respondArtifactResult(w, r, registry, err)
	}
}

func UpdateArtifactRegistry(service *artifactusecase.RegistryService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input artifactusecase.RegistryUpdateInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_request", Message: "request body must be an artifact registry update"})
			return
		}
		registry, err := service.Update(r.Context(), chi.URLParam(r, "id"), input)
		respondArtifactResult(w, r, registry, err)
	}
}

func DisableArtifactRegistry(service *artifactusecase.RegistryService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		registry, err := service.Disable(r.Context(), chi.URLParam(r, "id"))
		respondArtifactResult(w, r, registry, err)
	}
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
		projectID := ""
		if scopeType, scopeID := TenantScopeFilter(r); scopeType == tenant.ScopeProject {
			projectID = scopeID
		}
		record, err := service.CreateRelease(r.Context(), artifactusecase.CreateReleaseInput{Definition: def, ProjectID: projectID, ActorID: apimiddleware.Subject(r.Context()).ID})
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
		if err != nil {
			respondArtifactResult(w, r, nil, err)
			return
		}
		respondArtifactResult(w, r, filterReleaseRecordsForRequest(r, records), nil)
	}
}

func GetRelease(service *artifactusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		record, ok := getAuthorizedReleaseRecord(w, r, service)
		if !ok {
			return
		}
		respondArtifactResult(w, r, record, nil)
	}
}

func GetReleaseArtifacts(service *artifactusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := getAuthorizedReleaseRecord(w, r, service); !ok {
			return
		}
		artifacts, err := service.ReleaseArtifacts(r.Context(), chi.URLParam(r, "id"))
		respondArtifactResult(w, r, artifacts, err)
	}
}

func CancelRelease(service *artifactusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		record, ok := getAuthorizedReleaseRecord(w, r, service)
		if !ok {
			return
		}
		record, err := service.CancelRelease(r.Context(), record.Release.ID, apimiddleware.Subject(r.Context()).ID)
		respondArtifactResult(w, r, record, err)
	}
}

func filterReleaseRecordsForRequest(r *http.Request, records []artifactusecase.ReleaseRecord) []artifactusecase.ReleaseRecord {
	scopeType, _ := TenantScopeFilter(r)
	if scopeType == "" {
		return records
	}
	filtered := make([]artifactusecase.ReleaseRecord, 0, len(records))
	for _, record := range records {
		if releaseRecordInRequestScope(r, record) {
			filtered = append(filtered, record)
		}
	}
	return filtered
}

func getAuthorizedReleaseRecord(w http.ResponseWriter, r *http.Request, service *artifactusecase.Service) (artifactusecase.ReleaseRecord, bool) {
	record, err := service.GetRelease(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		respondArtifactResult(w, r, nil, err)
		return artifactusecase.ReleaseRecord{}, false
	}
	if releaseRecordInRequestScope(r, record) {
		return record, true
	}
	RespondError(w, r, http.StatusForbidden, dto.ErrorResponse{Code: "forbidden", Message: "release is outside requester scope", Path: r.URL.Path})
	return artifactusecase.ReleaseRecord{}, false
}

func releaseRecordInRequestScope(r *http.Request, record artifactusecase.ReleaseRecord) bool {
	scopeType, scopeID := TenantScopeFilter(r)
	if scopeType == "" {
		return true
	}
	if scopeID == "" {
		return false
	}
	switch scopeType {
	case tenant.ScopeProject:
		projectID := record.Release.Metadata["projectId"]
		return projectID == "" || projectID == scopeID
	case tenant.ScopeEnvironment:
		return record.Release.EnvironmentID == "" || record.Release.EnvironmentID == scopeID
	default:
		return false
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
	} else if errors.Is(err, artifactusecase.ErrReleaseAlreadyTerminal) {
		status = http.StatusConflict
		code = "release_terminal"
	} else if errors.Is(err, artifactusecase.ErrArtifactNotFound) {
		status = http.StatusNotFound
		code = "artifact_not_found"
	} else if errors.Is(err, artifactusecase.ErrRegistryNotFound) {
		status = http.StatusNotFound
		code = "artifact_registry_not_found"
	} else if errors.Is(err, artifactusecase.ErrRegistryAlreadyExists) {
		status = http.StatusConflict
		code = "artifact_registry_already_exists"
	} else if errors.Is(err, artifactusecase.ErrRegistryInvalid) {
		status = http.StatusBadRequest
		code = "invalid_artifact_registry"
	}
	RespondError(w, r, status, dto.ErrorResponse{Code: code, Message: err.Error()})
}
