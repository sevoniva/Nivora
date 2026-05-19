package routes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sevoniva/nivora/internal/infra/config"
)

func TestPipelineRunRoutes(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	router := newTestRouter(cfg)

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

	req = httptest.NewRequest(http.MethodGet, "/api/v1/pipeline-runs", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/pipeline-runs/"+runID+"/timeline", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("timeline status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("devops.pipeline.run.completed")) {
		t.Fatalf("timeline body = %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/pipeline-runs/"+runID+"/cancel", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("cancel terminal status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func TestPipelineRunInvalidRequestIncludesRequestID(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	router := newTestRouter(cfg)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/pipeline-runs", bytes.NewReader([]byte(`not-json`)))
	req.Header.Set("X-Request-Id", "test-request-id")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"request_id":"test-request-id"`)) {
		t.Fatalf("missing request id body = %s", rec.Body.String())
	}
}

func TestSystemInfoIncludesRuntimeMode(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	router := newTestRouter(cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/info", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"runtime_mode":"in_memory"`)) {
		t.Fatalf("missing runtime mode body = %s", rec.Body.String())
	}
}

func TestSystemDiagnosticsIncludesCorrelationContext(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	router := newTestRouter(cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/diagnostics", nil)
	req.Header.Set("X-Request-Id", "request-test")
	req.Header.Set("X-Correlation-Id", "correlation-test")
	req.Header.Set("traceparent", "00-0123456789abcdef0123456789abcdef-0123456789abcdef-01")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("X-Correlation-Id") != "correlation-test" {
		t.Fatalf("correlation header = %q", rec.Header().Get("X-Correlation-Id"))
	}
	for _, want := range []string{`"correlation_id":"correlation-test"`, `"trace_id":"0123456789abcdef0123456789abcdef"`, `"metrics"`, `"checks"`} {
		if !bytes.Contains(rec.Body.Bytes(), []byte(want)) {
			t.Fatalf("missing %s body = %s", want, rec.Body.String())
		}
	}
}

func TestRuntimeRecoveryRoutes(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	router := newTestRouter(cfg)

	for _, tc := range []struct {
		method string
		path   string
		field  string
	}{
		{method: http.MethodGet, path: "/api/v1/system/runtime/recovery", field: `"queuedPipelineRuns"`},
		{method: http.MethodPost, path: "/api/v1/system/runtime/reconcile", field: `"publishedOutboxEvents"`},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s %s status = %d body = %s", tc.method, tc.path, rec.Code, rec.Body.String())
		}
		if !bytes.Contains(rec.Body.Bytes(), []byte(tc.field)) {
			t.Fatalf("missing %s body = %s", tc.field, rec.Body.String())
		}
	}
}

func TestMetricsEndpointExposesRuntimeCounters(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	router := newTestRouter(cfg)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	for _, want := range []string{"nivora_pipeline_run_total", "nivora_deployment_run_total", "nivora_runner_heartbeat_total"} {
		if !bytes.Contains(rec.Body.Bytes(), []byte(want)) {
			t.Fatalf("missing metric %s body = %s", want, rec.Body.String())
		}
	}
}

func TestRunnerRoutes(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	router := newTestRouter(cfg)

	body := []byte(`{"id":"runner-api","name":"runner-api","status":"online","executors":["shell"],"labels":{"tier":"dev"}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runners/register", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register status = %d body = %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/runners/runner-api/heartbeat", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("heartbeat status = %d body = %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/runners", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list runners status = %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/runners/runner-api/jobs/claim", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("claim without queued job status = %d body = %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/jobs/job-missing/status", bytes.NewReader([]byte(`{"status":"Running"}`)))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing job status = %d body = %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/jobs/job-missing/logs", bytes.NewReader([]byte(`{"pipelineRunId":"run-missing","stream":"stdout","content":"hello"}`)))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing job log status = %d body = %s", rec.Code, rec.Body.String())
	}
}
