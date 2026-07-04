package handlers

import (
	"net/http"

	domainrunner "github.com/sevoniva/nivora/internal/domain/runner"
	"github.com/sevoniva/nivora/internal/domain/tenant"
)

func applyRunnerRequestScope(r *http.Request, runner *domainrunner.Runner) {
	if runner == nil {
		return
	}
	scopeType, scopeID := TenantScopeFilter(r)
	if scopeID == "" {
		return
	}
	if runner.Labels == nil {
		runner.Labels = map[string]string{}
	}
	switch scopeType {
	case tenant.ScopeOrg:
		runner.Labels["orgId"] = scopeID
	case tenant.ScopeProject:
		runner.Labels["projectId"] = scopeID
		delete(runner.Labels, "environmentId")
	case tenant.ScopeEnvironment:
		runner.Labels["environmentId"] = scopeID
		delete(runner.Labels, "projectId")
	}
}

func runnerInRequestScope(r *http.Request, runner domainrunner.Runner) bool {
	scopeType, scopeID := TenantScopeFilter(r)
	if scopeType == "" {
		return true
	}
	if scopeID == "" {
		return false
	}
	switch scopeType {
	case tenant.ScopeOrg:
		return runner.Labels["orgId"] == scopeID
	case tenant.ScopeProject:
		return runner.Labels["projectId"] == scopeID
	case tenant.ScopeEnvironment:
		return runner.Labels["environmentId"] == scopeID
	default:
		return false
	}
}

func filterRunnersForRequest(r *http.Request, runners []domainrunner.Runner) []domainrunner.Runner {
	if !ScopedByTenant(r) {
		return runners
	}
	filtered := make([]domainrunner.Runner, 0, len(runners))
	for _, runner := range runners {
		if runnerInRequestScope(r, runner) {
			filtered = append(filtered, runner)
		}
	}
	return filtered
}
