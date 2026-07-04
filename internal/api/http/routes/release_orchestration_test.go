package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sevoniva/nivora/internal/infra/config"
)

func TestReleaseOrchestrationRoutes(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	router := newTestRouter(cfg)
	body := `{
	  "apiVersion":"nivora.io/v1alpha1",
	  "kind":"ReleaseOrchestration",
	  "metadata":{"name":"api-release-orchestration"},
	  "spec":{
	    "environment":"dev",
	    "strategy":"plan-only",
	    "release":{
	      "apiVersion":"nivora.io/v1alpha1",
	      "kind":"Release",
	      "metadata":{"name":"api-demo"},
	      "spec":{
	        "version":"1.0.0",
	        "application":"api-demo",
	        "environment":"dev",
	        "artifacts":[{"name":"api-demo","type":"image","required":true,"reference":"registry.example.com/demo/api:1.0.0"}]
	      }
	    },
	    "targets":[{"name":"audit-only","type":"noop","order":1}]
	  }
	}`
	for _, path := range []string{"/api/v1/releases/local/plan", "/api/v1/releases/local/deploy"} {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK && rec.Code != http.StatusCreated {
			t.Fatalf("%s status = %d body = %s", path, rec.Code, rec.Body.String())
		}
	}
}

func TestReleasePlanAndDeployUsePathReleaseID(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	router := newTestRouter(cfg)
	releaseBody := `{
	  "apiVersion":"nivora.io/v1alpha1",
	  "kind":"Release",
	  "metadata":{"name":"path-release"},
	  "spec":{
	    "version":"1.0.0",
	    "application":"api-demo",
	    "environment":"dev",
	    "artifacts":[{"name":"api-demo","type":"image","required":true,"reference":"registry.example.com/demo/api@sha256:abcdef"}]
	  }
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/releases", strings.NewReader(releaseBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create release status = %d body = %s", rec.Code, rec.Body.String())
	}
	var created struct {
		Release struct {
			ID string `json:"id"`
		} `json:"release"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode release: %v", err)
	}
	if created.Release.ID == "" {
		t.Fatalf("release id missing: %s", rec.Body.String())
	}

	body := `{
	  "apiVersion":"nivora.io/v1alpha1",
	  "kind":"ReleaseOrchestration",
	  "metadata":{"name":"path-id-orchestration"},
	  "spec":{
	    "environment":"dev",
	    "strategy":"plan-only",
	    "targets":[{"name":"audit-only","type":"noop","order":1}]
	  }
	}`
	for _, tc := range []struct {
		method string
		path   string
		status int
		field  string
	}{
		{method: http.MethodPost, path: "/api/v1/releases/" + created.Release.ID + "/plan", status: http.StatusOK, field: "plan"},
		{method: http.MethodPost, path: "/api/v1/releases/" + created.Release.ID + "/deploy", status: http.StatusCreated, field: "execution"},
	} {
		req = httptest.NewRequest(tc.method, tc.path, strings.NewReader(body))
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != tc.status {
			t.Fatalf("%s status = %d body = %s", tc.path, rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), `"releaseId":"`+created.Release.ID+`"`) || !strings.Contains(rec.Body.String(), `"`+tc.field+`"`) {
			t.Fatalf("%s did not use path release id: %s", tc.path, rec.Body.String())
		}
	}
}
