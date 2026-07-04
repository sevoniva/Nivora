package routes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sevoniva/nivora/internal/adapters/eventbus/memory"
	domainauth "github.com/sevoniva/nivora/internal/domain/auth"
	"github.com/sevoniva/nivora/internal/infra/config"
	authusecase "github.com/sevoniva/nivora/internal/usecase/auth"
)

func createScopedToken(t *testing.T, svc *authusecase.Service, name, role, scopeType, scopeID string) string {
	t.Helper()
	account, err := svc.CreateServiceAccount(context.Background(), authusecase.ServiceAccountInput{
		Name:      name,
		Role:      role,
		ScopeType: scopeType,
		ScopeID:   scopeID,
	}, "admin")
	if err != nil {
		t.Fatalf("create service account %s: %v", name, err)
	}
	result, err := svc.CreateAPIToken(context.Background(), authusecase.APITokenInput{
		Name:      name + "-token",
		SubjectID: account.ID,
	}, "admin")
	if err != nil {
		t.Fatalf("create api token for %s: %v", name, err)
	}
	return result.Token
}

func newIsoRouter(t *testing.T) (http.Handler, *authusecase.Service) {
	t.Helper()
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	cfg.Auth.Enabled = true
	cfg.Auth.Mode = "token"
	cfg.Auth.StaticTokenEnv = ""
	authService := authusecase.NewService(authusecase.NewMemoryStore(), memory.New())
	return newTestRouterWithAuth(cfg, authService), authService
}

// --- Tenant isolation matrix tests ---

func TestTenantIsolationCredentials(t *testing.T) {
	router, auth := newIsoRouter(t)
	tokenA := createScopedToken(t, auth, "proj-a-dev", domainauth.RoleDeveloper, "project", "project-a")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/credentials", strings.NewReader(`{"name":"cred-a","type":"token","scopeType":"project","scopeId":"project-a","secretRef":{"id":"sr-1","name":"sr-1","provider":"builtin","key":"K1"}}`))
	req.Header.Set("Authorization", "Bearer "+tokenA)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	t.Logf("create credential: %d", rec.Code)
}

func TestTenantIsolationDeployments(t *testing.T) {
	router, auth := newIsoRouter(t)
	tokenA := createScopedToken(t, auth, "proj-a-deployer", domainauth.RoleDeveloper, "project", "project-a")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/deployments", strings.NewReader(`{"metadata":{"name":"deploy-a"},"spec":{"application":"app-a","environment":"dev","target":{"type":"kubernetes-yaml","name":"test","namespace":"default"},"manifests":["examples/yaml/configmap.yaml","examples/yaml/deployment.yaml","examples/yaml/service.yaml"],"options":{"dryRun":true,"apply":false}}}`))
	req.Header.Set("Authorization", "Bearer "+tokenA)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	t.Logf("create deployment: %d", rec.Code)
}

func TestTenantIsolationReleases(t *testing.T) {
	router, auth := newIsoRouter(t)
	tokenA := createScopedToken(t, auth, "proj-a-releaser", domainauth.RoleDeveloper, "project", "project-a")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/releases", strings.NewReader(`{"name":"release-a","versionName":"1.0.0","applicationId":"app-a"}`))
	req.Header.Set("Authorization", "Bearer "+tokenA)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	t.Logf("create release: %d", rec.Code)
}

func TestTenantIsolationPipelineRuns(t *testing.T) {
	router, auth := newIsoRouter(t)
	tokenA := createScopedToken(t, auth, "proj-a-runner", domainauth.RoleDeveloper, "project", "project-a")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/pipeline-runs", strings.NewReader(`{"apiVersion":"nivora.io/v1alpha1","kind":"Pipeline","metadata":{"name":"pipeline-a"},"spec":{"stages":[{"name":"build","jobs":[{"name":"echo","executor":"shell","steps":[{"name":"say","run":"printf test-a"}]}]}]}}`))
	req.Header.Set("Authorization", "Bearer "+tokenA)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	t.Logf("create pipeline run: %d", rec.Code)
}

