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

	req = httptest.NewRequest(http.MethodPost, "/api/v1/artifact-registries/areg-local/validate", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("validate status = %d body = %s", rec.Code, rec.Body.String())
	}
	var validation struct {
		Valid      bool     `json:"valid"`
		RegistryID string   `json:"registryId"`
		Enabled    bool     `json:"enabled"`
		Warnings   []string `json:"warnings"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &validation); err != nil {
		t.Fatalf("decode validation: %v", err)
	}
	if !validation.Valid || validation.RegistryID != "areg-local" || !validation.Enabled {
		t.Fatalf("unexpected validation result: %+v body=%s", validation, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "cred-registry") || strings.Contains(rec.Body.String(), "password") || strings.Contains(rec.Body.String(), "token") {
		t.Fatalf("validation response leaked credential-like data: %s", rec.Body.String())
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

	req = httptest.NewRequest(http.MethodPost, "/api/v1/artifact-registries/areg-local/validate", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("validate disabled status = %d body = %s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &validation); err != nil {
		t.Fatalf("decode disabled validation: %v", err)
	}
	if validation.Valid || validation.Enabled {
		t.Fatalf("disabled registry should be invalid: %+v body=%s", validation, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "disabled") {
		t.Fatalf("disabled validation should include warning: %s", rec.Body.String())
	}
}
