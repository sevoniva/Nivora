package routes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sevoniva/nivora/internal/infra/config"
)

func TestVisualizationRoutes(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	router := newTestRouter(cfg)

	pipelineID := createVisualizationPipelineRun(t, router)
	deploymentID := createVisualizationDeploymentRun(t, router)
	releaseID, executionID := createVisualizationReleaseExecution(t, router)
	createVisualizationSecurityScan(t, router)
	registerVisualizationRunner(t, router)

	assertVisualizationObject(t, router, "/api/v1/visualization/pipeline-runs/"+pipelineID+"/dag", "nodes")
	assertVisualizationTimeline(t, router, "/api/v1/visualization/pipeline-runs/"+pipelineID+"/timeline")
	assertVisualizationObject(t, router, "/api/v1/visualization/pipeline-runs/"+pipelineID+"/summary", "status")

	assertVisualizationTimeline(t, router, "/api/v1/visualization/deployments/"+deploymentID+"/timeline")
	assertVisualizationArray(t, router, "/api/v1/visualization/deployments/"+deploymentID+"/resources")
	assertVisualizationObject(t, router, "/api/v1/visualization/deployments/"+deploymentID+"/diff", "summary")
	assertVisualizationObject(t, router, "/api/v1/visualization/deployments/"+deploymentID+"/health", "status")

	assertVisualizationObject(t, router, "/api/v1/visualization/releases/"+releaseID+"/overview", "summary")
	assertVisualizationTimeline(t, router, "/api/v1/visualization/releases/executions/"+executionID+"/timeline")
	assertVisualizationArray(t, router, "/api/v1/visualization/releases/executions/"+executionID+"/targets")

	assertVisualizationObject(t, router, "/api/v1/visualization/environments/dev/topology", "healthSummary")
	assertVisualizationObject(t, router, "/api/v1/visualization/runners/summary", "runners")
	assertVisualizationObject(t, router, "/api/v1/visualization/security/summary", "findings")
	assertVisualizationTimeline(t, router, "/api/v1/visualization/audit/timeline")
}

func createVisualizationPipelineRun(t *testing.T, router http.Handler) string {
	t.Helper()
	body := []byte(`{
		"apiVersion": "nivora.io/v1alpha1",
		"kind": "Pipeline",
		"metadata": {"name": "visualization-pipeline"},
		"spec": {"stages": [{"name": "build", "jobs": [{"name": "echo", "executor": "shell", "steps": [{"name": "say", "run": "printf hello"}]}]}]}
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pipeline-runs", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create pipeline status = %d body = %s", rec.Code, rec.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode pipeline: %v", err)
	}
	return created["run"].(map[string]any)["id"].(string)
}

func createVisualizationDeploymentRun(t *testing.T, router http.Handler) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/deployments", bytes.NewReader(deploymentRequestBody(t)))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create deployment status = %d body = %s", rec.Code, rec.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode deployment: %v", err)
	}
	return created["run"].(map[string]any)["id"].(string)
}

func createVisualizationReleaseExecution(t *testing.T, router http.Handler) (string, string) {
	t.Helper()
	body := `{
	  "apiVersion":"nivora.io/v1alpha1",
	  "kind":"ReleaseOrchestration",
	  "metadata":{"name":"visualization-release"},
	  "spec":{
	    "environment":"dev",
	    "strategy":"plan-only",
	    "release":{
	      "apiVersion":"nivora.io/v1alpha1",
	      "kind":"Release",
	      "metadata":{"name":"visualization-demo"},
	      "spec":{
	        "version":"1.0.0",
	        "application":"visualization-demo",
	        "environment":"dev",
	        "artifacts":[{"name":"visualization-demo","type":"image","required":true,"reference":"registry.example.com/demo/api:1.0.0"}]
	      }
	    },
	    "targets":[{"name":"audit-only","type":"noop","order":1}]
	  }
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/releases/local/deploy", strings.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("deploy release status = %d body = %s", rec.Code, rec.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode release execution: %v", err)
	}
	releaseID := created["release"].(map[string]any)["id"].(string)
	executionID := created["execution"].(map[string]any)["id"].(string)
	return releaseID, executionID
}

func createVisualizationSecurityScan(t *testing.T, router http.Handler) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/security/scans", strings.NewReader(`{"subjectType":"artifact","subjectId":"registry.example.com/demo/app:latest","reference":"registry.example.com/demo/app:latest"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create security scan status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func registerVisualizationRunner(t *testing.T, router http.Handler) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runners/register", strings.NewReader(`{"id":"visualization-runner","name":"visualization-runner","status":"online","executors":["shell"]}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register runner status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func assertVisualizationObject(t *testing.T, router http.Handler, path string, field string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("%s status = %d body = %s", path, rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("%s response is not object json: %v", path, err)
	}
	if _, ok := body[field]; !ok {
		t.Fatalf("%s missing field %q body = %s", path, field, rec.Body.String())
	}
}

func assertVisualizationArray(t *testing.T, router http.Handler, path string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("%s status = %d body = %s", path, rec.Code, rec.Body.String())
	}
	var body []any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("%s response is not array json: %v", path, err)
	}
	if len(body) == 0 {
		t.Fatalf("%s returned empty array", path)
	}
}

func assertVisualizationTimeline(t *testing.T, router http.Handler, path string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("%s status = %d body = %s", path, rec.Code, rec.Body.String())
	}
	var body []struct {
		Time time.Time `json:"time"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("%s response is not timeline json: %v", path, err)
	}
	if len(body) == 0 {
		t.Fatalf("%s returned empty timeline", path)
	}
	for i := 1; i < len(body); i++ {
		if body[i].Time.Before(body[i-1].Time) {
			t.Fatalf("%s timeline is not ordered", path)
		}
	}
}
