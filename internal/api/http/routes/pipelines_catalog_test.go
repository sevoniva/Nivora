package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sevoniva/nivora/internal/infra/config"
)

func TestPipelineDefinitionCatalogRoutes(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	router := newTestRouter(cfg)
	body := `{"projectId":"project-a","definition":{"apiVersion":"nivora.io/v1alpha1","kind":"Pipeline","metadata":{"name":"build"},"spec":{"stages":[{"name":"build","jobs":[{"name":"test","executor":"shell","steps":[{"name":"echo","run":"printf ok"}]}]}]}}}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/pipelines", strings.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create pipeline definition status = %d body = %s", rec.Code, rec.Body.String())
	}
	var created struct {
		Pipeline struct {
			ID      string `json:"id"`
			Enabled bool   `json:"enabled"`
		} `json:"pipeline"`
		Version struct {
			Version        int    `json:"version"`
			DefinitionHash string `json:"definitionHash"`
		} `json:"version"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode pipeline definition: %v", err)
	}
	if created.Pipeline.ID == "" || !created.Pipeline.Enabled || created.Version.Version != 1 || created.Version.DefinitionHash == "" {
		t.Fatalf("unexpected create response: %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/pipelines/"+created.Pipeline.ID+"/versions", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), created.Version.DefinitionHash) {
		t.Fatalf("versions status = %d body = %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/pipelines/"+created.Pipeline.ID+"/runs", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("run from definition status = %d body = %s", rec.Code, rec.Body.String())
	}
	var runResponse struct {
		Run struct {
			PipelineID        string `json:"pipelineId"`
			PipelineVersionID string `json:"pipelineVersionId"`
			Status            string `json:"status"`
		} `json:"run"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &runResponse); err != nil {
		t.Fatalf("decode run response: %v", err)
	}
	if runResponse.Run.PipelineID != created.Pipeline.ID || runResponse.Run.PipelineVersionID == "" || runResponse.Run.Status != "Succeeded" {
		t.Fatalf("run did not preserve catalog identity: %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/pipelines?projectId=project-a", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), created.Pipeline.ID) {
		t.Fatalf("list pipeline definitions status = %d body = %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/v1/pipelines/"+created.Pipeline.ID, nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"enabled":false`) {
		t.Fatalf("disable pipeline definition status = %d body = %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/pipelines/"+created.Pipeline.ID+"/runs", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "pipeline_definition_disabled") {
		t.Fatalf("disabled definition run status = %d body = %s", rec.Code, rec.Body.String())
	}
}
