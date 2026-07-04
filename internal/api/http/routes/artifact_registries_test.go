package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sevoniva/nivora/internal/infra/config"
)

func TestArtifactRegistryCatalogRoutes(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	router := newTestRouter(cfg)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/artifact-registries", strings.NewReader(`{"id":"areg-local","projectId":"project-a","name":"local","type":"oci","endpoint":"http://localhost:30500","insecure":true,"credentialRef":"cred-registry"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d body = %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "password") || strings.Contains(rec.Body.String(), "token") {
		t.Fatalf("registry response leaked secret-like data: %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/artifact-registries?projectId=project-a", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d body = %s", rec.Code, rec.Body.String())
	}
	var listed struct {
		Registries []map[string]any `json:"registries"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(listed.Registries) != 1 {
		t.Fatalf("expected one registry, got %d", len(listed.Registries))
	}

	req = httptest.NewRequest(http.MethodPatch, "/api/v1/artifact-registries/areg-local", strings.NewReader(`{"endpoint":"registry.example.com"}`))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("update status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"endpoint":"registry.example.com"`) {
		t.Fatalf("updated registry missing endpoint: %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/v1/artifact-registries/areg-local", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("disable status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"enabled":false`) {
		t.Fatalf("disabled registry missing enabled=false: %s", rec.Body.String())
	}
}
