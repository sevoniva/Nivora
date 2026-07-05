package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sevoniva/nivora/internal/infra/config"
)

func TestRepositorySnapshotAndIntelligenceRoutesUseLocalStaticInspection(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	router := newTestRouter(cfg)

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.invalid/service\n\ngo 1.23\n"), 0o600); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "Dockerfile"), []byte("FROM scratch\n"), 0o600); err != nil {
		t.Fatalf("write Dockerfile: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".nivora", "workflows"), 0o700); err != nil {
		t.Fatalf("mkdir workflow dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".nivora", "workflows", "ci.yaml"), []byte("kind: Workflow\n"), 0o600); err != nil {
		t.Fatalf("write workflow: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("TOKEN=placeholder\n"), 0o600); err != nil {
		t.Fatalf("write env: %v", err)
	}

	org := postCatalogResource(t, router, "/api/v1/orgs", `{"name":"Repository Platform"}`, http.StatusCreated)
	project := postCatalogResource(t, router, "/api/v1/projects", `{"orgId":"`+stringField(t, org, "id")+`","name":"Repository Project"}`, http.StatusCreated)
	repository := postCatalogResource(t, router, "/api/v1/repositories", `{"projectId":"`+stringField(t, project, "id")+`","name":"Local Service","url":"https://example.invalid/team/local-service.git","provider":"generic","defaultBranch":"HEAD"}`, http.StatusCreated)
	repositoryID := stringField(t, repository, "id")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/repositories/"+repositoryID+"/snapshot", strings.NewReader(`{"localPath":"`+root+`","ref":"HEAD"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("snapshot status = %d body = %s", rec.Code, rec.Body.String())
	}
	for _, want := range []string{`"repositoryId":"` + repositoryID + `"`, "go", "docker", ".nivora/workflows"} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Fatalf("snapshot response missing %q: %s", want, rec.Body.String())
		}
	}
	if strings.Contains(rec.Body.String(), "TOKEN=placeholder") {
		t.Fatalf("snapshot response leaked .env content: %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/repositories/"+repositoryID+"/snapshots", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"snapshots"`) {
		t.Fatalf("snapshot list status = %d body = %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/repositories/"+repositoryID+"/intelligence", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("intelligence status = %d body = %s", rec.Code, rec.Body.String())
	}
	for _, want := range []string{"go test ./...", "go build ./...", "recommendedNivoraWorkflowDraft"} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Fatalf("intelligence response missing %q: %s", want, rec.Body.String())
		}
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/repositories/"+repositoryID+"/analyze", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "go test ./...") {
		t.Fatalf("analyze status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func TestWorkflowValidatePlanAndGuardedRunRoutes(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	router := newTestRouter(cfg)
	workflow := strings.TrimSpace(`
apiVersion: nivora.io/v1alpha1
kind: Workflow
metadata:
  name: go-ci
on: [manual, push]
jobs:
  test:
    runsOn: [self-hosted, shell]
    steps:
      - name: test
        run: go test ./...
  build:
    needs: [test]
    steps:
      - name: build
        run: go build ./...
`)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/validate", strings.NewReader(workflow))
	req.Header.Set("Content-Type", "application/yaml")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"valid":true`) || !strings.Contains(rec.Body.String(), `"conversionReady":true`) {
		t.Fatalf("validate status = %d body = %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/workflows/plan", strings.NewReader(`{"content":`+quoteJSON(workflow)+`}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"edges"`) || !strings.Contains(rec.Body.String(), `"workflowId"`) || !strings.Contains(rec.Body.String(), `"planId"`) {
		t.Fatalf("plan status = %d body = %s", rec.Code, rec.Body.String())
	}
	var plan map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &plan); err != nil {
		t.Fatalf("decode workflow plan: %v", err)
	}
	planID := stringField(t, plan, "planId")

	req = httptest.NewRequest(http.MethodGet, "/api/v1/workflows/plans/"+planID, nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"contentHash"`) || strings.Contains(rec.Body.String(), "raw-secret-value") {
		t.Fatalf("get plan status = %d body = %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/workflows/plans?workflowId=workflow-go-ci", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), planID) {
		t.Fatalf("list plans status = %d body = %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/workflows/validate", strings.NewReader(`{"content":"jobs:\n  test:\n    steps:\n      - env:\n          TOKEN: raw-secret-value\n        run: echo test\n"}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest || strings.Contains(rec.Body.String(), "raw-secret-value") {
		t.Fatalf("invalid secret workflow status = %d body = %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/workflows/run", strings.NewReader(`{}`))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotImplemented || !strings.Contains(rec.Body.String(), "not_implemented") {
		t.Fatalf("workflow run status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func quoteJSON(value string) string {
	quoted := `"` + strings.ReplaceAll(strings.ReplaceAll(value, `\`, `\\`), `"`, `\"`) + `"`
	quoted = strings.ReplaceAll(quoted, "\n", `\n`)
	return quoted
}
