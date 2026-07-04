package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	domainauth "github.com/sevoniva/nivora/internal/domain/auth"
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

func TestSecurityRoutesRespectTenantScope(t *testing.T) {
	router, authService := newIsoRouter(t)
	tokenA := createScopedToken(t, authService, "security-project-a", domainauth.RoleAdmin, "project", "project-a")
	tokenB := createScopedToken(t, authService, "security-project-b", domainauth.RoleAdmin, "project", "project-b")

	scanA := createScopedSecurityScan(t, router, tokenA, "manifest-project-a")
	scanB := createScopedSecurityScan(t, router, tokenB, "manifest-project-b")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/security/scans", nil)
	req.Header.Set("Authorization", "Bearer "+tokenA)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list scoped security scans status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), scanA.ID) || strings.Contains(rec.Body.String(), scanB.ID) || strings.Contains(rec.Body.String(), "manifest-project-b") {
		t.Fatalf("list scoped security scans leaked cross-project data: %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/security/findings?scanId="+scanB.ID, nil)
	req.Header.Set("Authorization", "Bearer "+tokenA)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), "Privileged container requested") || strings.Contains(rec.Body.String(), scanB.ID) {
		t.Fatalf("cross-project findings query returned data: status=%d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/security/scans/"+scanB.ID, nil)
	req.Header.Set("Authorization", "Bearer "+tokenA)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("cross-project scan get status = %d body = %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/security/scans/"+scanB.ID+"/findings", nil)
	req.Header.Set("Authorization", "Bearer "+tokenA)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("cross-project scan findings get status = %d body = %s", rec.Code, rec.Body.String())
	}
}

type scopedSecurityScan struct {
	ID        string `json:"id"`
	ProjectID string `json:"projectId"`
}

func createScopedSecurityScan(t *testing.T, router http.Handler, token string, subjectID string) scopedSecurityScan {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/security/scans", strings.NewReader(`{"subjectType":"manifest","subjectId":"`+subjectID+`","content":"securityContext:\n  privileged: true\n"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create scoped security scan status = %d body = %s", rec.Code, rec.Body.String())
	}
	var created struct {
		Scan scopedSecurityScan `json:"scan"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode scoped security scan: %v", err)
	}
	if created.Scan.ID == "" || created.Scan.ProjectID == "" {
		t.Fatalf("scoped security scan missing id or project id: %s", rec.Body.String())
	}
	return created.Scan
}
