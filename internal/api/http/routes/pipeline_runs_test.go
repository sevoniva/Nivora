package routes

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sevoniva/nivora/internal/infra/config"
	"github.com/sevoniva/nivora/internal/version"
)

func TestPipelineRunRoutes(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	router := New(cfg, version.Current(), slog.New(slog.NewTextHandler(io.Discard, nil)), newTestPipelineService())

	body := []byte(`{
		"apiVersion": "nivora.io/v1alpha1",
		"kind": "Pipeline",
		"metadata": {"name": "hello"},
		"spec": {"stages": [{"name": "build", "jobs": [{"name": "echo", "executor": "shell", "steps": [{"name": "say", "run": "printf hello"}]}]}]}
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pipeline-runs", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d body = %s", rec.Code, rec.Body.String())
	}

	var created map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	run := created["run"].(map[string]any)
	runID := run["id"].(string)

	req = httptest.NewRequest(http.MethodGet, "/api/v1/pipeline-runs/"+runID+"/logs", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("logs status = %d", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("hello")) {
		t.Fatalf("logs body = %s", rec.Body.String())
	}
}
