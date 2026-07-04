package routes

import (
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

func TestCatalogRoutesCreateListUpdateAndDisable(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	router := newTestRouter(cfg)

	org := postCatalogResource(t, router, "/api/v1/orgs", `{"name":"Platform","labels":{"tier":"root"}}`, http.StatusCreated)
	orgID := stringField(t, org, "id")
	if !boolField(t, org, "enabled") {
		t.Fatalf("created org should be enabled: %+v", org)
	}

	project := postCatalogResource(t, router, "/api/v1/projects", `{"orgId":"`+orgID+`","name":"Delivery"}`, http.StatusCreated)
	projectID := stringField(t, project, "id")

	application := postCatalogResource(t, router, "/api/v1/applications", `{"projectId":"`+projectID+`","name":"Control Plane"}`, http.StatusCreated)
	if stringField(t, application, "projectId") != projectID {
		t.Fatalf("application projectId mismatch: %+v", application)
	}

	environment := postCatalogResource(t, router, "/api/v1/environments", `{"projectId":"`+projectID+`","name":"Production"}`, http.StatusCreated)
	environmentID := stringField(t, environment, "id")

	target := postCatalogResource(t, router, "/api/v1/release-targets", `{"environmentId":"`+environmentID+`","name":"prod-noop","targetType":"noop","credentialRef":"target-cred-ref"}`, http.StatusCreated)
	targetID := stringField(t, target, "id")
	if stringField(t, target, "projectId") != projectID {
		t.Fatalf("target projectId mismatch: %+v", target)
	}
	if boolField(t, target, "allowApply") || boolField(t, target, "allowSync") || boolField(t, target, "allowRemoteHostDeploy") {
		t.Fatalf("unsafe target flags should default false: %+v", target)
	}
	if stringField(t, target, "credentialRef") != "target-cred-ref" {
		t.Fatalf("target credentialRef mismatch: %+v", target)
	}

	repository := postCatalogResource(t, router, "/api/v1/repositories", `{"projectId":"`+projectID+`","name":"Service Repo","url":"https://example.com/team/service.git","provider":"generic","credentialRef":"cred-ref"}`, http.StatusCreated)
	repositoryID := stringField(t, repository, "id")
	if stringField(t, repository, "credentialRef") != "cred-ref" {
		t.Fatalf("repository credentialRef mismatch: %+v", repository)
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/environments/"+environmentID, strings.NewReader(`{"description":"release gate target"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("update environment status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "release gate target") {
		t.Fatalf("update environment body = %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/v1/environments/"+environmentID, nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("disable environment status = %d body = %s", rec.Code, rec.Body.String())
	}
	var disabled map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &disabled); err != nil {
		t.Fatalf("decode disabled environment: %v", err)
	}
	if boolField(t, disabled, "enabled") {
		t.Fatalf("disabled environment still enabled: %+v", disabled)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/projects?orgId="+orgID, nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), projectID) {
		t.Fatalf("list projects status = %d body = %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/release-targets?projectId="+projectID, nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), targetID) {
		t.Fatalf("list targets status = %d body = %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/release-targets/"+targetID+"/validate", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"valid":true`) {
		t.Fatalf("validate target status = %d body = %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/v1/release-targets/"+targetID, nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"enabled":false`) {
		t.Fatalf("disable target status = %d body = %s", rec.Code, rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodPost, "/api/v1/release-targets/"+targetID+"/validate", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "target is disabled") {
		t.Fatalf("validate disabled target status = %d body = %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/v1/repositories/"+repositoryID, nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("disable repository status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"enabled":false`) {
		t.Fatalf("disable repository body = %s", rec.Body.String())
	}
}

func TestCatalogRoutesValidateParentsAndDuplicates(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	router := newTestRouter(cfg)

	postCatalogResource(t, router, "/api/v1/projects", `{"orgId":"missing","name":"Delivery"}`, http.StatusNotFound)
	postCatalogResource(t, router, "/api/v1/repositories", `{"projectId":"missing","name":"Repo","url":"https://example.com/team/service.git"}`, http.StatusNotFound)
	postCatalogResource(t, router, "/api/v1/release-targets", `{"environmentId":"missing","name":"target","targetType":"noop"}`, http.StatusNotFound)
	postCatalogResource(t, router, "/api/v1/orgs", `{"name":"Platform"}`, http.StatusCreated)
	postCatalogResource(t, router, "/api/v1/orgs", `{"name":"platform"}`, http.StatusConflict)
}

func TestCatalogRoutesArePermissionProtected(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	cfg.Auth.Enabled = true
	cfg.Auth.Mode = "token"
	authService := authusecase.NewService(authusecase.NewMemoryStore(), memory.New())
	router := newTestRouterWithAuth(cfg, authService)
	viewerToken := createServiceAccountAndToken(t, authService, "catalog-viewer", domainauth.RoleViewer, "", "")
	adminToken := createServiceAccountAndToken(t, authService, "catalog-admin", domainauth.RoleAdmin, "", "")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/orgs", strings.NewReader(`{"name":"Platform"}`))
	req.Header.Set("Authorization", "Bearer "+viewerToken)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("viewer create org status = %d body = %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/orgs", strings.NewReader(`{"name":"Platform"}`))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("admin create org status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func postCatalogResource(t *testing.T, router http.Handler, path string, body string, want int) map[string]any {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != want {
		t.Fatalf("%s status = %d want %d body = %s", path, rec.Code, want, rec.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode %s response: %v", path, err)
	}
	return out
}

func stringField(t *testing.T, object map[string]any, name string) string {
	t.Helper()
	value, ok := object[name].(string)
	if !ok || value == "" {
		t.Fatalf("field %s missing from %+v", name, object)
	}
	return value
}

func boolField(t *testing.T, object map[string]any, name string) bool {
	t.Helper()
	value, ok := object[name].(bool)
	if !ok {
		t.Fatalf("field %s missing from %+v", name, object)
	}
	return value
}
