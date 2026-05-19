package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sevoniva/nivora/internal/api/http/dto"
	cloudusecase "github.com/sevoniva/nivora/internal/usecase/cloud"
)

func ListCloudProviders(service *cloudusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := service.Providers(r.Context())
		respondCloudResult(w, r, result, err)
	}
}

func CreateCloudAccount(service *cloudusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input cloudusecase.CreateAccountInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_request", Message: "request body must be a cloud account request"})
			return
		}
		account, err := service.CreateAccount(r.Context(), input)
		respondCloudResult(w, r, account, err)
	}
}

func ListCloudAccounts(service *cloudusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		accounts, err := service.ListAccounts(r.Context())
		respondCloudResult(w, r, accounts, err)
	}
}

func GetCloudAccount(service *cloudusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		account, err := service.GetAccount(r.Context(), chi.URLParam(r, "id"))
		respondCloudResult(w, r, account, err)
	}
}

func ValidateCloudAccount(service *cloudusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := service.ValidateAccount(r.Context(), chi.URLParam(r, "id"))
		respondCloudResult(w, r, result, err)
	}
}

func ListCloudRegions(service *cloudusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := service.Regions(r.Context(), chi.URLParam(r, "id"))
		respondCloudResult(w, r, result, err)
	}
}

func ListCloudClusters(service *cloudusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := service.Clusters(r.Context(), chi.URLParam(r, "id"), r.URL.Query().Get("region"))
		respondCloudResult(w, r, result, err)
	}
}

func ListCloudHosts(service *cloudusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := service.Hosts(r.Context(), chi.URLParam(r, "id"), r.URL.Query().Get("region"))
		respondCloudResult(w, r, result, err)
	}
}

func ListCloudRegistries(service *cloudusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := service.Registries(r.Context(), chi.URLParam(r, "id"), r.URL.Query().Get("region"))
		respondCloudResult(w, r, result, err)
	}
}

func GetCloudInventory(service *cloudusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := service.Inventory(r.Context(), chi.URLParam(r, "id"))
		respondCloudResult(w, r, result, err)
	}
}

func respondCloudResult(w http.ResponseWriter, r *http.Request, payload any, err error) {
	if err == nil {
		RespondJSON(w, http.StatusOK, payload)
		return
	}
	status := http.StatusBadRequest
	code := "cloud_error"
	if errors.Is(err, cloudusecase.ErrAccountNotFound) {
		status = http.StatusNotFound
		code = "cloud_account_not_found"
	}
	RespondError(w, r, status, dto.ErrorResponse{Code: code, Message: err.Error()})
}
