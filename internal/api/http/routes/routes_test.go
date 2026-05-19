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
	domainsecurity "github.com/sevoniva/nivora/internal/domain/security"
	"github.com/sevoniva/nivora/internal/infra/config"
	portcloud "github.com/sevoniva/nivora/internal/ports/cloud"
	"github.com/sevoniva/nivora/internal/ports/policy"
	portsecurity "github.com/sevoniva/nivora/internal/ports/security"
	approvalusecase "github.com/sevoniva/nivora/internal/usecase/approval"
	artifactusecase "github.com/sevoniva/nivora/internal/usecase/artifact"
	authusecase "github.com/sevoniva/nivora/internal/usecase/auth"
	cloudusecase "github.com/sevoniva/nivora/internal/usecase/cloud"
	credentialusecase "github.com/sevoniva/nivora/internal/usecase/credential"
	deploymentusecase "github.com/sevoniva/nivora/internal/usecase/deployment"
	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
	pluginusecase "github.com/sevoniva/nivora/internal/usecase/plugin"
	releaseorchestration "github.com/sevoniva/nivora/internal/usecase/releaseorchestration"
	securityusecase "github.com/sevoniva/nivora/internal/usecase/security"
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
	req := httptest.NewRequest(http.MethodGet, "/api/v1/pipelines", nil)
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
	if body["path"] != "/api/v1/pipelines" {
		t.Fatalf("path = %v", body["path"])
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

	req = httptest.NewRequest(http.MethodGet, "/api/v1/secrets/refs", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list secret refs status = %d body = %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "sample-value-for-test-only") {
		t.Fatalf("secret refs response leaked secret value")
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
}

func newTestPipelineService() *pipelineusecase.Service {
	return pipelineusecase.NewService(
		pipelineusecase.NewMemoryStore(),
		pipelineusecase.NewLocalRunner("test-runner", shellexecutor.New()),
		memory.New(),
	)
}

func newTestRouter(cfg config.Config) http.Handler {
	artifactService := newTestArtifactService()
	deploymentService := newTestDeploymentService()
	return New(
		cfg,
		version.Current(),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		newTestPipelineService(),
		deploymentService,
		artifactService,
		newTestReleaseOrchestrationService(artifactService, deploymentService),
		securityusecase.NewService(securityusecase.NewMemoryStore(), fakeSecurityScanner{}, nil, memory.New()),
		credentialusecase.NewService(credentialusecase.NewMemoryStore(), builtinsecret.New(), memory.New()),
		authusecase.NewService(authusecase.NewMemoryStore(), memory.New()),
		approvalusecase.NewService(approvalusecase.NewMemoryStore(), noopnotification.New(), memory.New()),
		newTestCloudService(),
		pluginusecase.NewDefaultRegistry(),
	)
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