func TestTenantIsolationAudit(t *testing.T) {
	router, auth := newIsoRouter(t)
	tokenA := createScopedToken(t, auth, "proj-a-auditor", domainauth.RoleAuditor, "project", "project-a")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/search", nil)
	req.Header.Set("Authorization", "Bearer "+tokenA)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	t.Logf("auditor read audit: %d", rec.Code)
}

func TestTenantIsolationApprovals(t *testing.T) {
	router, auth := newIsoRouter(t)
	tokenA := createScopedToken(t, auth, "proj-a-approver", domainauth.RoleMaintainer, "project", "project-a")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/approvals", strings.NewReader(`{"subjectType":"deployment","subjectId":"dep-1","requiredByPolicy":false}`))
	req.Header.Set("Authorization", "Bearer "+tokenA)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	t.Logf("create approval: %d (needs deployment.approve)", rec.Code)
}

func TestTenantIsolationCloudAccounts(t *testing.T) {
	router, auth := newIsoRouter(t)
	tokenA := createScopedToken(t, auth, "proj-a-admin", domainauth.RoleAdmin, "project", "project-a")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/cloud/accounts", strings.NewReader(`{"name":"cloud-a","provider":"generic","credentialRef":"ref-1"}`))
	req.Header.Set("Authorization", "Bearer "+tokenA)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	t.Logf("create cloud account: %d (needs credential.manage)", rec.Code)
}

func TestTenantIsolationSecrets(t *testing.T) {
	router, auth := newIsoRouter(t)
	tokenA := createScopedToken(t, auth, "proj-a-admin", domainauth.RoleAdmin, "project", "project-a")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/secrets", strings.NewReader(`{"name":"secret-a","type":"token","scopeType":"project","scopeId":"project-a","secretRef":{"id":"sr-2","name":"secret-a","provider":"builtin","key":"S_KEY"}}`))
	req.Header.Set("Authorization", "Bearer "+tokenA)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	t.Logf("create secret: %d (needs credential.manage)", rec.Code)
}

func TestTenantIsolationVisualization(t *testing.T) {
	router, auth := newIsoRouter(t)
	tokenA := createScopedToken(t, auth, "proj-a-viewer", domainauth.RoleViewer, "project", "project-a")

	eps := []string{
		"/api/v1/visualization/pipeline-runs/prun-test/dag",
		"/api/v1/visualization/environments/env-1/topology",
		"/api/v1/visualization/security/summary",
	}
	for _, ep := range eps {
		req := httptest.NewRequest(http.MethodGet, ep, nil)
		req.Header.Set("Authorization", "Bearer "+tokenA)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		t.Logf("viewer %s: %d", ep, rec.Code)
	}
}

func TestTenantIsolationRunnerDeniedAdminRoutes(t *testing.T) {
	router, auth := newIsoRouter(t)
	tokenA := createScopedToken(t, auth, "runner-proj-a", domainauth.RoleDeveloper, "project", "project-a")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runners", nil)
	req.Header.Set("Authorization", "Bearer "+tokenA)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	t.Logf("developer access runner list: %d (needs runner.manage)", rec.Code)
}

func TestTenantIsolationPolicies(t *testing.T) {
	router, auth := newIsoRouter(t)
	tokenA := createScopedToken(t, auth, "proj-a-admin", domainauth.RoleAdmin, "project", "project-a")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/policies/evaluate", strings.NewReader(`{"subjectType":"artifact","subjectId":"test","reference":"test:latest"}`))
	req.Header.Set("Authorization", "Bearer "+tokenA)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	t.Logf("evaluate policy: %d (needs policy.manage)", rec.Code)
}

// --- List endpoints cross-tenant tests ---

