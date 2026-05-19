package routes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/sevoniva/nivora/internal/infra/config"
)

func TestDeploymentRoutes(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	router := newTestRouter(cfg)
	body := deploymentRequestBody(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/deployments/plan", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("plan status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"manifestCount":1`)) {
		t.Fatalf("plan body = %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/deployments", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d body = %s", rec.Code, rec.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	run := created["run"].(map[string]any)
	runID := run["id"].(string)
	if run["status"] != "Succeeded" {
		t.Fatalf("status = %v", run["status"])
	}

	for _, path := range []string{
		"/api/v1/deployments/" + runID,
		"/api/v1/deployments/" + runID + "/plan",
		"/api/v1/deployments/" + runID + "/resources",
		"/api/v1/deployments/" + runID + "/health",
		"/api/v1/deployments/" + runID + "/diff",
		"/api/v1/deployments/" + runID + "/manifest-snapshot",
		"/api/v1/deployments/" + runID + "/rollback-plan",
		"/api/v1/deployments/" + runID + "/logs",
		"/api/v1/deployments/" + runID + "/events",
		"/api/v1/deployments/" + runID + "/timeline",
	} {
		req = httptest.NewRequest(http.MethodGet, path, nil)
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status = %d body = %s", path, rec.Code, rec.Body.String())
		}
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/deployments/"+runID+"/cancel", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("cancel terminal status = %d body = %s", rec.Code, rec.Body.String())
	}

	applyBody, err := json.Marshal(map[string]any{"definition": deploymentRequest(t), "confirm": false})
	if err != nil {
		t.Fatalf("marshal apply request: %v", err)
	}
	req = httptest.NewRequest(http.MethodPost, "/api/v1/deployments/apply", bytes.NewReader(applyBody))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unconfirmed apply status = %d body = %s", rec.Code, rec.Body.String())
	}

	applyBody, err = json.Marshal(map[string]any{"definition": deploymentRequest(t), "confirm": true})
	if err != nil {
		t.Fatalf("marshal apply request: %v", err)
	}
	req = httptest.NewRequest(http.MethodPost, "/api/v1/deployments/apply", bytes.NewReader(applyBody))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("confirmed apply status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"apply"`)) {
		t.Fatalf("apply response = %s", rec.Body.String())
	}
}

