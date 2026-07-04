package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRootCommandIncludesSystemCommand(t *testing.T) {
	cmd := newRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("root help failed: %v", err)
	}
	if !strings.Contains(out.String(), "system") {
		t.Fatalf("root help missing system command: %s", out.String())
	}
}

func TestSystemCommandIncludesReadOnlyDiagnostics(t *testing.T) {
	cmd := newSystemCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("system help failed: %v", err)
	}
	help := out.String()
	for _, want := range []string{"info", "runtime", "diagnostics"} {
		if !strings.Contains(help, want) {
			t.Fatalf("system help missing %q: %s", want, help)
		}
	}
}

func TestSystemInspectCommandsUseServerRoutesWithToken(t *testing.T) {
	t.Setenv("NIVORA_TEST_TOKEN", "system-token")
	tests := []struct {
		name string
		path string
		body string
	}{
		{name: "info", path: "/api/v1/system/info", body: `{"app":"nivora-server","runtime_mode":"postgres"}`},
		{name: "runtime", path: "/api/v1/system/runtime", body: `{"runtime_mode":"postgres","telemetry":{"metrics_endpoint":"/metrics"}}`},
		{name: "diagnostics", path: "/api/v1/system/diagnostics", body: `{"runtime":{"runtime_mode":"postgres"},"checks":[]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var called bool
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				if r.Method != http.MethodGet || r.URL.Path != tt.path {
					t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
				}
				if got := r.Header.Get("Authorization"); got != "Bearer system-token" {
					t.Fatalf("unexpected authorization header %q", got)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			cmd := newSystemCommand()
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&out)
			cmd.SetArgs([]string{tt.name, "--server", server.URL, "--token-env", "NIVORA_TEST_TOKEN"})
			if err := cmd.Execute(); err != nil {
				t.Fatalf("system %s failed: %v output=%s", tt.name, err, out.String())
			}
			if !called {
				t.Fatalf("expected system %s to call server", tt.name)
			}
			if !strings.Contains(out.String(), "postgres") {
				t.Fatalf("system %s output missing payload: %s", tt.name, out.String())
			}
		})
	}
}
