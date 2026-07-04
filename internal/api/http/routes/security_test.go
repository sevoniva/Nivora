package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sevoniva/nivora/internal/infra/config"
)

func TestSecurityRoutes(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	router := newTestRouter(cfg)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/security/scans", strings.NewReader(`{"subjectType":"artifact","subjectId":"demo","reference":"demo:latest"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("scan status = %d body = %s", rec.Code, rec.Body.String())
	}
	var created struct {
		Scan struct {
			ID string `json:"id"`
		} `json:"scan"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode scan: %v", err)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/policies/evaluate", strings.NewReader(`{"subjectType":"artifact","subjectId":"demo","reference":"demo:latest"}`))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("evaluate status = %d body = %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/security/scans", strings.NewReader(`{"subjectType":"manifest","subjectId":"manifest-demo","content":"securityContext:\n  privileged: true\n"}`))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("manifest scan status = %d body = %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/security/scans?subjectType=manifest&limit=1", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"scans"`) || !strings.Contains(rec.Body.String(), `"pagination"`) {
		t.Fatalf("list scans status = %d body = %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/security/findings?severity=High", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"findings"`) || !strings.Contains(rec.Body.String(), `"Privileged container requested"`) {
		t.Fatalf("list findings status = %d body = %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/security/scans/"+created.Scan.ID+"/findings", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("scan findings status = %d body = %s", rec.Code, rec.Body.String())
	}
}
