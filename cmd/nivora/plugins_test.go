package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPluginsCommandIncludesCapabilities(t *testing.T) {
	cmd := newPluginsCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("plugins help failed: %v", err)
	}
	for _, want := range []string{"list", "inspect", "capabilities", "validate"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("plugins help missing %q: %s", want, out.String())
		}
	}
}

func TestPluginsCapabilitiesLocalUsesBuiltInRegistry(t *testing.T) {
	cmd := newPluginsCapabilitiesCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"artifact-oci", "--local"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("plugins capabilities local failed: %v output=%s", err, out.String())
	}
	for _, want := range []string{"artifact.inspect", "artifact.resolve_digest"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("local capabilities output missing %q: %s", want, out.String())
		}
	}
}

func TestPluginsCapabilitiesUsesServerRoute(t *testing.T) {
	var called bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/plugins/artifact-oci/capabilities" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"name":"artifact.resolve_digest","description":"resolve digests"}]`))
	}))
	defer server.Close()

	cmd := newPluginsCapabilitiesCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"artifact-oci", "--server", server.URL})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("plugins capabilities failed: %v output=%s", err, out.String())
	}
	if !called {
		t.Fatal("expected plugins capabilities to call server")
	}
	if !strings.Contains(out.String(), "artifact.resolve_digest") {
		t.Fatalf("server capabilities output missing payload: %s", out.String())
	}
}
