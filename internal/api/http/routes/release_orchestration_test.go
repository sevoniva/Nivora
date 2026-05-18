package routes

import (
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
