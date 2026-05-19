package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sevoniva/nivora/internal/api/http/dto"
	securityusecase "github.com/sevoniva/nivora/internal/usecase/security"
)

func CreateSecurityScan(service *securityusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input securityusecase.ScanInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_request", Message: "request body must be a security scan request"})
			return
		}
		record, err := service.Scan(r.Context(), input)
		if err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "security_scan_failed", Message: err.Error()})
			return
		}
		RespondJSON(w, http.StatusCreated, record)
	}
}

func GetSecurityScan(service *securityusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		record, err := service.Get(r.Context(), chi.URLParam(r, "id"))
		respondSecurityResult(w, r, record, err)
	}
}

func GetSecurityFindings(service *securityusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		findings, err := service.Findings(r.Context(), chi.URLParam(r, "id"))
		respondSecurityResult(w, r, findings, err)
	}
}

func EvaluatePolicy(service *securityusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input securityusecase.EvaluateInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_request", Message: "request body must be a policy evaluation request"})
			return
		}
		result, err := service.EvaluateAndStore(r.Context(), input)
		respondSecurityResult(w, r, result, err)
	}
}

func respondSecurityResult(w http.ResponseWriter, r *http.Request, payload any, err error) {
	if err == nil {
		RespondJSON(w, http.StatusOK, payload)
		return
	}
	status := http.StatusBadRequest
	code := "security_error"
	if errors.Is(err, securityusecase.ErrScanNotFound) {
		status = http.StatusNotFound
		code = "security_scan_not_found"
	}
	RespondError(w, r, status, dto.ErrorResponse{Code: code, Message: err.Error()})
}