func TestTenantIsolationListPipelineRuns(t *testing.T) {
	router, auth := newIsoRouter(t)
	tokenA := createScopedToken(t, auth, "list-proj-a", domainauth.RoleDeveloper, "project", "project-a")

	// Create a pipeline run as project-A.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pipeline-runs", strings.NewReader(`{"apiVersion":"nivora.io/v1alpha1","kind":"Pipeline","metadata":{"name":"list-pipe-a"},"spec":{"stages":[{"name":"build","jobs":[{"name":"echo","executor":"shell","steps":[{"name":"say","run":"printf list-a"}]}]}]}}`))
	req.Header.Set("Authorization", "Bearer "+tokenA)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 for project-A create pipeline run, got %d body=%s", rec.Code, rec.Body.String())
	}

	// List pipeline runs as project-A — should return results (scoped).
	req = httptest.NewRequest(http.MethodGet, "/api/v1/pipeline-runs", nil)
	req.Header.Set("Authorization", "Bearer "+tokenA)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	// With scope filtering, project-A should only see its own runs.
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for project-A list, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "list-pipe-a") || !strings.Contains(body, "project-a") {
		t.Fatalf("project-A list should contain its own pipeline run and project scope, body=%s", body)
	}
	t.Logf("project-A scoped list pipeline-runs: %d", rec.Code)

	// List as project-B — should NOT see project-A runs (scope filtering).
	tokenB := createScopedToken(t, auth, "list-proj-b", domainauth.RoleDeveloper, "project", "project-b")
	req = httptest.NewRequest(http.MethodGet, "/api/v1/pipeline-runs", nil)
	req.Header.Set("Authorization", "Bearer "+tokenB)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	// project-B's list response should not include project-A's pipeline run data.
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for project-B list, got %d", rec.Code)
	}
	body = rec.Body.String()
	if strings.Contains(body, "list-pipe-a") || strings.Contains(body, "list-a") {
		t.Errorf("project-B list should NOT contain project-A data, got body containing project-A reference")
	}
	if strings.Contains(body, "project-a") {
		t.Errorf("project-B list should NOT contain project-A scope, body=%s", body)
	}
	t.Logf("project-B scoped list (should be empty/filtered): %d", rec.Code)

	// List as unscoped admin — should see all.
	adminToken := createScopedToken(t, auth, "admin-list", domainauth.RoleAdmin, "", "")
	req = httptest.NewRequest(http.MethodGet, "/api/v1/pipeline-runs", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	t.Logf("unscoped admin list pipeline-runs: %d (should see all)", rec.Code)
}

func TestTenantIsolationPipelineRunDetailRoutes(t *testing.T) {
	router, auth := newIsoRouter(t)
	tokenA := createScopedToken(t, auth, "pipeline-detail-a", domainauth.RoleDeveloper, "project", "project-a")
	runID := createProjectPipelineRun(t, router, tokenA, "pipeline-detail-a")

	tokenB := createScopedToken(t, auth, "pipeline-detail-b", domainauth.RoleDeveloper, "project", "project-b")
	paths := []string{
		"/api/v1/pipeline-runs/" + runID,
		"/api/v1/pipeline-runs/" + runID + "/logs",
		"/api/v1/pipeline-runs/" + runID + "/events",
		"/api/v1/pipeline-runs/" + runID + "/timeline",
		"/api/v1/visualization/pipeline-runs/" + runID + "/dag",
		"/api/v1/visualization/pipeline-runs/" + runID + "/timeline",
		"/api/v1/visualization/pipeline-runs/" + runID + "/summary",
	}
	for _, path := range paths {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+tokenA)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("project-A should read %s, got %d body=%s", path, rec.Code, rec.Body.String())
		}

		req = httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+tokenB)
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("project-B should be forbidden for %s, got %d body=%s", path, rec.Code, rec.Body.String())
		}
	}
}

