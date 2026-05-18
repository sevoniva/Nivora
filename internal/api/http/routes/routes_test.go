package routes

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sevoniva/nivora/internal/adapters/eventbus/memory"
	shellexecutor "github.com/sevoniva/nivora/internal/adapters/executor/shell"
	"github.com/sevoniva/nivora/internal/infra/config"
	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
	"github.com/sevoniva/nivora/internal/version"
)

func TestHealthRoutes(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	router := New(cfg, version.Current(), slog.New(slog.NewTextHandler(io.Discard, nil)), newTestPipelineService())

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
	router := New(cfg, version.Current(), slog.New(slog.NewTextHandler(io.Discard, nil)), newTestPipelineService())
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

func newTestPipelineService() *pipelineusecase.Service {
	return pipelineusecase.NewService(
		pipelineusecase.NewMemoryStore(),
		pipelineusecase.NewLocalRunner("test-runner", shellexecutor.New()),
		memory.New(),
	)
}
