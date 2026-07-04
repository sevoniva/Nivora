package routes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sevoniva/nivora/internal/infra/config"
)

func TestArtifactAndReleaseRoutes(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	router := newTestRouter(cfg)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/artifacts/inspect", bytes.NewReader([]byte(`{"reference":"nginx:latest","type":"image"}`)))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("inspect status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("mutable_latest_tag")) {
		t.Fatalf("inspect body = %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/artifacts", bytes.NewReader([]byte(`{"name":"tracked-demo","type":"image","reference":"registry.example.com/team/tracked:1.0.0"}`)))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create artifact status = %d body = %s", rec.Code, rec.Body.String())
	}
	var trackedArtifact map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &trackedArtifact); err != nil {
		t.Fatalf("decode tracked artifact: %v", err)
	}
	trackedArtifactID := trackedArtifact["id"].(string)
	if trackedArtifact["reference"].(string) != "registry.example.com/team/tracked:1.0.0" {
		t.Fatalf("tracked artifact = %#v", trackedArtifact)
	}

	body := []byte(`{
		"apiVersion": "nivora.io/v1alpha1",
		"kind": "Release",
		"metadata": {"name": "demo"},
		"spec": {
			"version": "1.0.0",
			"application": "demo-app",
			"artifacts": [{"name": "demo-app", "type": "image", "reference": "registry.example.com/team/demo@sha256:abcdef", "required": true}]
		}
	}`)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/releases", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create release status = %d body = %s", rec.Code, rec.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode release: %v", err)
	}
	releaseObj := created["release"].(map[string]any)
	releaseID := releaseObj["id"].(string)
	artifactList := created["artifacts"].([]any)
	artifactID := artifactList[0].(map[string]any)["id"].(string)

	deployBody := []byte(`{
		"apiVersion": "nivora.io/v1alpha1",
		"kind": "ReleaseOrchestration",
		"metadata": {"name": "cancel-cascade"},
		"spec": {
			"releaseId": "` + releaseID + `",
			"environment": "dev",
			"approvalRequired": true,
			"targets": [{"name": "approval-noop", "type": "noop"}]
		}
	}`)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/releases/"+releaseID+"/deploy", bytes.NewReader(deployBody))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create release execution status = %d body = %s", rec.Code, rec.Body.String())
	}
	var executionCreated map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &executionCreated); err != nil {
		t.Fatalf("decode release execution: %v", err)
	}
	executionObj := executionCreated["execution"].(map[string]any)
	executionID := executionObj["id"].(string)
	if executionObj["status"].(string) != "WaitingApproval" {
		t.Fatalf("execution status = %s, want WaitingApproval", executionObj["status"].(string))
	}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/releases/"+releaseID, nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get waiting release status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"status":"WaitingApproval"`)) || !bytes.Contains(rec.Body.Bytes(), []byte("devops.release.status.updated")) {
		t.Fatalf("release should expose waiting approval status and audit event: %s", rec.Body.String())
	}

	for _, path := range []string{
		"/api/v1/releases",
		"/api/v1/releases/" + releaseID,
		"/api/v1/releases/" + releaseID + "/artifacts",
		"/api/v1/artifacts",
		"/api/v1/artifacts?registry=registry.example.com",
		"/api/v1/artifacts?reference=team/tracked",
		"/api/v1/artifacts/" + trackedArtifactID,
		"/api/v1/artifacts/" + artifactID,
		"/api/v1/artifacts/" + artifactID + "/releases",
	} {
		req = httptest.NewRequest(http.MethodGet, path, nil)
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status = %d body = %s", path, rec.Code, rec.Body.String())
		}
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/releases/"+releaseID+"/cancel", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("release cancel status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"status":"Canceled"`)) || !bytes.Contains(rec.Body.Bytes(), []byte("devops.release.canceled")) {
		t.Fatalf("release cancel body = %s", rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"canceledExecutionIds":["`+executionID+`"]`)) {
		t.Fatalf("release cancel should include canceled execution id %s: %s", executionID, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/artifacts/missing", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing artifact status = %d body = %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/releases/"+releaseID+"/evidence", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("release evidence status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"subjectType":"release"`)) || !bytes.Contains(rec.Body.Bytes(), []byte(`"artifacts"`)) {
		t.Fatalf("release evidence body = %s", rec.Body.String())
	}
}
