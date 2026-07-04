package main

import (
	"bytes"
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
