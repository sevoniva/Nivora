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

func TestHostGroupsCommandIncludesManagementCommands(t *testing.T) {
	cmd := newHostGroupsCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("host-groups help failed: %v", err)
	}
	for _, want := range []string{"list", "create", "get"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("host-groups help missing %q: %s", want, out.String())
		}
	}
}

func TestHostGroupsCreateAndGetUseServerRoutes(t *testing.T) {
	t.Setenv("NIVORA_TEST_TOKEN", "opaque-token")
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		if got := r.Header.Get("Authorization"); got != "Bearer opaque-token" {
			t.Fatalf("unexpected authorization header %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/host-groups":
			if r.Method != http.MethodPost {
				t.Fatalf("unexpected method for create: %s", r.Method)
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			if body["name"] != "local-host-group" || body["environmentId"] != "dev" || body["credentialRef"] != "cred-host" {
				t.Fatalf("unexpected group body: %#v", body)
			}
			labels, _ := body["labels"].(map[string]any)
			if labels["tier"] != "app" {
				t.Fatalf("unexpected labels: %#v", body["labels"])
			}
			hosts, _ := body["hosts"].([]any)
			if len(hosts) != 2 {
				t.Fatalf("unexpected hosts: %#v", body["hosts"])
			}
			first := hosts[0].(map[string]any)
			if first["name"] != "web-1" || first["address"] != "127.0.0.1" || first["credentialRef"] != "cred-host" || first["environmentId"] != "dev" {
				t.Fatalf("unexpected host entry: %#v", first)
			}
			raw, _ := json.Marshal(body)
			for _, forbidden := range []string{"opaque-token", "password", "secret", "privateKey", "kubeconfig"} {
				if strings.Contains(strings.ToLower(string(raw)), strings.ToLower(forbidden)) {
					t.Fatalf("host group request leaked forbidden marker %q: %s", forbidden, raw)
				}
			}
			_, _ = w.Write([]byte(`{"id":"hgrp-1","name":"local-host-group","environmentId":"dev","hosts":[{"name":"web-1","address":"127.0.0.1"}]}`))
		case "/api/v1/host-groups/hgrp-1":
			if r.Method != http.MethodGet {
				t.Fatalf("unexpected method for get: %s", r.Method)
			}
			_, _ = w.Write([]byte(`{"id":"hgrp-1","name":"local-host-group"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	createCmd := newHostGroupsCommand()
	var out bytes.Buffer
	createCmd.SetOut(&out)
	createCmd.SetErr(&out)
	createCmd.SetArgs([]string{
		"--server", server.URL,
		"--token-env", "NIVORA_TEST_TOKEN",
		"create",
		"--name", "local-host-group",
		"--env", "dev",
		"--credential-ref", "cred-host",
		"--host", "web-1=127.0.0.1",
		"--host", "web-2=127.0.0.2",
		"--label", "tier=app",
	})
	if err := createCmd.Execute(); err != nil {
		t.Fatalf("host-groups create failed: %v output=%s", err, out.String())
	}
	if !strings.Contains(out.String(), "hgrp-1") {
		t.Fatalf("create output missing host group id: %s", out.String())
	}

	getCmd := newHostGroupsCommand()
	out.Reset()
	getCmd.SetOut(&out)
	getCmd.SetErr(&out)
	getCmd.SetArgs([]string{"--server", server.URL, "--token-env", "NIVORA_TEST_TOKEN", "get", "hgrp-1"})
	if err := getCmd.Execute(); err != nil {
		t.Fatalf("host-groups get failed: %v output=%s", err, out.String())
	}
	if !strings.Contains(out.String(), "local-host-group") {
		t.Fatalf("get output missing host group name: %s", out.String())
	}

	want := []string{"POST /api/v1/host-groups", "GET /api/v1/host-groups/hgrp-1"}
	if len(requests) != len(want) {
		t.Fatalf("requests = %#v", requests)
	}
	for i := range want {
		if requests[i] != want[i] {
			t.Fatalf("request[%d] = %q, want %q", i, requests[i], want[i])
		}
	}
}

func TestBuildHostGroupCreateBodyRejectsInvalidHosts(t *testing.T) {
	for _, hosts := range [][]string{nil, []string{"web-1"}, []string{"=127.0.0.1"}, []string{"web-1="}} {
		if _, err := buildHostGroupCreateBody("local-host-group", "dev", "", nil, hosts); err == nil {
			t.Fatalf("expected invalid hosts %v to fail", hosts)
		}
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
