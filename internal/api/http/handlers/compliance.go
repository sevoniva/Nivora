package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sevoniva/nivora/internal/api/http/dto"
	domaincompliance "github.com/sevoniva/nivora/internal/domain/compliance"
	complianceusecase "github.com/sevoniva/nivora/internal/usecase/compliance"
)

func SearchAudit(service *complianceusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		scopeType, scopeID := ConstrainScopeToRequest(r, r.URL.Query().Get("scopeType"), r.URL.Query().Get("scopeId"))
		input := complianceusecase.AuditSearchInput{
			Subject:       r.URL.Query().Get("subject"),
			SubjectType:   r.URL.Query().Get("subjectType"),
			SubjectID:     r.URL.Query().Get("subjectId"),
			ActorID:       r.URL.Query().Get("actorId"),
			Action:        r.URL.Query().Get("action"),
			ScopeType:     scopeType,
			ScopeID:       scopeID,
			RequestID:     r.URL.Query().Get("requestId"),
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
		subjectType := chi.URLParam(r, "subject_type")
		subjectID := chi.URLParam(r, "id")
		if !ensureEvidenceSubjectAllowed(w, r, service, subjectType, subjectID) {
			return
		}
		bundle, err := service.EvidenceBundle(r.Context(), complianceusecase.EvidenceInput{SubjectType: subjectType, SubjectID: subjectID})
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
		if !ensureEvidenceSubjectAllowed(w, r, service, input.SubjectType, input.SubjectID) {
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

func ListEvidenceBundles(service *complianceusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bundles, err := service.SearchEvidenceBundles(r.Context(), r.URL.Query().Get("subjectType"), r.URL.Query().Get("subjectId"))
		if err != nil {
			respondComplianceResult(w, r, nil, err)
			return
		}
		bundles = filterEvidenceBundlesForRequest(r, service, bundles)
		page, pageErr := parsePagination(r)
		if pageErr != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_pagination", Message: pageErr.Error()})
			return
		}
		if page.Enabled {
			RespondJSON(w, http.StatusOK, paginatedPayload(bundles, page))
			return
		}
		RespondJSON(w, http.StatusOK, map[string]any{"bundles": bundles, "count": len(bundles)})
	}
}

func GenerateReleaseEvidenceBundle(service *complianceusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		releaseID := chi.URLParam(r, "id")
		if releaseID == "" {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_request", Message: "release id is required"})
			return
		}
		if !ensureEvidenceSubjectAllowed(w, r, service, "release", releaseID) {
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
		if err == nil && !evidenceBundleAllowedForRequest(r, service, bundle) {
			RespondError(w, r, http.StatusForbidden, dto.ErrorResponse{Code: "forbidden", Message: "evidence bundle is outside the caller scope"})
			return
		}
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
		if !evidenceBundleAllowedForRequest(r, service, bundle) {
			RespondError(w, r, http.StatusForbidden, dto.ErrorResponse{Code: "forbidden", Message: "evidence bundle is outside the caller scope"})
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
		scopeType, scopeID := ConstrainScopeToRequest(r, r.URL.Query().Get("scopeType"), r.URL.Query().Get("scopeId"))
		policy, err := service.RetentionPolicy(r.Context(), scopeType, scopeID)
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
		input.ScopeType, input.ScopeID = ConstrainScopeToRequest(r, input.ScopeType, input.ScopeID)
		policy, err := service.SetRetentionPolicy(r.Context(), input)
		respondComplianceResult(w, r, policy, err)
	}
}

func RunRetentionPolicy(service *complianceusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input complianceusecase.RetentionRunInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_request", Message: "request body must be a retention run request"})
			return
		}
		input.ScopeType, input.ScopeID = ConstrainScopeToRequest(r, input.ScopeType, input.ScopeID)
		result, err := service.RunRetention(r.Context(), input)
		respondComplianceResult(w, r, result, err)
	}
}

func VerifyAuditChain(service *complianceusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		scopeType, scopeID := ConstrainScopeToRequest(r, r.URL.Query().Get("scopeType"), r.URL.Query().Get("scopeId"))
		result, err := service.VerifyAuditChain(r.Context(), scopeType, scopeID)
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

func ensureEvidenceSubjectAllowed(w http.ResponseWriter, r *http.Request, service *complianceusecase.Service, subjectType, subjectID string) bool {
	requestScopeType, requestScopeID := TenantScopeFilter(r)
	if requestScopeType == "" {
		return true
	}
	scope, err := service.SubjectScope(r.Context(), subjectType, subjectID)
	if err != nil {
		respondComplianceResult(w, r, nil, err)
		return false
	}
	if !scope.Verified || !scopeMatchesRequest(requestScopeType, requestScopeID, scope.ScopeType, scope.ScopeID) {
		RespondError(w, r, http.StatusForbidden, dto.ErrorResponse{Code: "forbidden", Message: "evidence subject is outside the caller scope"})
		return false
	}
	return true
}

func filterEvidenceBundlesForRequest(r *http.Request, service *complianceusecase.Service, bundles []domaincompliance.EvidenceBundle) []domaincompliance.EvidenceBundle {
	if !ScopedByTenant(r) {
		return bundles
	}
	filtered := make([]domaincompliance.EvidenceBundle, 0, len(bundles))
	for _, bundle := range bundles {
		if evidenceBundleAllowedForRequest(r, service, bundle) {
			filtered = append(filtered, bundle)
		}
	}
	return filtered
}

func evidenceBundleAllowedForRequest(r *http.Request, service *complianceusecase.Service, bundle domaincompliance.EvidenceBundle) bool {
	requestScopeType, requestScopeID := TenantScopeFilter(r)
	if requestScopeType == "" {
		return true
	}
	if bundle.ScopeType != "" || bundle.ScopeID != "" {
		return scopeMatchesRequest(requestScopeType, requestScopeID, bundle.ScopeType, bundle.ScopeID)
	}
	scope, err := service.SubjectScope(r.Context(), bundle.SubjectType, bundle.SubjectID)
	if err != nil || !scope.Verified {
		return false
	}
	return scopeMatchesRequest(requestScopeType, requestScopeID, scope.ScopeType, scope.ScopeID)
}

func scopeMatchesRequest(requestScopeType, requestScopeID, resourceScopeType, resourceScopeID string) bool {
	return requestScopeType != "" &&
		requestScopeID != "" &&
		requestScopeType == resourceScopeType &&
		requestScopeID == resourceScopeID
}
