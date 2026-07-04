package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sevoniva/nivora/internal/api/http/dto"
	catalogusecase "github.com/sevoniva/nivora/internal/usecase/catalog"
)

func ListOrgs(service *catalogusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgs, err := service.ListOrgs(r.Context())
		if err != nil {
			respondCatalogError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, map[string]any{"orgs": orgs})
	}
}

func CreateOrg(service *catalogusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input catalogusecase.CreateOrgInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_json", Message: "request body must be valid JSON"})
			return
		}
		org, err := service.CreateOrg(r.Context(), input)
		if err != nil {
			respondCatalogError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusCreated, org)
	}
}

func GetOrg(service *catalogusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		org, err := service.GetOrg(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			respondCatalogError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, org)
	}
}

func UpdateOrg(service *catalogusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input catalogusecase.UpdateOrgInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_json", Message: "request body must be valid JSON"})
			return
		}
		org, err := service.UpdateOrg(r.Context(), chi.URLParam(r, "id"), input)
		if err != nil {
			respondCatalogError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, org)
	}
}

func DisableOrg(service *catalogusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		org, err := service.DisableOrg(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			respondCatalogError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, org)
	}
}

func ListProjects(service *catalogusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projects, err := service.ListProjects(r.Context(), r.URL.Query().Get("orgId"))
		if err != nil {
			respondCatalogError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, map[string]any{"projects": projects})
	}
}

func CreateProject(service *catalogusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input catalogusecase.CreateProjectInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_json", Message: "request body must be valid JSON"})
			return
		}
		project, err := service.CreateProject(r.Context(), input)
		if err != nil {
			respondCatalogError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusCreated, project)
	}
}

func GetProject(service *catalogusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		project, err := service.GetProject(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			respondCatalogError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, project)
	}
}

func UpdateProject(service *catalogusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input catalogusecase.UpdateProjectInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_json", Message: "request body must be valid JSON"})
			return
		}
		project, err := service.UpdateProject(r.Context(), chi.URLParam(r, "id"), input)
		if err != nil {
			respondCatalogError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, project)
	}
}

func DisableProject(service *catalogusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		project, err := service.DisableProject(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			respondCatalogError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, project)
	}
}

func ListApplications(service *catalogusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		applications, err := service.ListApplications(r.Context(), r.URL.Query().Get("projectId"))
		if err != nil {
			respondCatalogError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, map[string]any{"applications": applications})
	}
}

func CreateApplication(service *catalogusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input catalogusecase.CreateApplicationInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_json", Message: "request body must be valid JSON"})
			return
		}
		application, err := service.CreateApplication(r.Context(), input)
		if err != nil {
			respondCatalogError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusCreated, application)
	}
}

func GetApplication(service *catalogusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		application, err := service.GetApplication(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			respondCatalogError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, application)
	}
}

func UpdateApplication(service *catalogusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input catalogusecase.UpdateApplicationInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_json", Message: "request body must be valid JSON"})
			return
		}
		application, err := service.UpdateApplication(r.Context(), chi.URLParam(r, "id"), input)
		if err != nil {
			respondCatalogError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, application)
	}
}

func DisableApplication(service *catalogusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		application, err := service.DisableApplication(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			respondCatalogError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, application)
	}
}

func ListEnvironments(service *catalogusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		environments, err := service.ListEnvironments(r.Context(), r.URL.Query().Get("projectId"))
		if err != nil {
			respondCatalogError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, map[string]any{"environments": environments})
	}
}

func CreateEnvironment(service *catalogusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input catalogusecase.CreateEnvironmentInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_json", Message: "request body must be valid JSON"})
			return
		}
		environment, err := service.CreateEnvironment(r.Context(), input)
		if err != nil {
			respondCatalogError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusCreated, environment)
	}
}

func GetEnvironment(service *catalogusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		environment, err := service.GetEnvironment(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			respondCatalogError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, environment)
	}
}