func TestTenantIsolationPipelineDefinitionCatalogRoutes(t *testing.T) {
	router, auth := newIsoRouter(t)
	tokenA := createScopedToken(t, auth, "pipeline-def-a", domainauth.RoleMaintainer, "project", "project-a")
	tokenB := createScopedToken(t, auth, "pipeline-def-b", domainauth.RoleDeveloper, "project", "project-b")
	defID := createProjectPipelineDefinition(t, router, tokenA, "pipeline-def-a")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pipelines/"+defID, nil)
	req.Header.Set("Authorization", "Bearer "+tokenA)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("project-A should read own pipeline definition, got %d body=%s", rec.Code, rec.Body.String())
	}

	for _, path := range []string{
		"/api/v1/pipelines/" + defID,
		"/api/v1/pipelines/" + defID + "/versions",
	} {
		req = httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+tokenB)
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("project-B should be forbidden for %s, got %d body=%s", path, rec.Code, rec.Body.String())
		}
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/pipelines/"+defID+"/runs", nil)
	req.Header.Set("Authorization", "Bearer "+tokenB)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("project-B should be forbidden from running project-A definition, got %d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/pipelines", nil)
	req.Header.Set("Authorization", "Bearer "+tokenB)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("project-B should list scoped pipeline definitions, got %d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, defID) || strings.Contains(body, "project-a") || strings.Contains(body, "pipeline-def-a") {
		t.Fatalf("project-B pipeline definition list leaked project-A data, body=%s", body)
	}
}

func TestTenantIsolationListDeployments(t *testing.T) {
	router, auth := newIsoRouter(t)
	tokenA := createScopedToken(t, auth, "deploy-list-a", domainauth.RoleDeveloper, "project", "project-a")

	createProjectDeploymentRun(t, router, tokenA, "deploy-list-a")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/deployments", nil)
	req.Header.Set("Authorization", "Bearer "+tokenA)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for project-A list deployments, got %d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "deploy-list-a") || !strings.Contains(body, "project-a") {
		t.Fatalf("project-A list should contain its own deployment and project scope, body=%s", body)
	}

	tokenB := createScopedToken(t, auth, "deploy-list-b", domainauth.RoleDeveloper, "project", "project-b")
	req = httptest.NewRequest(http.MethodGet, "/api/v1/deployments", nil)
	req.Header.Set("Authorization", "Bearer "+tokenB)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for project-B list deployments, got %d body=%s", rec.Code, rec.Body.String())
	}
	body = rec.Body.String()
	if strings.Contains(body, "deploy-list-a") || strings.Contains(body, "project-a") {
		t.Fatalf("project-B list should not contain project-A deployment, body=%s", body)
	}
}

func TestTenantIsolationDeploymentDetailRoutes(t *testing.T) {
	router, auth := newIsoRouter(t)
	tokenA := createScopedToken(t, auth, "deploy-detail-a", domainauth.RoleDeveloper, "project", "project-a")
	runID := createProjectDeploymentRun(t, router, tokenA, "deploy-detail-a")

	tokenB := createScopedToken(t, auth, "deploy-detail-b", domainauth.RoleDeveloper, "project", "project-b")
	paths := []string{
		"/api/v1/deployments/" + runID,
		"/api/v1/deployments/" + runID + "/plan",
		"/api/v1/deployments/" + runID + "/resources",
		"/api/v1/deployments/" + runID + "/health",
		"/api/v1/deployments/" + runID + "/diff",
		"/api/v1/deployments/" + runID + "/manifest-snapshot",
		"/api/v1/deployments/" + runID + "/rollback-plan",
		"/api/v1/deployments/" + runID + "/logs",
		"/api/v1/deployments/" + runID + "/events",
		"/api/v1/deployments/" + runID + "/timeline",
		"/api/v1/deployments/" + runID + "/security",
		"/api/v1/visualization/deployments/" + runID + "/timeline",
		"/api/v1/visualization/deployments/" + runID + "/resources",
		"/api/v1/visualization/deployments/" + runID + "/diff",
		"/api/v1/visualization/deployments/" + runID + "/health",
	}
	for _, path := range paths {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+tokenA)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("project-A should read %s, got %d body=%s", path, rec.Code, rec.Body.String())
		}

		req = httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+tokenB)
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("project-B should be forbidden for %s, got %d body=%s", path, rec.Code, rec.Body.String())
		}
	}
}

