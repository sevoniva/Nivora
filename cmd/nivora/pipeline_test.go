package main

import (
	"bytes"
	"context"
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
	for _, command := range []string{"run", "versions"} {
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