func UpdateEnvironment(service *catalogusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input catalogusecase.UpdateEnvironmentInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_json", Message: "request body must be valid JSON"})
			return
		}
		environment, err := service.UpdateEnvironment(r.Context(), chi.URLParam(r, "id"), input)
		if err != nil {
			respondCatalogError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, environment)
	}
}

func DisableEnvironment(service *catalogusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		environment, err := service.DisableEnvironment(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			respondCatalogError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, environment)
	}
}

func ListRepositories(service *catalogusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repositories, err := service.ListRepositories(r.Context(), r.URL.Query().Get("projectId"))
		if err != nil {
			respondCatalogError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, map[string]any{"repositories": repositories})
	}
}

func CreateRepository(service *catalogusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input catalogusecase.CreateRepositoryInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_json", Message: "request body must be valid JSON"})
			return
		}
		repository, err := service.CreateRepository(r.Context(), input)
		if err != nil {
			respondCatalogError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusCreated, repository)
	}
}

func GetRepository(service *catalogusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repository, err := service.GetRepository(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			respondCatalogError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, repository)
	}
}

func UpdateRepository(service *catalogusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input catalogusecase.UpdateRepositoryInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_json", Message: "request body must be valid JSON"})
			return
		}
		repository, err := service.UpdateRepository(r.Context(), chi.URLParam(r, "id"), input)
		if err != nil {
			respondCatalogError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, repository)
	}
}

func DisableRepository(service *catalogusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repository, err := service.DisableRepository(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			respondCatalogError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, repository)
	}
}

func ValidateRepository(service *catalogusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := service.ValidateRepository(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			respondCatalogError(w, r, err)
			return
		}
		status := http.StatusOK
		if !result.Valid {
			status = http.StatusBadRequest
		}
		RespondJSON(w, status, result)
	}
}

func ListReleaseTargets(service *catalogusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		targets, err := service.ListReleaseTargets(r.Context(), r.URL.Query().Get("projectId"), r.URL.Query().Get("environmentId"))
		if err != nil {
			respondCatalogError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, map[string]any{"releaseTargets": targets})
	}
}

func CreateReleaseTarget(service *catalogusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input catalogusecase.CreateReleaseTargetInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_json", Message: "request body must be valid JSON"})
			return
		}
		target, err := service.CreateReleaseTarget(r.Context(), input)
		if err != nil {
			respondCatalogError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusCreated, target)
	}
}

func GetReleaseTarget(service *catalogusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		target, err := service.GetReleaseTarget(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			respondCatalogError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, target)
	}
}

func UpdateReleaseTarget(service *catalogusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input catalogusecase.UpdateReleaseTargetInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_json", Message: "request body must be valid JSON"})
			return
		}
		target, err := service.UpdateReleaseTarget(r.Context(), chi.URLParam(r, "id"), input)
		if err != nil {
			respondCatalogError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, target)
	}
}

func DisableReleaseTarget(service *catalogusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		target, err := service.DisableReleaseTarget(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			respondCatalogError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, target)
	}
}

func ValidateReleaseTarget(service *catalogusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := service.ValidateReleaseTarget(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			respondCatalogError(w, r, err)
			return
		}
		status := http.StatusOK
		if !result.Valid {
			status = http.StatusBadRequest
		}
		RespondJSON(w, status, result)
	}
}

func respondCatalogError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, catalogusecase.ErrInvalid):
		RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_catalog_resource", Message: err.Error()})
	case errors.Is(err, catalogusecase.ErrAlreadyExists):
		RespondError(w, r, http.StatusConflict, dto.ErrorResponse{Code: "catalog_resource_exists", Message: err.Error()})
	case errors.Is(err, catalogusecase.ErrNotFound):
		RespondError(w, r, http.StatusNotFound, dto.ErrorResponse{Code: "catalog_resource_not_found", Message: err.Error()})
	default:
		RespondError(w, r, http.StatusInternalServerError, dto.ErrorResponse{Code: "catalog_error", Message: err.Error()})
	}
}
