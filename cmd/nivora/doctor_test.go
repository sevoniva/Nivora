package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoctorCommandChecksProductionExample(t *testing.T) {
	cmd := newDoctorCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--file", "../../configs/production.example.yaml"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("doctor command failed: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), `"status": "PASS"`) {
		t.Fatalf("doctor output = %s", out.String())
	}
}

func TestDoctorCommandFailsUnsafeConfigWithoutLeakingTokenEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "unsafe.yaml")
	body := []byte(`
environment: production
database:
  url: "postgres://nivora@postgres.example.internal:5432/nivora?sslmode=require"
  runtime_store: memory
auth:
  enabled: true
  mode: token
  static_token_env: NIVORA_SECRET_TOKEN
runtime:
  allow_local_shell_executor: true
  runner_isolation_profile: local-dev
event_bus:
  type: memory
object_store:
  type: local
log:
  level: info
`)
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := newDoctorCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--file", path})
	if err := cmd.Execute(); err == nil {
		t.Fatalf("expected unsafe config to fail: %s", out.String())
	}
	if !strings.Contains(out.String(), `"status": "FAIL"`) {
		t.Fatalf("doctor output = %s", out.String())
	}
	if strings.Contains(out.String(), "NIVORA_SECRET_TOKEN") {
		t.Fatalf("doctor leaked token-like env var: %s", out.String())
	}
}

func TestDoctorLiveUsesReadOnlyServerRoutesWithToken(t *testing.T) {
	t.Setenv("NIVORA_TEST_TOKEN", "live-token")
	seen := map[string]bool{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer live-token" {
			t.Fatalf("unexpected authorization header %q", got)
		}
		if r.Method != http.MethodGet {
			t.Fatalf("doctor live should only use GET routes, got %s", r.Method)
		}
		seen[r.URL.Path] = true
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/system/diagnostics":
			_, _ = w.Write([]byte(`{"runtime":{"runtime_mode":"postgres"},"checks":[{"name":"database","status":"ok","critical":true}]}`))
		case "/api/v1/system/runtime/recovery":
			_, _ = w.Write([]byte(`{"status":"healthy","pendingOutboxEvents":0,"failedOutboxEvents":0,"publishedOutboxEvents":2}`))
		case "/api/v1/audit/verify":
			if r.URL.Query().Get("scopeType") != "deployment" || r.URL.Query().Get("scopeId") != "project-a" {
				t.Fatalf("unexpected audit verify query: %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"valid":true,"message":"chain verified"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	cmd := newDoctorCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"live", "--server", server.URL, "--token-env", "NIVORA_TEST_TOKEN", "--audit-scope-type", "deployment", "--audit-scope-id", "project-a"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("doctor live failed: %v output=%s", err, out.String())
	}
	for _, path := range []string{"/api/v1/system/diagnostics", "/api/v1/system/runtime/recovery", "/api/v1/audit/verify"} {
		if !seen[path] {
			t.Fatalf("doctor live did not call %s; saw %#v", path, seen)
		}
	}
	body := out.String()
	for _, want := range []string{`"status": "PASS"`, `"live.runtime_mode"`, `"live.event_outbox"`, `"live.audit_hash_chain"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("doctor live output missing %s: %s", want, body)
		}
	}
	if strings.Contains(body, "live-token") {
		t.Fatalf("doctor live leaked bearer token: %s", body)
	}
}

func TestDoctorLiveFailsOnCriticalDiagnostics(t *testing.T) {
	t.Setenv("NIVORA_TEST_TOKEN", "live-token")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/system/diagnostics":
			_, _ = w.Write([]byte(`{"runtime":{"runtime_mode":"postgres"},"checks":[{"name":"database","status":"degraded","critical":true}]}`))
		case "/api/v1/system/runtime/recovery":
			_, _ = w.Write([]byte(`{"status":"healthy","pendingOutboxEvents":0,"failedOutboxEvents":0}`))
		case "/api/v1/audit/verify":
			_, _ = w.Write([]byte(`{"valid":true}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	cmd := newDoctorCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"live", "--server", server.URL, "--token-env", "NIVORA_TEST_TOKEN"})
	if err := cmd.Execute(); err == nil {
		t.Fatalf("expected doctor live to fail on critical diagnostics: %s", out.String())
	}
	if !strings.Contains(out.String(), `"status": "FAIL"`) || !strings.Contains(out.String(), "critical dependencies") {
		t.Fatalf("doctor live failure output = %s", out.String())
	}
}

func TestDoctorLiveFailureDoesNotLeakToken(t *testing.T) {
	t.Setenv("NIVORA_TEST_TOKEN", "live-token")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"code":"unauthorized","message":"missing audit.read permission"}`, http.StatusUnauthorized)
	}))
	defer server.Close()

	cmd := newDoctorCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"live", "--server", server.URL, "--token-env", "NIVORA_TEST_TOKEN"})
	if err := cmd.Execute(); err == nil {
		t.Fatalf("expected doctor live to fail on unauthorized server response")
	}
	body := out.String()
	if !strings.Contains(body, `"status": "FAIL"`) {
		t.Fatalf("doctor live failure output missing FAIL status: %s", body)
	}
	if strings.Contains(body, "live-token") || strings.Contains(body, "Authorization") {
		t.Fatalf("doctor live leaked sensitive auth material: %s", body)
	}
}
