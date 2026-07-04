package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sevoniva/nivora/internal/api/http/dto"
	complianceusecase "github.com/sevoniva/nivora/internal/usecase/compliance"
)

func SearchAudit(service *complianceusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		input := complianceusecase.AuditSearchInput{
			Subject:       r.URL.Query().Get("subject"),
			ActorID:       r.URL.Query().Get("actorId"),
			Action:        r.URL.Query().Get("action"),
			ScopeType:     r.URL.Query().Get("scopeType"),
			ScopeID:       r.URL.Query().Get("scopeId"),
			CorrelationID: r.URL.Query().Get("correlationId"),
		}
		result, err := service.SearchAudit(r.Context(), input)
		if err != nil {
			respondComplianceResult(w, r, nil, err)
			return
		}
		page, pageErr := parsePagination(r)
		if pageErr != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_pagination", Message: pageErr.Error()})
			return
		}
		if page.Enabled {
			RespondJSON(w, http.StatusOK, paginatedPayload(result.Items, page))
			return
		}
		RespondJSON(w, http.StatusOK, result)
	}
}

func GetEvidenceBundle(service *complianceusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bundle, err := service.EvidenceBundle(r.Context(), complianceusecase.EvidenceInput{SubjectType: chi.URLParam(r, "subject_type"), SubjectID: chi.URLParam(r, "id")})
		if err != nil {
			respondComplianceResult(w, r, nil, err)
			return
		}
		if r.URL.Query().Get("format") == "markdown" {
			w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
			_, _ = w.Write([]byte(service.ExportMarkdown(bundle)))
			return
		}
		RespondJSON(w, http.StatusOK, bundle)
	}
}

func GenerateEvidenceBundle(service *complianceusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input complianceusecase.EvidenceInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_request", Message: "request body must be an evidence generation request"})
			return
		}
		if input.SubjectType == "" || input.SubjectID == "" {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_request", Message: "subjectType and subjectId are required"})
			return
		}
		bundle, err := service.EvidenceBundle(r.Context(), input)
		if err != nil {
			respondComplianceResult(w, r, nil, err)
			return
		}
		RespondJSON(w, http.StatusCreated, bundle)
	}
}

func GenerateReleaseEvidenceBundle(service *complianceusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		releaseID := chi.URLParam(r, "id")
		if releaseID == "" {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_request", Message: "release id is required"})
			return
		}
		bundle, err := service.EvidenceBundle(r.Context(), complianceusecase.EvidenceInput{SubjectType: "release", SubjectID: releaseID})
		if err != nil {
			respondComplianceResult(w, r, nil, err)
			return
		}
		RespondJSON(w, http.StatusCreated, bundle)
	}
}

func GetEvidenceBundleByID(service *complianceusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bundle, err := service.GetEvidenceBundle(r.Context(), chi.URLParam(r, "id"))
		respondComplianceResult(w, r, bundle, err)
	}
}

func ExportEvidenceBundleByID(service *complianceusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bundle, err := service.GetEvidenceBundle(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			respondComplianceResult(w, r, nil, err)
			return
		}
		if r.URL.Query().Get("format") == "markdown" {
			w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
			_, _ = w.Write([]byte(service.ExportMarkdown(bundle)))
			return
		}
		RespondJSON(w, http.StatusOK, bundle)
	}
}

func GetRetentionPolicy(service *complianceusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		policy, err := service.RetentionPolicy(r.Context(), r.URL.Query().Get("scopeType"), r.URL.Query().Get("scopeId"))
		respondComplianceResult(w, r, policy, err)
	}
}

func SetRetentionPolicy(service *complianceusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input complianceusecase.RetentionInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_request", Message: "request body must be a retention policy request"})
			return
		}
		policy, err := service.SetRetentionPolicy(r.Context(), input)
		respondComplianceResult(w, r, policy, err)
	}
}

func VerifyAuditChain(service *complianceusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := service.VerifyAuditChain(r.Context(), r.URL.Query().Get("scopeType"), r.URL.Query().Get("scopeId"))
		respondComplianceResult(w, r, result, err)
	}
}

func respondComplianceResult(w http.ResponseWriter, r *http.Request, payload any, err error) {
	if err == nil {
		RespondJSON(w, http.StatusOK, payload)
		return
	}
	RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "compliance_error", Message: err.Error()})
}
