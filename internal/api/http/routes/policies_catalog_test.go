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

	req = httptest.NewRequest(http.MethodPost, "/api/v1/policies/policy-digest/attachments", strings.NewReader(`{"id":"attach-prod","scopeType":"environment","scopeId":"prod"}`))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("attach status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"scopeType":"environment"`) {
		t.Fatalf("attachment missing scope: %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/policies/policy-digest/attachments", strings.NewReader(`{"scopeType":"environment","scopeId":"prod"}`))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate attach status = %d body = %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/policies/policy-digest/attachments?scopeType=environment", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list attachments status = %d body = %s", rec.Code, rec.Body.String())
	}
	var attachments struct {
		Attachments []map[string]any `json:"attachments"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &attachments); err != nil {
		t.Fatalf("decode attachments: %v", err)
	}
	if len(attachments.Attachments) != 1 {
		t.Fatalf("expected one attachment, got %d", len(attachments.Attachments))
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/policies/policy-digest/evaluate", strings.NewReader(`{"subjectType":"artifact","subjectId":"registry.example.invalid/demo/app:1.0.0","reference":"registry.example.invalid/demo/app:1.0.0"}`))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("evaluate status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"policyId":"policy-digest"`) || !strings.Contains(rec.Body.String(), `"decision":"deny"`) {
		t.Fatalf("saved policy evaluation did not apply requireDigest policy: %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/security/scans", strings.NewReader(`{"subjectType":"artifact","reference":"registry.example.invalid/demo/app:1.0.0","environmentId":"prod"}`))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("security scan status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"policyId":"policy-digest"`) || !strings.Contains(rec.Body.String(), `"decision":"deny"`) {
		t.Fatalf("security scan did not apply attached policy: %s", rec.Body.String())
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

	req = httptest.NewRequest(http.MethodPost, "/api/v1/policies/policy-digest/evaluate", strings.NewReader(`{"subjectType":"artifact","subjectId":"registry.example.invalid/demo/app:1.0.0","reference":"registry.example.invalid/demo/app:1.0.0"}`))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("disabled evaluate status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"code":"policy_disabled"`) {
		t.Fatalf("disabled policy evaluation should be structured: %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/security/scans", strings.NewReader(`{"subjectType":"artifact","reference":"registry.example.invalid/demo/app:1.0.0","policyId":"policy-digest"}`))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("disabled policy scan status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"code":"policy_disabled"`) {
		t.Fatalf("disabled policy scan should be structured: %s", rec.Body.String())
	}
}
