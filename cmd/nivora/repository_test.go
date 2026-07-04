package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
