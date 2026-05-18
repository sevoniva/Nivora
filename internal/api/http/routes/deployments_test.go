package routes

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/sevoniva/nivora/internal/infra/config"
	"github.com/sevoniva/nivora/internal/version"
)

func TestDeploymentRoutes(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	router := New(cfg, version.Current(), slog.New(slog.NewTextHandler(io.Discard, nil)), newTestPipelineService(), newTestDeploymentService())
	body := deploymentRequestBody(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/deployments/plan", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("plan status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"manifestCount":1`)) {
		t.Fatalf("plan body = %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/deployments", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d body = %s", rec.Code, rec.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	run := created["run"].(map[string]any)
	runID := run["id"].(string)
	if run["status"] != "Succeeded" {
		t.Fatalf("status = %v", run["status"])
	}

	for _, path := range []string{
		"/api/v1/deployments/" + runID,
		"/api/v1/deployments/" + runID + "/plan",
		"/api/v1/deployments/" + runID + "/logs",
		"/api/v1/deployments/" + runID + "/events",
		"/api/v1/deployments/" + runID + "/timeline",
	} {
		req = httptest.NewRequest(http.MethodGet, path, nil)
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status = %d body = %s", path, rec.Code, rec.Body.String())
		}
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/deployments/"+runID+"/cancel", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("cancel terminal status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func deploymentRequestBody(t *testing.T) []byte {
	t.Helper()
	dir := t.TempDir()
	manifest := filepath.Join(dir, "deployment.yaml")
	if err := os.WriteFile(manifest, []byte(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: demo
`), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	body, err := json.Marshal(map[string]any{
		"apiVersion": "nivora.io/v1alpha1",
		"kind":       "Deployment",
		"metadata": map[string]any{
			"name": "demo-yaml",
		},
		"spec": map[string]any{
			"application": "demo-app",
			"environment": "dev",
			"target": map[string]any{
				"type":      "kubernetes-yaml",
				"name":      "dev-kind",
				"namespace": "default",
			},
			"artifacts": []map[string]any{{
				"name":      "demo-app",
				"type":      "image",
				"reference": "example.local/demo:dev",
			}},
			"manifests": []string{manifest},
			"options": map[string]any{
				"dryRun": true,
				"apply":  false,
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	return body
}
