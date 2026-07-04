package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRootCommandIncludesIntegrationsCommand(t *testing.T) {
	cmd := newRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("root help failed: %v", err)
	}
	if !strings.Contains(out.String(), "integrations") {
		t.Fatalf("root help missing integrations command: %s", out.String())
	}
}

func TestIntegrationsListLocalUsesBuiltInRegistry(t *testing.T) {
	cmd := newIntegrationsCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"list", "--local"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("integrations list --local failed: %v output=%s", err, out.String())
	}
	output := out.String()
	for _, want := range []string{"integrations", "warnings", "foundation", "experimental"} {
		if !strings.Contains(output, want) {
			t.Fatalf("integrations local output missing %q: %s", want, output)
		}
	}
	if strings.Contains(output, "not_implemented") {
		t.Fatalf("integrations local output still looks placeholder: %s", output)
	}
}

func TestIntegrationsListUsesServerRouteWithToken(t *testing.T) {
	t.Setenv("NIVORA_TEST_TOKEN", "integration-token")
	var called bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/integrations" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer integration-token" {
			t.Fatalf("unexpected authorization header %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"count":1,"integrations":[{"name":"executor-argocd","maturity":"experimental"}],"warnings":["metadata only"]}`))
	}))
	defer server.Close()

	cmd := newIntegrationsCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"list", "--server", server.URL, "--token-env", "NIVORA_TEST_TOKEN"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("integrations list failed: %v output=%s", err, out.String())
	}
	if !called {
		t.Fatal("expected integrations list to call server")
	}
	if !strings.Contains(out.String(), "executor-argocd") {
		t.Fatalf("integrations list output missing payload: %s", out.String())
	}
}
