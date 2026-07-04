package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sevoniva/nivora/internal/infra/config"
)

func TestPolicyCatalogRoutes(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	router := newTestRouter(cfg)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/policies", strings.NewReader(`{"id":"policy-digest","projectId":"project-a","environmentId":"prod","name":"Require digest","requireDigest":true}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d body = %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/policies?projectId=project-a&environmentId=prod", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d body = %s", rec.Code, rec.Body.String())
	}
	var listed struct {
		Policies []map[string]any `json:"policies"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(listed.Policies) != 1 {
		t.Fatalf("expected one policy, got %d", len(listed.Policies))
	}

	req = httptest.NewRequest(http.MethodPatch, "/api/v1/policies/policy-digest", strings.NewReader(`{"highWarnThreshold":2}`))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("update status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"highWarnThreshold":2`) {
		t.Fatalf("updated policy missing threshold: %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/v1/policies/policy-digest", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("disable status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"enabled":false`) {
		t.Fatalf("disabled policy missing enabled=false: %s", rec.Body.String())
	}
}
