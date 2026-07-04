package handlers

import (
	"net/http"

	apimiddleware "github.com/sevoniva/nivora/internal/api/http/middleware"
	"github.com/sevoniva/nivora/internal/domain/tenant"
)

// TenantScopeFilter extracts the requesting subject's scope from the request
// context. Returns empty scope when the subject is not tenant-scoped (global access).
func TenantScopeFilter(r *http.Request) (scopeType, scopeID string) {
	subject := apimiddleware.Subject(r.Context())
	if subject.ScopeType == "" || subject.ScopeType == tenant.ScopeGlobal {
		return "", ""
	}
	if subject.ScopeType == tenant.ScopeOrg ||
		subject.ScopeType == tenant.ScopeProject ||
		subject.ScopeType == tenant.ScopeEnvironment {
		return subject.ScopeType, subject.ScopeID
	}
	return "", ""
}

// ScopedByTenant returns true when the subject is tenant-scoped and should have
// list/search/visualization results filtered.
func ScopedByTenant(r *http.Request) bool {
	scopeType, _ := TenantScopeFilter(r)
	return scopeType != ""
}

// ConstrainScopeToRequest keeps unscoped/admin requests unchanged and forces
// scoped subjects to query only within their own tenant boundary.
func ConstrainScopeToRequest(r *http.Request, scopeType, scopeID string) (string, string) {
	requestScopeType, requestScopeID := TenantScopeFilter(r)
	if requestScopeType == "" {
		return scopeType, scopeID
	}
	return requestScopeType, requestScopeID
}
