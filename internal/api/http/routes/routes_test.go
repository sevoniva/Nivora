package routes

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	ociartifact "github.com/sevoniva/nivora/internal/adapters/artifact/oci"
	"github.com/sevoniva/nivora/internal/adapters/eventbus/memory"
	shellexecutor "github.com/sevoniva/nivora/internal/adapters/executor/shell"
	yamlapply "github.com/sevoniva/nivora/internal/adapters/executor/yaml_apply"
	"github.com/sevoniva/nivora/internal/infra/config"
	"github.com/sevoniva/nivora/internal/ports/policy"
	artifactusecase "github.com/sevoniva/nivora/internal/usecase/artifact"
	deploymentusecase "github.com/sevoniva/nivora/internal/usecase/deployment"
	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
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

func newTestPipelineService() *pipelineusecase.Service {
	return pipelineusecase.NewService(
		pipelineusecase.NewMemoryStore(),
		pipelineusecase.NewLocalRunner("test-runner", shellexecutor.New()),
		memory.New(),
	)
}

func newTestRouter(cfg config.Config) http.Handler {
	return New(
		cfg,
		version.Current(),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		newTestPipelineService(),
		newTestDeploymentService(),
		newTestArtifactService(),
	)
}

func newTestDeploymentService() *deploymentusecase.Service {
	return deploymentusecase.NewService(
		deploymentusecase.NewMemoryStore(),
		deploymentusecase.StaticManifestRenderer{},
		yamlapply.NoopManifestClient{},
		allowPolicy{},
		memory.New(),
	)
}

func newTestArtifactService() *artifactusecase.Service {
	return artifactusecase.NewService(artifactusecase.NewMemoryStore(), ociartifact.New(), memory.New())
}

type allowPolicy struct{}

func (allowPolicy) Evaluate(ctx context.Context, request policy.Request) (policy.Result, error) {
	return policy.Result{Allowed: true}, nil
}
