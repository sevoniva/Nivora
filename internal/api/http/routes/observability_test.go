package routes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sevoniva/nivora/internal/infra/config"
)

func TestAggregateObservabilityRoutes(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	router := newTestRouter(cfg)

	body := []byte(`{
		"apiVersion": "nivora.io/v1alpha1",
		"kind": "Pipeline",
		"metadata": {"name": "observability"},
		"spec": {"stages": [{"name": "build", "jobs": [{"name": "echo", "executor": "shell", "steps": [{"name": "say", "run": "printf observable"}]}]}]}
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pipeline-runs", bytes.NewReader(body))
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
		t.Fatalf("decode pipeline run: %v", err)
	}
	if created.Run.ID == "" {
		t.Fatalf("pipeline run response missing id: %s", rec.Body.String())
	}

	for _, path := range []string{"/api/v1/events", "/api/v1/logs"} {
		req = httptest.NewRequest(http.MethodGet, path, nil)
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status = %d body = %s", path, rec.Code, rec.Body.String())
		}
		if !bytes.Contains(rec.Body.Bytes(), []byte(`"count"`)) {
			t.Fatalf("%s missing count body = %s", path, rec.Body.String())
		}
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/logs?limit=1", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("paginated logs status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"pagination"`)) {
		t.Fatalf("paginated logs missing pagination body = %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/events?pipelineRunId="+created.Run.ID+"&type=devops.pipeline.run.created&limit=1", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("filtered events status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"pagination"`)) || !bytes.Contains(rec.Body.Bytes(), []byte(`devops.pipeline.run.created`)) {
		t.Fatalf("filtered events body = %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/events?pipelineRunId=missing-run", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("missing-run events status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"count":0`)) {
		t.Fatalf("missing-run events should be empty: %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/logs?pipelineRunId="+created.Run.ID+"&stream=stdout&contains=observable", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("filtered logs status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`observable`)) {
		t.Fatalf("filtered logs missing content: %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/logs?pipelineRunId=missing-run", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("missing-run logs status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"count":0`)) {
		t.Fatalf("missing-run logs should be empty: %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/audit-logs", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("audit logs status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"count"`)) {
		t.Fatalf("audit logs missing count body = %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/audit/search?subject="+created.Run.ID+"&limit=1", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("filtered audit search status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"pagination"`)) || !bytes.Contains(rec.Body.Bytes(), []byte(created.Run.ID)) {
		t.Fatalf("filtered audit search body = %s", rec.Body.String())
	}
}
