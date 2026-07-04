package routes

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	ociartifact "github.com/sevoniva/nivora/internal/adapters/artifact/oci"
	"github.com/sevoniva/nivora/internal/adapters/cloud/aws"
	"github.com/sevoniva/nivora/internal/adapters/eventbus/memory"
	argocdadapter "github.com/sevoniva/nivora/internal/adapters/executor/argocd"
	shellexecutor "github.com/sevoniva/nivora/internal/adapters/executor/shell"
	yamlapply "github.com/sevoniva/nivora/internal/adapters/executor/yaml_apply"
	noopnotification "github.com/sevoniva/nivora/internal/adapters/notification/noop"
	builtinsecret "github.com/sevoniva/nivora/internal/adapters/secret/builtin"
	"github.com/sevoniva/nivora/internal/api/http/handlers"
	domainauth "github.com/sevoniva/nivora/internal/domain/auth"
	domainsecurity "github.com/sevoniva/nivora/internal/domain/security"
	"github.com/sevoniva/nivora/internal/infra/config"
	portcloud "github.com/sevoniva/nivora/internal/ports/cloud"
	"github.com/sevoniva/nivora/internal/ports/policy"
	portsecurity "github.com/sevoniva/nivora/internal/ports/security"
	approvalusecase "github.com/sevoniva/nivora/internal/usecase/approval"
	artifactusecase "github.com/sevoniva/nivora/internal/usecase/artifact"
	authusecase "github.com/sevoniva/nivora/internal/usecase/auth"
	cloudusecase "github.com/sevoniva/nivora/internal/usecase/cloud"
	complianceusecase "github.com/sevoniva/nivora/internal/usecase/compliance"
	credentialusecase "github.com/sevoniva/nivora/internal/usecase/credential"
	deploymentusecase "github.com/sevoniva/nivora/internal/usecase/deployment"
	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
	pluginusecase "github.com/sevoniva/nivora/internal/usecase/plugin"
	releaseorchestration "github.com/sevoniva/nivora/internal/usecase/releaseorchestration"
	securityusecase "github.com/sevoniva/nivora/internal/usecase/security"
	tenancyusecase "github.com/sevoniva/nivora/internal/usecase/tenancy"
	"github.com/sevoniva/nivora/internal/version"
)

func TestHealthRoutes(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	router := newTestRouter(cfg)

	tests := []string{"/healthz", "/readyz", "/api/v1/version"}
	for _, path := range tests {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("%s status = %d", path, rec.Code)
		}
		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("%s response is not json: %v", path, err)
		}
	}
}

func TestPlaceholderRoute(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	router := newTestRouter(cfg)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d", rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not json: %v", err)
	}
	if body["code"] != "not_implemented" {
		t.Fatalf("code = %v", body["code"])
	}
	if body["path"] != "/api/v1/integrations" {
		t.Fatalf("path = %v", body["path"])
	}
}

func TestRequestBodyLimit(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	router := newTestRouter(cfg)
	body := strings.NewReader(strings.Repeat("x", handlers.MaxRequestBodyBytes+1))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pipeline-runs", body)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func TestArtifactRoutes(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	router := newTestRouter(cfg)
	for _, tc := range []struct {
		path string
		body string
	}{
		{"/api/v1/artifacts/inspect", `{"reference":"registry.example.com/team/app:1.0.0","type":"image"}`},
		{"/api/v1/artifacts/resolve", `{"reference":"registry.example.com/team/app@sha256:abcdef","type":"image"}`},
		{"/api/v1/artifact-registries/validate", `{"name":"local-oci","type":"oci","endpoint":"registry.example.com","insecure":false}`},
	} {
		req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status = %d body = %s", tc.path, rec.Code, rec.Body.String())
		}
	}
}

func TestCredentialRoutesDoNotReturnSecretValue(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	router := newTestRouter(cfg)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/secrets", strings.NewReader(`{"name":"registry-token","key":"examples/registry/token","value":"sample-value-for-test-only"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("create secret status = %d body = %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "sample-value-for-test-only") {
		t.Fatalf("secret create response leaked secret value")
	}
	var createdSecret struct {
		ID      string `json:"id"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &createdSecret); err != nil {
		t.Fatalf("decode secret ref: %v", err)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/secrets/refs", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list secret refs status = %d body = %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "sample-value-for-test-only") {
		t.Fatalf("secret refs response leaked secret value")
	}
	req = httptest.NewRequest(http.MethodPost, "/api/v1/secrets/"+createdSecret.ID+"/rotate", strings.NewReader(`{"value":"placeholder-rotated-value"}`))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("rotate secret status = %d body = %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "placeholder-rotated-value") {
		t.Fatalf("secret rotate response leaked secret value")
	}
	req = httptest.NewRequest(http.MethodPost, "/api/v1/secrets/provider/validate", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("validate provider status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"provider":"builtin"`) {
		t.Fatalf("provider validation body = %s", rec.Body.String())
	}
}