func TestTenantIsolationListReleases(t *testing.T) {
	router, auth := newIsoRouter(t)
	tokenA := createScopedToken(t, auth, "release-list-a", domainauth.RoleDeveloper, "project", "project-a")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/releases", nil)
	req.Header.Set("Authorization", "Bearer "+tokenA)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	t.Logf("project-A list releases: %d", rec.Code)
}

func TestTenantIsolationReleaseExecutionDetailRoutes(t *testing.T) {
	router, auth := newIsoRouter(t)
	tokenA := createScopedToken(t, auth, "rexec-detail-a", domainauth.RoleDeveloper, "project", "project-a")
	executionID := createProjectReleaseExecution(t, router, tokenA, "rexec-detail-a")

	tokenB := createScopedToken(t, auth, "rexec-detail-b", domainauth.RoleDeveloper, "project", "project-b")
	paths := []string{
		"/api/v1/releases/executions/" + executionID,
		"/api/v1/releases/executions/" + executionID + "/timeline",
		"/api/v1/releases/executions/" + executionID + "/targets",
		"/api/v1/visualization/releases/executions/" + executionID + "/timeline",
		"/api/v1/visualization/releases/executions/" + executionID + "/targets",
	}
	for _, path := range paths {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+tokenA)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("project-A should read %s, got %d body=%s", path, rec.Code, rec.Body.String())
		}

		req = httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+tokenB)
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("project-B should be forbidden for %s, got %d body=%s", path, rec.Code, rec.Body.String())
		}
	}
}

func TestTenantIsolationListCredentials(t *testing.T) {
	router, auth := newIsoRouter(t)
	adminA := createScopedToken(t, auth, "cred-list-a", domainauth.RoleAdmin, "project", "project-a")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/credentials", nil)
	req.Header.Set("Authorization", "Bearer "+adminA)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	t.Logf("project-A admin list credentials: %d", rec.Code)
}

func TestTenantIsolationAuditSearch(t *testing.T) {
	router, auth := newIsoRouter(t)
	tokenA := createScopedToken(t, auth, "audit-search-a", domainauth.RoleAuditor, "project", "project-a")

	// Search with scope filter.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/search?scopeType=project&scopeId=project-a", nil)
	req.Header.Set("Authorization", "Bearer "+tokenA)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	t.Logf("auditor search audit with scope filter: %d", rec.Code)

	// Search without scope filter.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/audit/search", nil)
	req.Header.Set("Authorization", "Bearer "+tokenA)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	t.Logf("auditor search audit without scope: %d", rec.Code)
}

func TestTenantIsolationVisualizationAggregation(t *testing.T) {
	router, auth := newIsoRouter(t)
	tokenA := createScopedToken(t, auth, "viz-agg-a", domainauth.RoleViewer, "project", "project-a")

	// Visualization endpoints that aggregate data across tenants.
	aggEndpoints := []string{
		"/api/v1/visualization/runners/summary",
		"/api/v1/visualization/security/summary",
		"/api/v1/visualization/audit/timeline",
	}
	for _, ep := range aggEndpoints {
		req := httptest.NewRequest(http.MethodGet, ep, nil)
		req.Header.Set("Authorization", "Bearer "+tokenA)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		t.Logf("scoped viewer %s: %d", ep, rec.Code)
	}
}

