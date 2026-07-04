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
			ID             string `json:"id"`
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

	updateBody := `{"definition":{"apiVersion":"nivora.io/v1alpha1","kind":"Pipeline","metadata":{"name":"build-v2"},"spec":{"stages":[{"name":"build","jobs":[{"name":"test","executor":"shell","steps":[{"name":"echo","run":"printf v2"}]}]}]}}}`
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/pipelines/"+created.Pipeline.ID, strings.NewReader(updateBody))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("update pipeline definition status = %d body = %s", rec.Code, rec.Body.String())
	}
	var updated struct {
		Version struct {
			ID             string `json:"id"`
			Version        int    `json:"version"`
			DefinitionHash string `json:"definitionHash"`
		} `json:"version"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode updated definition: %v", err)
	}
	if updated.Version.Version != 2 || updated.Version.DefinitionHash == created.Version.DefinitionHash {
		t.Fatalf("unexpected update response: %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/pipelines/"+created.Pipeline.ID+"/versions", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), created.Version.DefinitionHash) || !strings.Contains(rec.Body.String(), updated.Version.DefinitionHash) || strings.Contains(rec.Body.String(), `"historyComplete":false`) {
		t.Fatalf("versions status = %d body = %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/pipelines/"+created.Pipeline.ID+"/runs?version=1&environmentId=env-prod", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("run first definition version status = %d body = %s", rec.Code, rec.Body.String())
	}
	var firstRunResponse struct {
		Pipeline struct {
			EnvironmentID string            `json:"environmentId"`
			Labels        map[string]string `json:"labels"`
			Metadata      map[string]string `json:"metadata"`
		} `json:"pipeline"`
		Run struct {
			PipelineID        string `json:"pipelineId"`
			PipelineVersionID string `json:"pipelineVersionId"`
			Status            string `json:"status"`
		} `json:"run"`
		Logs []struct {
			Content string `json:"content"`
		} `json:"logs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &firstRunResponse); err != nil {
		t.Fatalf("decode first version run response: %v", err)
	}
	if firstRunResponse.Run.PipelineID != created.Pipeline.ID || firstRunResponse.Run.PipelineVersionID != created.Version.ID || firstRunResponse.Run.Status != "Succeeded" {
		t.Fatalf("first version run did not preserve catalog identity: %s", rec.Body.String())
	}
	if firstRunResponse.Pipeline.EnvironmentID != "env-prod" || firstRunResponse.Pipeline.Labels["environmentId"] != "env-prod" || firstRunResponse.Pipeline.Metadata["environmentId"] != "env-prod" {
		t.Fatalf("first version run did not preserve environment ownership: %s", rec.Body.String())
	}
	if len(firstRunResponse.Logs) == 0 || !strings.Contains(firstRunResponse.Logs[0].Content, "ok") {
		t.Fatalf("first version run did not use version 1 definition: %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/pipelines/"+created.Pipeline.ID+"/runs?version=abc", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "invalid_pipeline_version") {
		t.Fatalf("invalid version status = %d body = %s", rec.Code, rec.Body.String())
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
	if runResponse.Run.PipelineID != created.Pipeline.ID || runResponse.Run.PipelineVersionID != updated.Version.ID || runResponse.Run.Status != "Succeeded" {
		t.Fatalf("run did not preserve catalog identity: %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/pipelines/"+created.Pipeline.ID+"/rollback", strings.NewReader(`{"version":1,"description":"restored stable"}`))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("rollback pipeline definition status = %d body = %s", rec.Code, rec.Body.String())
	}
	var rolledBack struct {
		Pipeline struct {
			Description string `json:"description"`
		} `json:"pipeline"`
		Version struct {
			ID             string `json:"id"`
			Version        int    `json:"version"`
			DefinitionHash string `json:"definitionHash"`
		} `json:"version"`
		Definition struct {
			Metadata struct {
				Name string `json:"name"`
			} `json:"metadata"`
		} `json:"definition"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &rolledBack); err != nil {
		t.Fatalf("decode rollback response: %v", err)
	}
	if rolledBack.Version.Version != 3 || rolledBack.Version.DefinitionHash != created.Version.DefinitionHash || rolledBack.Definition.Metadata.Name != "build" || rolledBack.Pipeline.Description != "restored stable" {
		t.Fatalf("unexpected rollback response: %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/pipelines/"+created.Pipeline.ID+"/runs", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("run rolled back definition status = %d body = %s", rec.Code, rec.Body.String())
	}
	var rollbackRunResponse struct {
		Run struct {
			PipelineVersionID string `json:"pipelineVersionId"`
			Status            string `json:"status"`
		} `json:"run"`
		Logs []struct {
			Content string `json:"content"`
		} `json:"logs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &rollbackRunResponse); err != nil {
		t.Fatalf("decode rollback run response: %v", err)
	}
	if rollbackRunResponse.Run.PipelineVersionID != rolledBack.Version.ID || rollbackRunResponse.Run.Status != "Succeeded" {
		t.Fatalf("rolled back current run did not use new version: %s", rec.Body.String())
	}
	if len(rollbackRunResponse.Logs) == 0 || !strings.Contains(rollbackRunResponse.Logs[0].Content, "ok") {
		t.Fatalf("rolled back current run did not restore version 1 definition: %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/pipelines/"+created.Pipeline.ID+"/rollback", strings.NewReader(`{"version":3}`))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "older than current") {
		t.Fatalf("rollback current version status = %d body = %s", rec.Code, rec.Body.String())
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