func TestTenancyQuotaRoutes(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	router := newTestRouter(cfg)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tenancy/quota?scopeType=project&scopeId=project-a", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("quota status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"maxConcurrentPipelineRuns"`) {
		t.Fatalf("quota body missing limits: %s", rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/tenancy/usage?scopeType=project&scopeId=project-a", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("usage status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func TestComplianceEvidenceRoutes(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	router := newTestRouter(cfg)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pipeline-runs", strings.NewReader(`{"apiVersion":"nivora.io/v1alpha1","kind":"Pipeline","metadata":{"name":"evidence"},"spec":{"stages":[{"name":"build","jobs":[{"name":"job","executor":"shell","steps":[{"name":"step","run":"printf ok"}]}]}]}}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create pipeline status = %d body = %s", rec.Code, rec.Body.String())
	}
	var created struct {
		Run struct {
			ID string `json:"id"`
		} `json:"run"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode pipeline: %v", err)
	}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/evidence/pipelineRun/"+created.Run.ID, nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("evidence status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"logReferences"`) {
		t.Fatalf("evidence missing log refs: %s", rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodPost, "/api/v1/evidence/bundles", strings.NewReader(`{"subjectType":"pipelineRun","subjectId":"`+created.Run.ID+`"}`))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("generate evidence bundle status = %d body = %s", rec.Code, rec.Body.String())
	}
	var bundle struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &bundle); err != nil {
		t.Fatalf("decode evidence bundle: %v", err)
	}
	if bundle.ID == "" {
		t.Fatalf("generated evidence bundle missing id: %s", rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/evidence/bundles/"+bundle.ID, nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get evidence bundle status = %d body = %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(strings.ToLower(rec.Body.String()), "password") || strings.Contains(strings.ToLower(rec.Body.String()), "authorization") {
		t.Fatalf("evidence bundle leaked secret-like field: %s", rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/evidence/bundles/"+bundle.ID+"/export?format=markdown", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("export evidence bundle status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "# Evidence Bundle") {
		t.Fatalf("markdown evidence bundle missing heading: %s", rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/audit/search?subject="+created.Run.ID, nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("audit search status = %d body = %s", rec.Code, rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/audit/search?subject="+created.Run.ID+"&limit=1", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("paginated audit search status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"pagination"`) || !strings.Contains(rec.Body.String(), `"limit":1`) {
		t.Fatalf("paginated audit search body = %s", rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodPost, "/api/v1/retention-policy", strings.NewReader(`{"scopeType":"project","scopeId":"project-a","logDays":14,"auditDays":730}`))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("retention status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func TestAuthWhoamiDevMode(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	router := newTestRouter(cfg)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/whoami", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("whoami status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "local-admin") {
		t.Fatalf("expected local-admin subject, body = %s", rec.Body.String())
	}
}

func TestAuthTokenModeRequiresBearerToken(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	t.Setenv("NIVORA_TEST_AUTH_TOKEN", "test-token")
	cfg.Auth.Enabled = true
	cfg.Auth.Mode = "token"
	cfg.Auth.StaticTokenEnv = "NIVORA_TEST_AUTH_TOKEN"
	router := newTestRouter(cfg)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/whoami", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized, got %d body = %s", rec.Code, rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/auth/whoami", nil)
	req.Header.Set("Authorization", "Bearer "+os.Getenv("NIVORA_TEST_AUTH_TOKEN"))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected authorized, got %d body = %s", rec.Code, rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/auth/token-info", nil)
	req.Header.Set("Authorization", "Bearer "+os.Getenv("NIVORA_TEST_AUTH_TOKEN"))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected token-info, got %d body = %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), os.Getenv("NIVORA_TEST_AUTH_TOKEN")) {
		t.Fatalf("token-info response leaked token value")
	}
}

func TestScopedAPITokenCannotReadAnotherProject(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	cfg.Auth.Enabled = true
	cfg.Auth.Mode = "token"
	authService := authusecase.NewService(authusecase.NewMemoryStore(), memory.New())
	account, err := authService.CreateServiceAccount(context.Background(), authusecase.ServiceAccountInput{Name: "project-a-ci", Role: domainauth.RoleDeveloper, ScopeType: "project", ScopeID: "project-a"}, "admin")
	if err != nil {
		t.Fatalf("create service account: %v", err)
	}
	token, err := authService.CreateAPIToken(context.Background(), authusecase.APITokenInput{Name: "project-a-token", SubjectID: account.ID}, "admin")
	if err != nil {
		t.Fatalf("create api token: %v", err)
	}
	router := newTestRouterWithAuth(cfg, authService)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/project-b/members", nil)
	req.Header.Set("Authorization", "Bearer "+token.Token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected cross-project denial, got %d body = %s", rec.Code, rec.Body.String())
	}
}

func TestServiceAccountAndAPITokenRoutesDoNotLeakHashes(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	router := newTestRouter(cfg)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/service-accounts", strings.NewReader(`{"name":"ci","role":"developer","scopeType":"project","scopeId":"project-1"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("create service account status = %d body = %s", rec.Code, rec.Body.String())
	}
	var account struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &account); err != nil {
		t.Fatalf("decode service account: %v", err)
	}
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/tokens", strings.NewReader(`{"name":"ci-token","subjectId":"`+account.ID+`"}`))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create api token status = %d body = %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "tokenHash") || strings.Contains(rec.Body.String(), "sha256:") {
		t.Fatalf("token response leaked hash = %s", rec.Body.String())
	}
	var token struct {
		Metadata struct {
			ID string `json:"id"`
		} `json:"metadata"`
		Token string `json:"token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &token); err != nil {
		t.Fatalf("decode api token: %v", err)
	}
	if token.Token == "" || token.Metadata.ID == "" {
		t.Fatalf("expected one-time token and metadata id: %s", rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/auth/tokens", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list api tokens status = %d body = %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), token.Token) || strings.Contains(rec.Body.String(), "tokenHash") {
		t.Fatalf("list api tokens leaked token material = %s", rec.Body.String())
	}
}

func TestCriticalRoutesRequirePermissionInOIDCMode(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	cfg.Auth.Enabled = true
	cfg.Auth.Mode = "oidc"
	cfg.Auth.OIDC.Issuer = "https://issuer.example"
	cfg.Auth.OIDC.ClientID = "nivora"
	authService := authusecase.NewService(authusecase.NewMemoryStore(), memory.New())
	authService.SetOIDCProvider(routeOIDCProvider{})
	router := newTestRouterWithAuth(cfg, authService)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/deployments", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer viewer-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden for viewer deployment create, got %d body = %s", rec.Code, rec.Body.String())
	}
}

func TestRouteRBACAllowsSufficientPermission(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	cfg.Auth.Enabled = true
	cfg.Auth.Mode = "oidc"
	cfg.Auth.OIDC.Issuer = "https://issuer.example"
	cfg.Auth.OIDC.ClientID = "nivora"
	authService := authusecase.NewService(authusecase.NewMemoryStore(), memory.New())
	authService.SetOIDCProvider(securityOIDCProvider{})
	router := newTestRouterWithAuth(cfg, authService)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/deployments/plan", strings.NewReader(`{"apiVersion":"nivora.io/v1alpha1","kind":"Deployment","metadata":{"name":"rbac-plan"},"spec":{"application":"demo","environment":"dev","target":{"type":"kubernetes-yaml","name":"dev","namespace":"default"},"manifests":["../../../../examples/yaml/deployment.yaml"],"options":{"dryRun":true,"apply":false}}}`))
	req.Header.Set("Authorization", "Bearer developer-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("developer deployment plan status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func TestAuditorCanReadAuditButCannotMutate(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	cfg.Auth.Enabled = true
	cfg.Auth.Mode = "oidc"
	cfg.Auth.OIDC.Issuer = "https://issuer.example"
	cfg.Auth.OIDC.ClientID = "nivora"
	authService := authusecase.NewService(authusecase.NewMemoryStore(), memory.New())
	authService.SetOIDCProvider(securityOIDCProvider{})
	router := newTestRouterWithAuth(cfg, authService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/search", nil)
	req.Header.Set("Authorization", "Bearer auditor-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("auditor audit read status = %d body = %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/deployments", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer auditor-token")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("auditor mutate status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func TestRunnerTokenScopeInTokenAuthMode(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	t.Setenv("NIVORA_TEST_AUTH_TOKEN", "admin-token")
	cfg.Auth.Enabled = true
	cfg.Auth.Mode = "token"
	cfg.Auth.StaticTokenEnv = "NIVORA_TEST_AUTH_TOKEN"
	router := newTestRouter(cfg)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/runners/register", strings.NewReader(`{"id":"scoped-runner","name":"scoped-runner","status":"online","executors":["shell"]}`))
	req.Header.Set("Authorization", "Bearer admin-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("runner register status = %d body = %s", rec.Code, rec.Body.String())
	}
	var registered struct {
		Token struct {
			Token string `json:"token"`
		} `json:"token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &registered); err != nil {
		t.Fatalf("decode runner token: %v", err)
	}
	if registered.Token.Token == "" {
		t.Fatalf("expected one-time runner token body = %s", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "tokenHash") {
		t.Fatalf("runner token response leaked hash = %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/runners/scoped-runner/heartbeat", nil)
	req.Header.Set("X-Nivora-Runner-Token", registered.Token.Token)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("runner heartbeat with runner token status = %d body = %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/runners", nil)
	req.Header.Set("Authorization", "Bearer "+registered.Token.Token)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("runner bearer token reached admin route status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func TestRunnerTokenCannotMutateUnrelatedJob(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	t.Setenv("NIVORA_TEST_AUTH_TOKEN", "admin-token")
	cfg.Auth.Enabled = true
	cfg.Auth.Mode = "token"
	cfg.Auth.StaticTokenEnv = "NIVORA_TEST_AUTH_TOKEN"
	pipelineService := newTestPipelineService()
	created, err := pipelineService.CreateQueued(context.Background(), pipelineusecase.CreateRunInput{
		Definition: pipelineusecase.Definition{
			APIVersion: "nivora.io/v1alpha1",
			Kind:       "Pipeline",
			Metadata:   pipelineusecase.Metadata{Name: "runner-boundary"},
			Spec: pipelineusecase.Spec{Stages: []pipelineusecase.Stage{{
				Name: "build",
				Jobs: []pipelineusecase.Job{{
					Name:     "job",
					Executor: "shell",
					Steps:    []pipelineusecase.Step{{Name: "step", Run: "printf ok"}},
				}},
			}}},
		},
	})
	if err != nil {
		t.Fatalf("create queued run: %v", err)
	}
	router := newTestRouterWithPipelineAndAuth(cfg, pipelineService, authusecase.NewService(authusecase.NewMemoryStore(), memory.New()))

	tokenA := registerRunnerAndToken(t, router, "runner-a")
	tokenB := registerRunnerAndToken(t, router, "runner-b")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/runners/runner-a/jobs/claim", nil)
	req.Header.Set("X-Nivora-Runner-Token", tokenA)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("runner claim status = %d body = %s", rec.Code, rec.Body.String())
	}
	var claim struct {
		JobRunID string `json:"jobRunId"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &claim); err != nil {
		t.Fatalf("decode claim: %v", err)
	}
	if claim.JobRunID == "" {
		t.Fatalf("claim missing jobRunId: %s", rec.Body.String())
	}

	body := `{"pipelineRunId":"` + created.Record.Run.ID + `","stream":"stdout","content":"nope"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/runners/runner-b/jobs/"+claim.JobRunID+"/logs", strings.NewReader(body))
	req.Header.Set("X-Nivora-Runner-Token", tokenB)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("runner-b appended unrelated job log status = %d body = %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/runners/runner-b/jobs/"+claim.JobRunID+"/status", strings.NewReader(`{"status":"Succeeded"}`))
	req.Header.Set("X-Nivora-Runner-Token", tokenB)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("runner-b updated unrelated job status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func TestCredentialRoutesDoNotReturnCredentialValues(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	router := newTestRouter(cfg)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/credentials", strings.NewReader(`{"name":"registry","type":"registry","scopeType":"project","scopeId":"demo","secretRef":{"id":"sec-placeholder","name":"placeholder","provider":"builtin","key":"registry/token"}}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("create credential status = %d body = %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "password") || strings.Contains(rec.Body.String(), "tokenValue") {
		t.Fatalf("credential response leaked value-like field = %s", rec.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode credential: %v", err)
	}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/credentials/"+created.ID+"?scopeType=project&scopeId=demo", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get credential status = %d body = %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "password") || strings.Contains(rec.Body.String(), "tokenValue") || strings.Contains(rec.Body.String(), "secretValue") {
		t.Fatalf("credential GET leaked secret material = %s", rec.Body.String())
	}
}

func TestApprovalRoutes(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	router := newTestRouter(cfg)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/approvals", strings.NewReader(`{"subjectType":"deployment","subjectId":"drun-test","environmentId":"prod","requestedBy":"tester","reason":"deployment approval"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("create approval status = %d body = %s", rec.Code, rec.Body.String())
	}
	var created struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode approval: %v", err)
	}
	if created.ID == "" || created.Status != "Pending" {
		t.Fatalf("created approval = %#v", created)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/approvals/"+created.ID+"/approve", strings.NewReader(`{"approver":"reviewer","comment":"ok"}`))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("approve status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Approved") {
		t.Fatalf("approve body = %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/approvals", strings.NewReader(`{"subjectType":"deployment","subjectId":"drun-expire","environmentId":"prod","requestedBy":"tester","reason":"deployment approval"}`))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("create approval for expire status = %d body = %s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode approval for expire: %v", err)
	}
	req = httptest.NewRequest(http.MethodPost, "/api/v1/approvals/"+created.ID+"/expire", strings.NewReader(`{"approver":"system","comment":"expired"}`))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expire status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Expired") {
		t.Fatalf("expire body = %s", rec.Body.String())
	}
}

func TestChangeWindowRoutes(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	router := newTestRouter(cfg)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/change-windows", strings.NewReader(`{"name":"prod-hours","environmentId":"prod","timezone":"Asia/Shanghai","startTime":"09:00","endTime":"17:00","daysOfWeek":["Monday"],"allowed":true}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("create change window status = %d body = %s", rec.Code, rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodPost, "/api/v1/change-windows/evaluate", strings.NewReader(`{"environmentId":"prod","at":"2026-05-18T02:00:00Z"}`))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("evaluate change window status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"allowed":true`) {
		t.Fatalf("evaluate body = %s", rec.Body.String())
	}
}

func newTestPipelineService() *pipelineusecase.Service {
	return pipelineusecase.NewService(
		pipelineusecase.NewMemoryStore(),
		pipelineusecase.NewLocalRunner("test-runner", shellexecutor.New()),
		memory.New(),
	)
}

func newTestRouter(cfg config.Config) http.Handler {
	return newTestRouterWithAuth(cfg, authusecase.NewService(authusecase.NewMemoryStore(), memory.New()))
}

func newTestRouterWithAuth(cfg config.Config, authService *authusecase.Service) http.Handler {
	return newTestRouterWithPipelineAndAuth(cfg, newTestPipelineService(), authService)
}

func newTestRouterWithPipelineAndAuth(cfg config.Config, pipelineService *pipelineusecase.Service, authService *authusecase.Service) http.Handler {
	artifactService := newTestArtifactService()
	deploymentService := newTestDeploymentService()
	approvalService := approvalusecase.NewService(approvalusecase.NewMemoryStore(), noopnotification.New(), memory.New())
	securityService := securityusecase.NewService(securityusecase.NewMemoryStore(), fakeSecurityScanner{}, nil, memory.New())
	releaseService := newTestReleaseOrchestrationService(artifactService, deploymentService)
	return New(
		cfg,
		version.Current(),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		pipelineService,
		deploymentService,
		artifactService,
		releaseService,
		securityService,
		credentialusecase.NewService(credentialusecase.NewMemoryStore(), builtinsecret.New(), memory.New()),
		authService,
		approvalService,
		newTestCloudService(),
		tenancyusecase.NewService(),
		complianceusecase.NewService(pipelineService, deploymentService, artifactService, releaseService, securityService, approvalService),
		pluginusecase.NewDefaultRegistry(),
	)
}

func registerRunnerAndToken(t *testing.T, router http.Handler, id string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runners/register", strings.NewReader(`{"id":"`+id+`","name":"`+id+`","status":"online","executors":["shell"]}`))
	req.Header.Set("Authorization", "Bearer admin-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register %s status = %d body = %s", id, rec.Code, rec.Body.String())
	}
	var registered struct {
		Token struct {
			Token string `json:"token"`
		} `json:"token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &registered); err != nil {
		t.Fatalf("decode runner registration: %v", err)
	}
	if registered.Token.Token == "" {
		t.Fatalf("register %s missing token: %s", id, rec.Body.String())
	}
	return registered.Token.Token
}

type routeOIDCProvider struct{}

func (routeOIDCProvider) Validate(ctx context.Context, token string, issuer string, audience string) (authusecase.OIDCClaims, error) {
	if token != "viewer-token" || issuer != "https://issuer.example" || audience != "nivora" {
		return authusecase.OIDCClaims{}, authusecase.ErrUnauthorized
	}
	return authusecase.OIDCClaims{Subject: "viewer", Username: "viewer", Roles: []string{domainauth.RoleViewer}}, nil
}

type securityOIDCProvider struct{}

func (securityOIDCProvider) Validate(ctx context.Context, token string, issuer string, audience string) (authusecase.OIDCClaims, error) {
	if issuer != "https://issuer.example" || audience != "nivora" {
		return authusecase.OIDCClaims{}, authusecase.ErrUnauthorized
	}
	switch token {
	case "developer-token":
		return authusecase.OIDCClaims{Subject: "developer", Username: "developer", Roles: []string{domainauth.RoleDeveloper}}, nil
	case "auditor-token":
		return authusecase.OIDCClaims{Subject: "auditor", Username: "auditor", Roles: []string{domainauth.RoleAuditor}}, nil
	default:
		return authusecase.OIDCClaims{}, authusecase.ErrUnauthorized
	}
}

func TestCloudRoutes(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	router := newTestRouter(cfg)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cloud/accounts", strings.NewReader(`{"name":"dev-aws","provider":"aws","credentialRef":"cred-placeholder"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("create account status = %d body = %s", rec.Code, rec.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode account: %v", err)
	}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/cloud/accounts/"+created.ID+"/inventory", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("inventory status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "clusters") || strings.Contains(rec.Body.String(), "secret") {
		t.Fatalf("inventory body = %s", rec.Body.String())
	}
}

