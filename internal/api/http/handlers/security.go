package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sevoniva/nivora/internal/api/http/dto"
	domainsecurity "github.com/sevoniva/nivora/internal/domain/security"
	"github.com/sevoniva/nivora/internal/infra/telemetry"
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
		if record.Policy.Decision == domainsecurity.GateDeny {
			telemetry.DefaultMetrics().IncPolicyDenial()
		}
		RespondJSON(w, http.StatusCreated, record)
	}
}

func ListSecurityScans(service *securityusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		records, err := service.ListScans(r.Context(), securityusecase.ListScansInput{
			SubjectType: domainsecurity.SubjectType(query.Get("subjectType")),
			SubjectID:   query.Get("subjectId"),
			Status:      domainsecurity.ScanStatus(query.Get("status")),
		})
		if err != nil {
			respondSecurityResult(w, r, nil, err)
			return
		}
		page, pageErr := parsePagination(r)
		if pageErr != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_pagination", Message: pageErr.Error()})
			return
		}
		payload := map[string]any{"scans": records}
		if page.Enabled {
			page.Total = len(records)
			paged := paginatedPayload(records, page).(map[string]any)
			payload["scans"] = paged["items"]
			payload["pagination"] = paged["pagination"]
		}
		RespondJSON(w, http.StatusOK, payload)
	}
}

func GetSecurityScan(service *securityusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		record, err := service.Get(r.Context(), chi.URLParam(r, "id"))
		respondSecurityResult(w, r, record, err)
	}
}

func ListSecurityFindings(service *securityusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		findings, err := service.ListFindings(r.Context(), securityusecase.ListFindingsInput{
			ScanID:      query.Get("scanId"),
			SubjectType: domainsecurity.SubjectType(query.Get("subjectType")),
			SubjectID:   query.Get("subjectId"),
			Severity:    domainsecurity.Severity(query.Get("severity")),
			Category:    domainsecurity.FindingCategory(query.Get("category")),
		})
		if err != nil {
			respondSecurityResult(w, r, nil, err)
			return
		}
		page, pageErr := parsePagination(r)
		if pageErr != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_pagination", Message: pageErr.Error()})
			return
		}
		payload := map[string]any{"findings": findings}
		if page.Enabled {
			page.Total = len(findings)
			paged := paginatedPayload(findings, page).(map[string]any)
			payload["findings"] = paged["items"]
			payload["pagination"] = paged["pagination"]
		}
		RespondJSON(w, http.StatusOK, payload)
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
		if err == nil && result.Decision == domainsecurity.GateDeny {
			telemetry.DefaultMetrics().IncPolicyDenial()
		}
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
