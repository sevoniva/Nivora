package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sevoniva/nivora/internal/api/http/dto"
	policyusecase "github.com/sevoniva/nivora/internal/usecase/policy"
)

func ListPolicies(service *policyusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		policies, err := service.List(r.Context(), r.URL.Query().Get("projectId"), r.URL.Query().Get("environmentId"))
		if err != nil {
			respondPolicyCatalogError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, map[string]any{"policies": policies})
	}
}

func CreatePolicy(service *policyusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input policyusecase.CreateInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_json", Message: "request body must be valid JSON"})
			return
		}
		policy, err := service.Create(r.Context(), input)
		if err != nil {
			respondPolicyCatalogError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusCreated, policy)
	}
}

func GetPolicy(service *policyusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		policy, err := service.Get(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			respondPolicyCatalogError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, policy)
	}
}

func UpdatePolicy(service *policyusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input policyusecase.UpdateInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_json", Message: "request body must be valid JSON"})
			return
		}
		policy, err := service.Update(r.Context(), chi.URLParam(r, "id"), input)
		if err != nil {
			respondPolicyCatalogError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, policy)
	}
}

func DisablePolicy(service *policyusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		policy, err := service.Disable(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			respondPolicyCatalogError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, policy)
	}
}

func respondPolicyCatalogError(w http.ResponseWriter, r *http.Request, err error) {
	status := http.StatusBadRequest
	code := "policy_catalog_error"
	switch {
	case errors.Is(err, policyusecase.ErrInvalid):
		status = http.StatusBadRequest
		code = "invalid_policy"
	case errors.Is(err, policyusecase.ErrAlreadyExists):
		status = http.StatusConflict
		code = "policy_already_exists"
	case errors.Is(err, policyusecase.ErrNotFound):
		status = http.StatusNotFound
		code = "policy_not_found"
	}
	RespondError(w, r, status, dto.ErrorResponse{Code: code, Message: err.Error()})
}
