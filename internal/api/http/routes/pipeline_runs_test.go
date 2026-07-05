package routes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sevoniva/nivora/internal/api/http/handlers"
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

	req = httptest.NewRequest(http.MethodGet, "/api/v1/pipeline-runs/"+runID+"/dag", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("dag status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"nodes"`)) || !bytes.Contains(rec.Body.Bytes(), []byte(`"edges"`)) {
		t.Fatalf("dag body = %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/pipeline-runs/"+runID+"/jobs", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("jobs status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"job"`)) || !bytes.Contains(rec.Body.Bytes(), []byte(`"steps"`)) {
		t.Fatalf("jobs body = %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/pipeline-runs/"+runID+"/steps?limit=1", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("steps status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"pagination"`)) || !bytes.Contains(rec.Body.Bytes(), []byte(`"items"`)) {
		t.Fatalf("steps body = %s", rec.Body.String())
	}

	for _, path := range []string{"/artifacts", "/caches", "/annotations"} {
		req = httptest.NewRequest(http.MethodGet, "/api/v1/pipeline-runs/"+runID+path, nil)
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status = %d body = %s", path, rec.Code, rec.Body.String())
		}
		if !bytes.Contains(rec.Body.Bytes(), []byte("[]")) {
			t.Fatalf("%s body = %s", path, rec.Body.String())
		}
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/pipeline-runs/"+runID+"/summary", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("summary status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"artifactCount":0`)) || !bytes.Contains(rec.Body.Bytes(), []byte(`"annotationCount":0`)) {
		t.Fatalf("summary body = %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/pipeline-runs", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/pipeline-runs?limit=1&offset=0", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("paginated list status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"pagination"`)) || !bytes.Contains(rec.Body.Bytes(), []byte(`"limit":1`)) {
		t.Fatalf("paginated list body = %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/pipeline-runs/"+runID+"/logs?limit=1", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("paginated logs status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"pagination"`)) || !bytes.Contains(rec.Body.Bytes(), []byte(`"items"`)) {
		t.Fatalf("paginated logs body = %s", rec.Body.String())
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

func TestPipelineRunCreateAttachesEnvironment(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	router := newTestRouter(cfg)

	body := []byte(`{
		"apiVersion": "nivora.io/v1alpha1",
		"kind": "Pipeline",
		"metadata": {"name": "hello-env"},
		"spec": {"stages": [{"name": "build", "jobs": [{"name": "echo", "executor": "shell", "steps": [{"name": "say", "run": "printf hello-env"}]}]}]}
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pipeline-runs?environmentId=env-prod", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d body = %s", rec.Code, rec.Body.String())
	}

	var created map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	pipeline := created["pipeline"].(map[string]any)
	if pipeline["environmentId"] != "env-prod" {
		t.Fatalf("environment id = %#v body=%s", pipeline["environmentId"], rec.Body.String())
	}
	labels, _ := pipeline["labels"].(map[string]any)
	metadata, _ := pipeline["metadata"].(map[string]any)
	if labels["environmentId"] != "env-prod" || metadata["environmentId"] != "env-prod" {
		t.Fatalf("environment ownership missing: labels=%#v metadata=%#v body=%s", labels, metadata, rec.Body.String())
	}
}

func TestPipelinePaginationAndLogLimits(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	router := newTestRouter(cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pipeline-runs?limit=501", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid pagination status = %d body = %s", rec.Code, rec.Body.String())
	}

	body := []byte(`{"apiVersion":"nivora.io/v1alpha1","kind":"Pipeline","metadata":{"name":"limit-test"},"spec":{"stages":[{"name":"build","jobs":[{"name":"job","executor":"shell","steps":[{"name":"step","run":"printf ok"}]}]}]}}`)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/pipeline-runs", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d body = %s", rec.Code, rec.Body.String())
	}
	var created struct {
		Run struct {
			ID string `json:"id"`
		} `json:"run"`
		Stages []struct {
			Jobs []struct {
				Job struct {
					ID string `json:"id"`
				} `json:"job"`
			} `json:"jobs"`
		} `json:"stages"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	large := bytes.Repeat([]byte("x"), handlers.MaxLogChunkBytes+1)
	payload, err := json.Marshal(map[string]string{
		"pipelineRunId": created.Run.ID,
		"stream":        "stdout",
		"content":       string(large),
	})
	if err != nil {
		t.Fatalf("marshal log append: %v", err)
	}
	req = httptest.NewRequest(http.MethodPost, "/api/v1/jobs/"+created.Stages[0].Jobs[0].Job.ID+"/logs", bytes.NewReader(payload))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("large log status = %d body = %s", rec.Code, rec.Body.String())
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
	for _, want := range []string{`"correlation_id":"correlation-test"`, `"trace_id":"0123456789abcdef0123456789abcdef"`, `"metrics"`, `"checks"`, `"database"`, `"event_outbox"`} {
		if !bytes.Contains(rec.Body.Bytes(), []byte(want)) {
			t.Fatalf("missing %s body = %s", want, rec.Body.String())
		}
	}
}

func TestReadinessReportsFailedDependencyConfig(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	cfg.Database.RuntimeStore = "postgres"
	cfg.Database.URL = ""
	router := newTestRouter(cfg)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	for _, want := range []string{`"status":"degraded"`, `"name":"database"`, "database.url is empty"} {
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
		{method: http.MethodGet, path: "/api/v1/system/runtime/recovery", field: `"nonTerminalDeploymentRuns"`},
		{method: http.MethodPost, path: "/api/v1/system/runtime/reconcile", field: `"staleReleaseExecutions"`},
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
	var registered map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &registered); err != nil {
		t.Fatalf("decode register response: %v", err)
	}
	token, ok := registered["token"].(map[string]any)["token"].(string)
	if !ok || token == "" {
		t.Fatalf("missing one-time token body = %s", rec.Body.String())
	}
	if bytes.Contains(rec.Body.Bytes(), []byte("tokenHash")) {
		t.Fatalf("token hash leaked body = %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/runners/runner-api/heartbeat", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("heartbeat without token status = %d body = %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/runners/runner-api/heartbeat", nil)
	req.Header.Set("X-Nivora-Runner-Token", token)
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
	req.Header.Set("X-Nivora-Runner-Token", token)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("claim without queued job status = %d body = %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/runners/runner-api/token/rotate", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("rotate status = %d body = %s", rec.Code, rec.Body.String())
	}
	if bytes.Contains(rec.Body.Bytes(), []byte("tokenHash")) {
		t.Fatalf("token hash leaked after rotate body = %s", rec.Body.String())
	}
	var rotated map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &rotated); err != nil {
		t.Fatalf("decode rotate response: %v", err)
	}
	rotatedToken, ok := rotated["token"].(map[string]any)["token"].(string)
	if !ok || rotatedToken == "" || rotatedToken == token {
		t.Fatalf("rotation did not return a new one-time token body = %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/runners/runner-api/heartbeat", nil)
	req.Header.Set("X-Nivora-Runner-Token", token)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("heartbeat with old token after rotate status = %d body = %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/runners/runner-api/heartbeat", nil)
	req.Header.Set("X-Nivora-Runner-Token", rotatedToken)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("heartbeat with rotated token status = %d body = %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/runners/runner-api/token/revoke", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("revoke status = %d body = %s", rec.Code, rec.Body.String())
	}
	if bytes.Contains(rec.Body.Bytes(), []byte("tokenHash")) || bytes.Contains(rec.Body.Bytes(), []byte(rotatedToken)) {
		t.Fatalf("token material leaked after revoke body = %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/runners/runner-api/heartbeat", nil)
	req.Header.Set("X-Nivora-Runner-Token", rotatedToken)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("heartbeat with revoked token status = %d body = %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/runners/offline-detect?timeoutSeconds=0", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("offline detect status = %d body = %s", rec.Code, rec.Body.String())
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