func newTestDeploymentService() *deploymentusecase.Service {
	return deploymentusecase.NewService(
		deploymentusecase.NewMemoryStore(),
		deploymentusecase.StaticManifestRenderer{},
		yamlapply.NoopManifestClient{},
		allowPolicy{},
		memory.New(),
	).WithGitOps(nil, argocdadapter.NoopProvider{AllowSync: true})
}

func newTestArtifactService() *artifactusecase.Service {
	return artifactusecase.NewService(artifactusecase.NewMemoryStore(), ociartifact.New(), memory.New())
}

func newTestReleaseOrchestrationService(artifactService *artifactusecase.Service, deploymentService *deploymentusecase.Service) *releaseorchestration.Service {
	return releaseorchestration.NewService(
		releaseorchestration.NewMemoryStore(),
		artifactService,
		deploymentService,
		allowPolicy{},
		memory.New(),
	)
}

func newTestCloudService() *cloudusecase.Service {
	return cloudusecase.NewService(cloudusecase.NewMemoryStore(), map[string]portcloud.CloudProvider{"aws": aws.New()}, memory.New())
}

type allowPolicy struct{}

func (allowPolicy) Evaluate(ctx context.Context, request policy.Request) (policy.Result, error) {
	return policy.Result{Allowed: true}, nil
}

type fakeSecurityScanner struct{}

func (fakeSecurityScanner) ScanArtifact(ctx context.Context, request portsecurity.ScanRequest) (portsecurity.ScanResult, error) {
	return portsecurity.ScanResult{Scanner: "fake", Findings: nil}, nil
}

func (fakeSecurityScanner) ScanManifest(ctx context.Context, request portsecurity.ScanRequest) (portsecurity.ScanResult, error) {
	return portsecurity.ScanResult{Scanner: "fake", Findings: nil}, nil
}

func (fakeSecurityScanner) ScanDeploymentPlan(ctx context.Context, request portsecurity.ScanRequest) (portsecurity.ScanResult, error) {
	return portsecurity.ScanResult{Scanner: "fake", Findings: []domainsecurity.SecurityFinding{}}, nil
}

func (fakeSecurityScanner) GetCapabilities(ctx context.Context) ([]portsecurity.Capability, error) {
	return nil, nil
}
