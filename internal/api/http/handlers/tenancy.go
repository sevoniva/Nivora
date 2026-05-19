package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/sevoniva/nivora/internal/api/http/dto"
	tenancyusecase "github.com/sevoniva/nivora/internal/usecase/tenancy"
)

func GetQuota(service *tenancyusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		quota, err := service.Quota(r.Context(), tenancyScope(r))
		respondTenancyResult(w, r, quota, err)
	}
}

func SetQuota(service *tenancyusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input tenancyusecase.QuotaUpdateInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_request", Message: "request body must be a quota update request"})
			return
		}
		quota, err := service.SetQuota(r.Context(), input)
		respondTenancyResult(w, r, quota, err)
	}
}

func GetUsageSummary(service *tenancyusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		usage, err := service.Usage(r.Context(), tenancyScope(r))
		respondTenancyResult(w, r, usage, err)
	}
}

func tenancyScope(r *http.Request) tenancyusecase.ScopeInput {
	query := r.URL.Query()
	return tenancyusecase.ScopeInput{ScopeType: query.Get("scopeType"), ScopeID: query.Get("scopeId")}
}

func respondTenancyResult(w http.ResponseWriter, r *http.Request, payload any, err error) {
	if err == nil {
		RespondJSON(w, http.StatusOK, payload)
		return
	}
	RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "tenancy_error", Message: err.Error()})
}
