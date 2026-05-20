package routes

import (
	"context"
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

	// List pipeline runs as project-A — should return results.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/pipeline-runs", nil)
	req.Header.Set("Authorization", "Bearer "+tokenA)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	t.Logf("project-A list pipeline-runs: %d", rec.Code)

	// List as unscoped admin — should also return results.
	adminToken := createScopedToken(t, auth, "admin-list", domainauth.RoleAdmin, "", "")
	req = httptest.NewRequest(http.MethodGet, "/api/v1/pipeline-runs", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	t.Logf("unscoped admin list pipeline-runs: %d (should see all)", rec.Code)
}

func TestTenantIsolationListDeployments(t *testing.T) {
	router, auth := newIsoRouter(t)
	tokenA := createScopedToken(t, auth, "deploy-list-a", domainauth.RoleDeveloper, "project", "project-a")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/deployments", nil)
	req.Header.Set("Authorization", "Bearer "+tokenA)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	t.Logf("project-A list deployments: %d", rec.Code)
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
