package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRepositoryCommandHelpIncludesValidate(t *testing.T) {
	cmd := newRepositoryCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("repository help failed: %v", err)
	}
	if !strings.Contains(out.String(), "validate") {
		t.Fatalf("repository help missing validate command: %s", out.String())
	}
}

func TestRepositoryCreateAndUpdateHelpDescribeCredentialRefBoundary(t *testing.T) {
	for _, tc := range []struct {
		name string
		cmd  *cobra.Command
	}{
		{name: "create", cmd: newRepositoryCreateCommand()},
		{name: "update", cmd: newRepositoryUpdateCommand()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			tc.cmd.SetOut(&out)
			tc.cmd.SetErr(&out)
			tc.cmd.SetArgs([]string{"--help"})
			if err := tc.cmd.Execute(); err != nil {
				t.Fatalf("repository %s help failed: %v", tc.name, err)
			}
			help := out.String()
			for _, want := range []string{"--credential-ref", "no secret value"} {
				if !strings.Contains(help, want) {
					t.Fatalf("repository %s help missing %q: %s", tc.name, want, help)
				}
			}
		})
	}
}

func TestRepositoryValidateUsesBearerToken(t *testing.T) {
	t.Setenv("NIVORA_TEST_TOKEN", "repo-token")
	var called bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/repositories/repo-1/validate" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer repo-token" {
			t.Fatalf("Authorization header = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"valid":true,"repositoryId":"repo-1"}`))
	}))
	defer server.Close()

	cmd := newRepositoryCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"validate", "repo-1", "--server", server.URL, "--token-env", "NIVORA_TEST_TOKEN"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("repository validate failed: %v output=%s", err, out.String())
	}
	if !called {
		t.Fatal("repository validate did not call server")
	}
	if !strings.Contains(out.String(), `"valid": true`) {
		t.Fatalf("repository validate output = %s", out.String())
	}
}

func TestRepositoryCreateFromFileUsesBearerTokenAndDefinitionBody(t *testing.T) {
	t.Setenv("NIVORA_TEST_TOKEN", "repo-token")
	definition := filepath.Join(t.TempDir(), "repository.yaml")
	if err := os.WriteFile(definition, []byte(`apiVersion: nivora.io/v1alpha1
kind: Repository
metadata:
  id: repo-file
  name: platform-config
  labels:
    team: platform
spec:
  provider: generic_git
  url: https://example.invalid/platform/config.git
  defaultBranch: main
  projectId: project-a
  credentialRef: cred-repository-readonly
  metadata:
    purpose: gitops
`), 0o600); err != nil {
		t.Fatalf("write repository definition: %v", err)
	}

	var called bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/repositories" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer repo-token" {
			t.Fatalf("Authorization header = %q", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		for key, want := range map[string]string{
			"id":            "repo-file",
			"projectId":     "project-a",
			"name":          "platform-config",
			"url":           "https://example.invalid/platform/config.git",
			"provider":      "generic_git",
			"defaultBranch": "main",
			"credentialRef": "cred-repository-readonly",
		} {
			if got, _ := body[key].(string); got != want {
				t.Fatalf("body[%s] = %q, want %q in %#v", key, got, want, body)
			}
		}
		labels, ok := body["labels"].(map[string]any)
		if !ok || labels["team"] != "platform" {
			t.Fatalf("labels not preserved: %#v", body["labels"])
		}
		metadata, ok := body["metadata"].(map[string]any)
		if !ok || metadata["purpose"] != "gitops" {
			t.Fatalf("metadata not preserved: %#v", body["metadata"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"repo-file","provider":"generic"}`))
	}))
	defer server.Close()

	cmd := newRepositoryCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"create", "--file", definition, "--server", server.URL, "--token-env", "NIVORA_TEST_TOKEN"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("repository create --file failed: %v output=%s", err, out.String())
	}
	if !called {
		t.Fatal("repository create --file did not call server")
	}
	if !strings.Contains(out.String(), `"id": "repo-file"`) {
		t.Fatalf("repository create output = %s", out.String())
	}
}
