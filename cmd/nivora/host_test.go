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
)

func TestDeploymentHostRunSupportsServerBackedDryRun(t *testing.T) {
	path := writeHostDryRunDefinition(t)
	var called bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/deployments" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		spec := body["spec"].(map[string]any)
		target := spec["target"].(map[string]any)
		if target["type"] != "host" {
			t.Fatalf("target = %#v", target)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"run":{"id":"drun-host","status":"Succeeded"},"logs":[{"stream":"system","content":"ok"}],"hostDetails":[{"hostName":"local-noop-host","status":"Succeeded"}],"rollbackPlan":{"strategy":"symlink-restore"}}`))
	}))
	defer server.Close()

	cmd := newDeploymentHostRunCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--file", path, "--local=false", "--server", server.URL})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("host run failed: %v output=%s", err, out.String())
	}
	if !called {
		t.Fatal("expected command to call server")
	}
	output := out.String()
	for _, want := range []string{"DeploymentRun: drun-host", "Status: Succeeded", "Logs: 1 chunk(s)", "Hosts: 1", "RollbackPlan: symlink-restore"} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q: %s", want, output)
		}
	}
}

func TestDeploymentHostRunServerModeRejectsRemoteApply(t *testing.T) {
	path := writeHostApplyDefinition(t)
	cmd := newDeploymentHostRunCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--file", path, "--local=false", "--server", "http://127.0.0.1:1"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "safe dry-run/noop only") {
		t.Fatalf("expected server remote apply guard, got err=%v output=%s", err, out.String())
	}
}

func writeHostDryRunDefinition(t *testing.T) string {
	t.Helper()
	return writeHostDefinition(t, "host-dry-run.yaml", `apiVersion: nivora.io/v1alpha1
kind: Deployment
metadata:
  name: demo-host-release
spec:
  application: demo
  environment: dev
  target:
    type: host
    name: local-host-group
  artifact:
    name: demo
    type: binary
    reference: ./dist/demo.tar.gz
  host:
    hosts:
      - id: local-noop-host
        name: local-noop-host
        address: 127.0.0.1
    deployPath: /opt/nivora/apps/demo
    dryRun: true
  options:
    dryRun: true
    apply: false
`)
}

func writeHostApplyDefinition(t *testing.T) string {
	t.Helper()
	return writeHostDefinition(t, "host-apply.yaml", `apiVersion: nivora.io/v1alpha1
kind: Deployment
metadata:
  name: demo-host-apply
spec:
  application: demo
  environment: dev
  target:
    type: host
    name: local-host-group
  artifact:
    name: demo
    type: binary
    reference: ./dist/demo.tar.gz
  host:
    hosts:
      - id: local-noop-host
        name: local-noop-host
        address: 127.0.0.1
    deployPath: /opt/nivora/apps/demo
    allowRemoteHostDeploy: true
    credentialRef: cred-host-placeholder
  options:
    dryRun: false
    apply: true
`)
}

func writeHostDefinition(t *testing.T, name string, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write host definition: %v", err)
	}
	return path
}
