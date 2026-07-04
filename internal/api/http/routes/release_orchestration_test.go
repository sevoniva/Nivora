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

func TestReleasePlanResolvesCatalogTargetID(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	router := newTestRouter(cfg)

	org := postCatalogResource(t, router, "/api/v1/orgs", `{"name":"Platform"}`, http.StatusCreated)
	project := postCatalogResource(t, router, "/api/v1/projects", `{"orgId":"`+stringField(t, org, "id")+`","name":"Delivery"}`, http.StatusCreated)
	environment := postCatalogResource(t, router, "/api/v1/environments", `{"projectId":"`+stringField(t, project, "id")+`","name":"Production"}`, http.StatusCreated)
	target := postCatalogResource(t, router, "/api/v1/release-targets", `{"environmentId":"`+stringField(t, environment, "id")+`","name":"catalog-audit","targetType":"noop"}`, http.StatusCreated)
	targetID := stringField(t, target, "id")

	releaseBody := `{
	  "apiVersion":"nivora.io/v1alpha1",
	  "kind":"Release",
	  "metadata":{"name":"catalog-target-release"},
	  "spec":{
	    "version":"1.0.0",
	    "application":"api-demo",
	    "environment":"Production",
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

	body := `{
	  "apiVersion":"nivora.io/v1alpha1",
	  "kind":"ReleaseOrchestration",
	  "metadata":{"name":"catalog-target-orchestration"},
	  "spec":{
	    "environment":"Production",
	    "strategy":"plan-only",
	    "targets":[{"targetId":"` + targetID + `","order":1}]
	  }
	}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/releases/"+created.Release.ID+"/plan", strings.NewReader(body))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("plan status = %d body = %s", rec.Code, rec.Body.String())
	}
	var planned struct {
		Plan struct {
			EnvironmentID   string `json:"environmentId"`
			EnvironmentName string `json:"environmentName"`
			Targets         []struct {
				ID            string `json:"id"`
				ProjectID     string `json:"projectId"`
				EnvironmentID string `json:"environmentId"`
				Name          string `json:"name"`
				TargetType    string `json:"targetType"`
			} `json:"targets"`
			DeploymentPlans []struct {
				TargetType string `json:"targetType"`
			} `json:"deploymentPlans"`
		} `json:"plan"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &planned); err != nil {
		t.Fatalf("decode plan: %v", err)
	}
	if planned.Plan.EnvironmentID != stringField(t, environment, "id") || planned.Plan.EnvironmentName != "Production" {
		t.Fatalf("plan environment = %#v", planned.Plan)
	}
	if len(planned.Plan.Targets) != 1 {
		t.Fatalf("planned targets = %#v", planned.Plan.Targets)
	}
	resolved := planned.Plan.Targets[0]
	if resolved.ID != targetID || resolved.Name != "catalog-audit" || resolved.TargetType != "noop" ||
		resolved.ProjectID != stringField(t, project, "id") || resolved.EnvironmentID != stringField(t, environment, "id") {
		t.Fatalf("resolved target = %#v", resolved)
	}
	if len(planned.Plan.DeploymentPlans) != 1 || planned.Plan.DeploymentPlans[0].TargetType != "noop" {
		t.Fatalf("deployment plans = %#v", planned.Plan.DeploymentPlans)
	}
}
