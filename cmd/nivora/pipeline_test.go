package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPipelineDefinitionCommandIncludesRunAndVersions(t *testing.T) {
	cmd := newPipelineCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"definition", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("pipeline definition help failed: %v", err)
	}
	help := out.String()
	for _, command := range []string{"run", "versions", "rollback"} {
		if !strings.Contains(help, command) {
			t.Fatalf("pipeline definition help missing %s: %s", command, help)
		}
	}
}

func TestPipelineRunHelpMentionsCatalogMode(t *testing.T) {
	cmd := newPipelineCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"run", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("pipeline run help failed: %v", err)
	}
	help := out.String()
	for _, text := range []string{"--local=false", "--token-env", "--environment-id"} {
		if !strings.Contains(help, text) {
			t.Fatalf("pipeline run help missing %s: %s", text, help)
		}
	}
}

func TestRunPipelineAgainstServerPassesEnvironmentIDForFile(t *testing.T) {
	pipelineFile := filepath.Join(t.TempDir(), "pipeline.yaml")
	if err := os.WriteFile(pipelineFile, []byte(`apiVersion: nivora.io/v1alpha1
kind: Pipeline
metadata:
  name: env-file
spec:
  stages:
    - name: build
      jobs:
        - name: echo
          executor: shell
          steps:
            - name: say
              run: printf env-file
`), 0o600); err != nil {
		t.Fatalf("write pipeline file: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/pipeline-runs" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("environmentId"); got != "env-prod" {
			t.Fatalf("environmentId query = %q", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer token" {
			t.Fatalf("Authorization header = %q", got)
		}
		_, _ = w.Write([]byte(`{"run":{"id":"prun-env","status":"Succeeded"},"logs":[]}`))
	}))
	defer server.Close()

	if _, err := runPipelineAgainstServer(context.Background(), server.URL, pipelineFile, "token", "env-prod"); err != nil {
		t.Fatalf("run pipeline against server: %v", err)
	}
}

func TestRunPipelineAgainstServerPassesEnvironmentIDForCatalogID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/pipelines/pipe-1/runs" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("environmentId"); got != "env-prod" {
			t.Fatalf("environmentId query = %q", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer token" {
			t.Fatalf("Authorization header = %q", got)
		}
		_, _ = w.Write([]byte(`{"run":{"id":"prun-env","status":"Succeeded"},"logs":[]}`))
	}))
	defer server.Close()

	if _, err := runPipelineAgainstServer(context.Background(), server.URL, "pipe-1", "token", "env-prod"); err != nil {
		t.Fatalf("run catalog pipeline against server: %v", err)
	}
}

func TestPipelineDefinitionRunRejectsInvalidVersion(t *testing.T) {
	cmd := newPipelineDefinitionRunCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"pipe-1", "--version", "0"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "--version must be greater than zero") {
		t.Fatalf("expected invalid version error, got err=%v output=%s", err, out.String())
	}
}

func TestPipelineDefinitionRollbackUsesBearerToken(t *testing.T) {
	t.Setenv("NIVORA_TEST_TOKEN", "rollback-token")
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/pipelines/pipe-1/rollback" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer rollback-token" {
			t.Fatalf("Authorization header = %q", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		raw := string(body)
		if !strings.Contains(raw, `"version":1`) || !strings.Contains(raw, `"description":"stable"`) {
			t.Fatalf("unexpected rollback body: %s", raw)
		}
		_, _ = w.Write([]byte(`{"pipeline":{"id":"pipe-1"},"version":{"id":"pver-3","version":3}}`))
	}))
	defer server.Close()

	cmd := newPipelineDefinitionRollbackCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"pipe-1", "--version", "1", "--description", "stable", "--server", server.URL, "--token-env", "NIVORA_TEST_TOKEN"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("rollback command failed: %v output=%s", err, out.String())
	}
	if !called {
		t.Fatal("expected rollback command to call server")
	}
	if !strings.Contains(out.String(), `"pver-3"`) {
		t.Fatalf("rollback output missing response: %s", out.String())
	}
}

func TestPipelineDefinitionRollbackRejectsInvalidVersion(t *testing.T) {
	cmd := newPipelineDefinitionRollbackCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"pipe-1", "--version", "0"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "--version must be greater than zero") {
		t.Fatalf("expected invalid rollback version error, got err=%v output=%s", err, out.String())
	}
}

func TestPipelineExplainFailureCommandAggregatesReadModels(t *testing.T) {
	t.Setenv("NIVORA_TEST_TOKEN", "explain-token")
	responses := map[string]string{
		"/api/v1/pipeline-runs/prun-1":             `{"run":{"id":"prun-1","status":"Failed","failureReason":"step exited 1"}}`,
		"/api/v1/pipeline-runs/prun-1/jobs":        `[{"id":"job-1","status":"Failed","name":"build"}]`,
		"/api/v1/pipeline-runs/prun-1/steps":       `[{"id":"step-1","jobRunId":"job-1","status":"Failed","name":"test"}]`,
		"/api/v1/pipeline-runs/prun-1/timeline":    `{"timeline":[{"id":"evt-1","message":"created"},{"id":"evt-2","message":"failed"}]}`,
		"/api/v1/pipeline-runs/prun-1/logs":        `{"logs":[{"sequence":1,"stream":"stdout","content":"ok"},{"sequence":2,"stream":"stderr","content":"authorization Bearer raw-token-value"}]}`,
		"/api/v1/pipeline-runs/prun-1/annotations": `[{"level":"error","title":"test failed","message":"assertion failed"}]`,
		"/api/v1/pipeline-runs/prun-1/summary":     `{"pipelineRunId":"prun-1","annotations":1}`,
	}
	called := map[string]bool{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer explain-token" {
			t.Fatalf("Authorization header = %q", got)
		}
		body, ok := responses[r.URL.Path]
		if !ok {
			t.Fatalf("unexpected request path %s", r.URL.Path)
		}
		called[r.URL.Path] = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	cmd := newPipelineExplainFailureCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"prun-1", "--server", server.URL, "--token-env", "NIVORA_TEST_TOKEN", "--log-limit", "1", "--timeline-limit", "1"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("explain-failure failed: %v output=%s", err, out.String())
	}
	for path := range responses {
		if !called[path] {
			t.Fatalf("expected request to %s", path)
		}
	}
	body := out.String()
	for _, want := range []string{`"mutated": false`, `"failedJobs"`, `"failedSteps"`, `"recentTimeline"`, `"logPreview"`, "[REDACTED]"} {
		if !strings.Contains(body, want) {
			t.Fatalf("explanation missing %q: %s", want, body)
		}
	}
	for _, forbidden := range []string{"raw-token-value", "authorization Bearer"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("explanation leaked %q: %s", forbidden, body)
		}
	}
	var decoded map[string]any
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("explanation is not JSON: %v body=%s", err, body)
	}
	if decoded["pipelineRunId"] != "prun-1" {
		t.Fatalf("pipelineRunId = %#v", decoded["pipelineRunId"])
	}
}
