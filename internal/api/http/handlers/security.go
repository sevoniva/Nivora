package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/sevoniva/nivora/internal/api/http/dto"
	domainsecurity "github.com/sevoniva/nivora/internal/domain/security"
	"github.com/sevoniva/nivora/internal/domain/tenant"
	"github.com/sevoniva/nivora/internal/infra/telemetry"
	policyusecase "github.com/sevoniva/nivora/internal/usecase/policy"
	securityusecase "github.com/sevoniva/nivora/internal/usecase/security"
)

func CreateSecurityScan(service *securityusecase.Service, policyCatalog ...*policyusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input securityusecase.ScanInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_request", Message: "request body must be a security scan request"})
			return
		}
		input.ProjectID, input.EnvironmentID = constrainSecurityScope(r, input.ProjectID, input.EnvironmentID)
		if len(policyCatalog) > 0 && policyCatalog[0] != nil {
			if err := applySavedPolicyForScan(r.Context(), policyCatalog[0], &input); err != nil {
				respondPolicyCatalogError(w, r, err)
				return
			}
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

func applySavedPolicyForScan(ctx context.Context, service *policyusecase.Service, input *securityusecase.ScanInput) error {
	if policyID := strings.TrimSpace(input.PolicyID); policyID != "" {
		policy, err := service.GetEnabled(ctx, policyID)
		if err != nil {
			return err
		}
		securityusecase.ApplyPolicyDefinition(policy, input)
		return nil
	}
	if !isZeroPolicyConfig(input.Policy) {
		return nil
	}
	policy, ok, err := service.ResolveEnabledForScope(ctx, policyusecase.ResolveInput{
		ProjectID:     input.ProjectID,
		EnvironmentID: input.EnvironmentID,
	})
	if err != nil || !ok {
		return err
	}
	securityusecase.ApplyPolicyDefinition(policy, input)
	return nil
}

func isZeroPolicyConfig(policy securityusecase.PolicyConfig) bool {
	return policy.CriticalDenyThreshold == 0 &&
		policy.HighWarnThreshold == 0 &&
		!policy.RequireDigest &&
		!policy.ApprovalOnCritical
}

func ListSecurityScans(service *securityusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		projectID, environmentID := constrainSecurityScope(r, query.Get("projectId"), query.Get("environmentId"))
		records, err := service.ListScans(r.Context(), securityusecase.ListScansInput{
			SubjectType:   domainsecurity.SubjectType(query.Get("subjectType")),
			SubjectID:     query.Get("subjectId"),
			ProjectID:     projectID,
			EnvironmentID: environmentID,
			Status:        domainsecurity.ScanStatus(query.Get("status")),
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
		if err == nil && !securityScanVisibleToRequest(r, record.Scan) {
			RespondError(w, r, http.StatusForbidden, dto.ErrorResponse{Code: "forbidden", Message: "security scan is outside requester scope"})
			return
		}
		respondSecurityResult(w, r, record, err)
	}
}

func ListSecurityFindings(service *securityusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		projectID, environmentID := constrainSecurityScope(r, query.Get("projectId"), query.Get("environmentId"))
		findings, err := service.ListFindings(r.Context(), securityusecase.ListFindingsInput{
			ScanID:        query.Get("scanId"),
			SubjectType:   domainsecurity.SubjectType(query.Get("subjectType")),
			SubjectID:     query.Get("subjectId"),
			ProjectID:     projectID,
			EnvironmentID: environmentID,
			Severity:      domainsecurity.Severity(query.Get("severity")),
			Category:      domainsecurity.FindingCategory(query.Get("category")),
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

func GetSecurityFinding(service *securityusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID, environmentID := constrainSecurityScope(r, "", "")
		finding, err := service.GetFinding(r.Context(), securityusecase.GetFindingInput{
			FindingID:     chi.URLParam(r, "id"),
			ProjectID:     projectID,
			EnvironmentID: environmentID,
		})
		respondSecurityResult(w, r, finding, err)
	}
}

func GetSecurityFindings(service *securityusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		record, err := service.Get(r.Context(), chi.URLParam(r, "id"))
		if err == nil && !securityScanVisibleToRequest(r, record.Scan) {
			RespondError(w, r, http.StatusForbidden, dto.ErrorResponse{Code: "forbidden", Message: "security scan is outside requester scope"})
			return
		}
		respondSecurityResult(w, r, record.Scan.Findings, err)
	}
}

func EvaluatePolicy(service *securityusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input securityusecase.EvaluateInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_request", Message: "request body must be a policy evaluation request"})
			return
		}
		input.ProjectID, input.EnvironmentID = constrainSecurityScope(r, input.ProjectID, input.EnvironmentID)
		result, err := service.EvaluateAndStore(r.Context(), input)
		if err == nil && result.Decision == domainsecurity.GateDeny {
			telemetry.DefaultMetrics().IncPolicyDenial()
		}
		respondSecurityResult(w, r, result, err)
	}
}

func ListPolicyResults(service *securityusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		projectID, environmentID := constrainSecurityScope(r, query.Get("projectId"), query.Get("environmentId"))
		results, err := service.ListPolicyResults(r.Context(), securityusecase.ListPolicyResultsInput{
			PolicyID:      query.Get("policyId"),
			SubjectType:   domainsecurity.SubjectType(query.Get("subjectType")),
			SubjectID:     query.Get("subjectId"),
			ProjectID:     projectID,
			EnvironmentID: environmentID,
			Decision:      domainsecurity.GateDecision(query.Get("decision")),
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
		payload := map[string]any{"results": results}
		if page.Enabled {
			page.Total = len(results)
			paged := paginatedPayload(results, page).(map[string]any)
			payload["results"] = paged["items"]
			payload["pagination"] = paged["pagination"]
		}
		RespondJSON(w, http.StatusOK, payload)
	}
}

func GetPolicyResult(service *securityusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID, environmentID := constrainSecurityScope(r, "", "")
		result, err := service.GetPolicyResult(r.Context(), securityusecase.GetPolicyResultInput{
			ResultID:      chi.URLParam(r, "id"),
			ProjectID:     projectID,
			EnvironmentID: environmentID,
		})
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
	} else if errors.Is(err, securityusecase.ErrFindingNotFound) {
		status = http.StatusNotFound
		code = "security_finding_not_found"
	} else if errors.Is(err, securityusecase.ErrPolicyResultNotFound) {
		status = http.StatusNotFound
		code = "security_policy_result_not_found"
	}
	RespondError(w, r, status, dto.ErrorResponse{Code: code, Message: err.Error()})
}

func constrainSecurityScope(r *http.Request, projectID string, environmentID string) (string, string) {
	scopeType, scopeID := TenantScopeFilter(r)
	switch scopeType {
	case tenant.ScopeProject:
		return scopeID, ""
	case tenant.ScopeEnvironment:
		return "", scopeID
	default:
		return projectID, environmentID
	}
}

func securityScanVisibleToRequest(r *http.Request, scan domainsecurity.SecurityScan) bool {
	scopeType, scopeID := TenantScopeFilter(r)
	switch scopeType {
	case "":
		return true
	case tenant.ScopeProject:
		return scan.ProjectID != "" && scan.ProjectID == scopeID
	case tenant.ScopeEnvironment:
		return scan.EnvironmentID != "" && scan.EnvironmentID == scopeID
	default:
		return false
	}
}