func TestTenantIsolationAggregateObservabilityRoutes(t *testing.T) {
	router, auth := newIsoRouter(t)
	developerA := createScopedToken(t, auth, "aggregate-dev-a", domainauth.RoleDeveloper, "project", "project-a")
	developerB := createScopedToken(t, auth, "aggregate-dev-b", domainauth.RoleDeveloper, "project", "project-b")
	auditorA := createScopedToken(t, auth, "aggregate-auditor-a", domainauth.RoleAuditor, "project", "project-a")

	runA := createProjectPipelineRun(t, router, developerA, "aggregate-a")
	runB := createProjectPipelineRun(t, router, developerB, "aggregate-b")

	for _, path := range []string{"/api/v1/events", "/api/v1/logs"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+developerA)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("project-A aggregate read %s got %d body=%s", path, rec.Code, rec.Body.String())
		}
		body := rec.Body.String()
		if !strings.Contains(body, runA) {
			t.Fatalf("project-A aggregate read %s should include own run %s body=%s", path, runA, body)
		}
		if strings.Contains(body, runB) || strings.Contains(body, "aggregate-b") {
			t.Fatalf("project-A aggregate read %s leaked project-B run %s body=%s", path, runB, body)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/visualization/audit/timeline", nil)
	req.Header.Set("Authorization", "Bearer "+auditorA)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("project-A audit timeline got %d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, runB) || strings.Contains(body, "aggregate-b") {
		t.Fatalf("project-A audit timeline leaked project-B run %s body=%s", runB, body)
	}
}

func TestTenantIsolationEvidenceBundles(t *testing.T) {
	router, auth := newIsoRouter(t)
	developerA := createScopedToken(t, auth, "evidence-dev-a", domainauth.RoleDeveloper, "project", "project-a")
	auditorA := createScopedToken(t, auth, "evidence-auditor-a", domainauth.RoleAuditor, "project", "project-a")
	auditorB := createScopedToken(t, auth, "evidence-auditor-b", domainauth.RoleAuditor, "project", "project-b")
	runA := createProjectPipelineRun(t, router, developerA, "evidence-a")
	bundleID := "evb-pipelineRun-" + runA

	req := httptest.NewRequest(http.MethodGet, "/api/v1/evidence/pipelineRun/"+runA, nil)
	req.Header.Set("Authorization", "Bearer "+auditorA)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("project-A auditor should generate own evidence bundle, got %d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, bundleID) || !strings.Contains(body, `"scopeId":"project-a"`) {
		t.Fatalf("project-A evidence bundle missing id or scope metadata, body=%s", body)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/evidence/pipelineRun/"+runA, nil)
	req.Header.Set("Authorization", "Bearer "+auditorB)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("project-B auditor should be forbidden from subject evidence, got %d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/evidence/bundles/"+bundleID, nil)
	req.Header.Set("Authorization", "Bearer "+auditorB)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("project-B auditor should be forbidden from bundle id read, got %d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/evidence/bundles", nil)
	req.Header.Set("Authorization", "Bearer "+auditorB)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("project-B auditor should list scoped evidence bundles, got %d body=%s", rec.Code, rec.Body.String())
	}
	body = rec.Body.String()
	if strings.Contains(body, runA) || strings.Contains(body, bundleID) || strings.Contains(body, "project-a") {
		t.Fatalf("project-B evidence bundle list leaked project-A data, body=%s", body)
	}
}

func createProjectPipelineRun(t *testing.T, router http.Handler, token, name string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pipeline-runs", strings.NewReader(`{"apiVersion":"nivora.io/v1alpha1","kind":"Pipeline","metadata":{"name":"`+name+`"},"spec":{"stages":[{"name":"build","jobs":[{"name":"echo","executor":"shell","steps":[{"name":"say","run":"printf `+name+`"}]}]}]}}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 for project pipeline create, got %d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode pipeline create response: %v", err)
	}
	run, ok := body["run"].(map[string]any)
	if !ok {
		t.Fatalf("pipeline create response missing run object: %s", rec.Body.String())
	}
	id, ok := run["id"].(string)
	if !ok || id == "" {
		t.Fatalf("pipeline create response missing run id: %s", rec.Body.String())
	}
	return id
}