func TestDeploymentRollbackRouteRequiresConfirmation(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	router := newTestRouter(cfg)
	def := deploymentRequest(t)
	def["spec"].(map[string]any)["options"] = map[string]any{"dryRun": false, "apply": true}
	body, err := json.Marshal(map[string]any{"definition": def, "confirm": true})
	if err != nil {
		t.Fatalf("marshal apply request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/deployments/apply", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("apply status = %d body = %s", rec.Code, rec.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode apply response: %v", err)
	}
	runID := created["run"].(map[string]any)["id"].(string)

	req = httptest.NewRequest(http.MethodPost, "/api/v1/deployments/"+runID+"/rollback", bytes.NewReader([]byte(`{"confirm":false}`)))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unconfirmed rollback status = %d body = %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/deployments/"+runID+"/rollback", bytes.NewReader([]byte(`{"confirm":true}`)))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("confirmed rollback status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"status":"RolledBack"`)) {
		t.Fatalf("rollback response = %s", rec.Body.String())
	}
}

func TestGitOpsDeploymentRoutes(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	router := newTestRouter(cfg)
	body := gitOpsDeploymentRequestBody(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/deployments/gitops/plan", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("gitops plan status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"applicationName":"demo-springboot"`)) {
		t.Fatalf("gitops plan body = %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/deployments/gitops", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("gitops create status = %d body = %s", rec.Code, rec.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	run := created["run"].(map[string]any)
	runID := run["id"].(string)
	for _, path := range []string{"/api/v1/deployments/" + runID + "/gitops-plan", "/api/v1/deployments/" + runID + "/argocd-status", "/api/v1/deployments/" + runID + "/diff", "/api/v1/deployments/" + runID + "/timeline", "/api/v1/integrations/argocd/applications/demo-springboot/status", "/api/v1/integrations/argocd/applications/demo-springboot/resources"} {
		req = httptest.NewRequest(http.MethodGet, path, nil)
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status = %d body = %s", path, rec.Code, rec.Body.String())
		}
	}
	req = httptest.NewRequest(http.MethodPost, "/api/v1/integrations/argocd/applications/demo-springboot/sync", bytes.NewReader([]byte(`{"applicationName":"demo-springboot","allowSync":false,"confirmed":false}`)))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unguarded sync status = %d body = %s", rec.Code, rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodPost, "/api/v1/integrations/argocd/applications/demo-springboot/sync", bytes.NewReader([]byte(`{"applicationName":"demo-springboot","allowSync":true,"confirmed":true}`)))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("guarded sync status = %d body = %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/deployments/gitops/commit", bytes.NewReader([]byte(`{"confirm":false}`)))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unguarded gitops commit status = %d body = %s", rec.Code, rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodPost, "/api/v1/deployments/gitops/rollback", bytes.NewReader([]byte(`{"confirm":false}`)))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unguarded gitops rollback status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func TestHostDeploymentRoutes(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	router := newTestRouter(cfg)
	body := hostDeploymentRequestBody(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/deployments/host/plan", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("host plan status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"groupName":"local-host-group"`)) {
		t.Fatalf("host plan body = %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/deployments", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("host create status = %d body = %s", rec.Code, rec.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode host response: %v", err)
	}
	run := created["run"].(map[string]any)
	runID := run["id"].(string)
	for _, path := range []string{"/api/v1/deployments/" + runID + "/hosts", "/api/v1/deployments/" + runID + "/rollback-plan", "/api/v1/deployments/" + runID + "/timeline"} {
		req = httptest.NewRequest(http.MethodGet, path, nil)
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status = %d body = %s", path, rec.Code, rec.Body.String())
		}
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/host-groups", bytes.NewReader([]byte(`{"name":"local-host-group","environmentId":"dev","hosts":[{"name":"local-noop-host","address":"127.0.0.1"}]}`)))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("create host group status = %d body = %s", rec.Code, rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/host-groups", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list host groups status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func gitOpsDeploymentRequestBody(t *testing.T) []byte {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"apiVersion": "nivora.io/v1alpha1",
		"kind":       "Deployment",
		"metadata": map[string]any{
			"name": "demo-gitops",
		},
		"spec": map[string]any{
			"application": "demo-springboot",
			"environment": "dev",
			"target": map[string]any{
				"type":            "argocd",
				"name":            "demo-argocd",
				"applicationName": "demo-springboot",
				"repoURL":         "https://example.com/gitops/demo.git",
				"path":            "apps/demo-springboot/dev",
				"revision":        "main",
			},
			"artifacts": []map[string]any{{
				"name":      "demo-springboot",
				"type":      "image",
				"reference": "registry.example.com/demo/demo-springboot@sha256:example",
			}},
			"gitops": map[string]any{
				"mode":               "plan",
				"writeToWorkingTree": false,
				"sync":               false,
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	return body
}

func hostDeploymentRequestBody(t *testing.T) []byte {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"apiVersion": "nivora.io/v1alpha1",
		"kind":       "Deployment",
		"metadata": map[string]any{
			"name": "demo-host-release",
		},
		"spec": map[string]any{
			"application": "demo",
			"environment": "dev",
			"target": map[string]any{
				"type": "host",
				"name": "local-host-group",
			},
			"artifact": map[string]any{
				"name":      "demo",
				"type":      "binary",
				"reference": "./dist/demo.tar.gz",
			},
			"host": map[string]any{
				"deployPath":  "/opt/nivora/apps/demo",
				"serviceName": "demo",
				"strategy":    "symlink",
				"healthCheck": "http://localhost:8080/healthz",
				"hosts": []map[string]any{{
					"id":      "local-noop-host",
					"name":    "local-noop-host",
					"address": "127.0.0.1",
				}},
			},
			"options": map[string]any{
				"dryRun": true,
				"apply":  false,
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal host request: %v", err)
	}
	return body
}

func deploymentRequestBody(t *testing.T) []byte {
	t.Helper()
	body, err := json.Marshal(deploymentRequest(t))
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	return body
}

func deploymentRequest(t *testing.T) map[string]any {
	t.Helper()
	dir := t.TempDir()
	manifest := filepath.Join(dir, "deployment.yaml")
	if err := os.WriteFile(manifest, []byte(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: demo
`), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	return map[string]any{
		"apiVersion": "nivora.io/v1alpha1",
		"kind":       "Deployment",
		"metadata": map[string]any{
			"name": "demo-yaml",
		},
		"spec": map[string]any{
			"application": "demo-app",
			"environment": "dev",
			"target": map[string]any{
				"type":      "kubernetes-yaml",
				"name":      "dev-kind",
				"namespace": "default",
			},
			"artifacts": []map[string]any{{
				"name":      "demo-app",
				"type":      "image",
				"reference": "example.local/demo:dev",
			}},
			"manifests": []string{manifest},
			"options": map[string]any{
				"dryRun": true,
				"apply":  false,
			},
		},
	}
}