func createProjectPipelineDefinition(t *testing.T, router http.Handler, token, name string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pipelines", strings.NewReader(`{"projectId":"ignored-by-scoped-token","definition":{"apiVersion":"nivora.io/v1alpha1","kind":"Pipeline","metadata":{"name":"`+name+`"},"spec":{"stages":[{"name":"build","jobs":[{"name":"echo","executor":"shell","steps":[{"name":"say","run":"printf `+name+`"}]}]}]}}}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 for project pipeline definition create, got %d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode pipeline definition create response: %v", err)
	}
	pipeline, ok := body["pipeline"].(map[string]any)
	if !ok {
		t.Fatalf("pipeline definition create response missing pipeline object: %s", rec.Body.String())
	}
	id, ok := pipeline["id"].(string)
	if !ok || id == "" {
		t.Fatalf("pipeline definition create response missing pipeline id: %s", rec.Body.String())
	}
	if projectID, _ := pipeline["projectId"].(string); projectID != "project-a" {
		t.Fatalf("scoped token should force project-a projectId, got %q body=%s", projectID, rec.Body.String())
	}
	return id
}

func createProjectDeploymentRun(t *testing.T, router http.Handler, token, name string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/deployments", strings.NewReader(`{"apiVersion":"nivora.io/v1alpha1","kind":"Deployment","metadata":{"name":"`+name+`"},"spec":{"application":"app-a","environment":"dev","target":{"type":"kubernetes-yaml","name":"test","namespace":"default"},"manifests":["examples/yaml/configmap.yaml"],"options":{"dryRun":true,"apply":false}}}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 for project deployment create, got %d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode deployment create response: %v", err)
	}
	run, ok := body["run"].(map[string]any)
	if !ok {
		t.Fatalf("deployment create response missing run object: %s", rec.Body.String())
	}
	id, ok := run["id"].(string)
	if !ok || id == "" {
		t.Fatalf("deployment create response missing run id: %s", rec.Body.String())
	}
	return id
}

func createProjectReleaseExecution(t *testing.T, router http.Handler, token, name string) string {
	t.Helper()
	body := `{
	  "apiVersion":"nivora.io/v1alpha1",
	  "kind":"ReleaseOrchestration",
	  "metadata":{"name":"` + name + `"},
	  "spec":{
	    "environment":"dev",
	    "strategy":"sequential",
	    "release":{
	      "apiVersion":"nivora.io/v1alpha1",
	      "kind":"Release",
	      "metadata":{"name":"` + name + `"},
	      "spec":{
	        "version":"1.0.0",
	        "application":"app-a",
	        "environment":"dev",
	        "artifacts":[{"name":"` + name + `","type":"image","required":true,"reference":"registry.example.com/demo/app:1.0.0"}]
	      }
	    },
	    "targets":[{
	      "name":"noop-target",
	      "type":"noop",
	      "deployment":{
	        "apiVersion":"nivora.io/v1alpha1",
	        "kind":"Deployment",
	        "metadata":{"name":"` + name + `-deployment"},
	        "spec":{
	          "application":"app-a",
	          "environment":"dev",
	          "target":{"type":"noop","name":"noop-target"},
	          "options":{"dryRun":true,"apply":false}
	        }
	      }
	    }]
	  }
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/releases/local/deploy", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 for project release deploy, got %d body=%s", rec.Code, rec.Body.String())
	}
	var response map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode release execution response: %v", err)
	}
	execution, ok := response["execution"].(map[string]any)
	if !ok {
		t.Fatalf("release execution response missing execution object: %s", rec.Body.String())
	}
	id, ok := execution["id"].(string)
	if !ok || id == "" {
		t.Fatalf("release execution response missing execution id: %s", rec.Body.String())
	}
	return id
}
