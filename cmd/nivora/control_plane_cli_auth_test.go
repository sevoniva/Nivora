package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func TestControlPlaneServerCommandsUseBearerToken(t *testing.T) {
	t.Setenv("NIVORA_TEST_TOKEN", "control-plane-token")
	pluginFile := writePluginManifest(t)
	gitOpsCatalogFile := filepath.Join(t.TempDir(), "gitops-catalog.yaml")
	if err := os.WriteFile(gitOpsCatalogFile, []byte(`apiVersion: nivora.io/v1alpha1
kind: Deployment
metadata:
  name: demo-gitops
spec:
  application: demo
  environment: dev
  target:
    type: argocd
    name: demo-argocd
    applicationName: demo
    repositoryId: repo-1
    path: apps/demo/dev
  artifacts:
    - name: demo
      type: image
      reference: registry.example.invalid/demo/app@sha256:example
  gitops:
    mode: plan
`), 0o600); err != nil {
		t.Fatalf("write gitops catalog deployment: %v", err)
	}
	runResponse := `{"run":{"id":"prun-1","status":"Succeeded"},"logs":[]}`

	tests := []struct {
		name       string
		cmd        *cobra.Command
		args       []string
		wantMethod string
		wantPath   string
		wantQuery  string
		response   string
	}{
		{name: "audit verify", cmd: newAuditVerifyCommand(), args: []string{"--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodGet, wantPath: "/api/v1/audit/verify", response: `{"valid":true}`},
		{name: "timeline search", cmd: newTimelineSearchCommand(), args: []string{"--pipeline-run-id", "prun-1", "--limit", "2", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodGet, wantPath: "/api/v1/timeline", wantQuery: "limit=2&pipelineRunId=prun-1", response: `{"timeline":[],"count":0}`},
		{name: "evidence generate", cmd: newEvidenceGenerateCommand(), args: []string{"--subject-type", "release", "--subject-id", "rel-1", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodPost, wantPath: "/api/v1/evidence/bundles", response: `{"id":"evb-1"}`},
		{name: "evidence export json", cmd: newEvidenceExportCommand(), args: []string{"evb-1", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodGet, wantPath: "/api/v1/evidence/bundles/evb-1/export", response: `{"id":"evb-1"}`},
		{name: "evidence export markdown", cmd: newEvidenceExportCommand(), args: []string{"evb-1", "--format", "markdown", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodGet, wantPath: "/api/v1/evidence/bundles/evb-1/export", response: `# Evidence`},
		{name: "quota view", cmd: newScopedGetCommand("view", "View quota", "/api/v1/tenancy/quota"), args: []string{"--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodGet, wantPath: "/api/v1/tenancy/quota", response: `{"scopeType":"global"}`},
		{name: "runtime status", cmd: newRuntimeInspectCommand("status", "Show runtime", http.MethodGet, "/api/v1/system/runtime/recovery"), args: []string{"--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodGet, wantPath: "/api/v1/system/runtime/recovery", response: `{"queued":0}`},

		{name: "plugins list", cmd: newPluginsListCommand(), args: []string{"--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodGet, wantPath: "/api/v1/plugins", response: `[]`},
		{name: "plugins inspect", cmd: newPluginsInspectCommand(), args: []string{"scanner-noop", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodGet, wantPath: "/api/v1/plugins/scanner-noop", response: `{"name":"scanner-noop"}`},
		{name: "plugins capabilities", cmd: newPluginsCapabilitiesCommand(), args: []string{"scanner-noop", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodGet, wantPath: "/api/v1/plugins/scanner-noop/capabilities", response: `[]`},
		{name: "plugins validate", cmd: newPluginsValidateCommand(), args: []string{"--file", pluginFile, "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodPost, wantPath: "/api/v1/plugins/validate", response: `{"valid":true}`},

		{name: "repository validate", cmd: newRepositoryValidateCommand(), args: []string{"repo-1", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodPost, wantPath: "/api/v1/repositories/repo-1/validate", response: `{"valid":true,"repositoryId":"repo-1"}`},
		{name: "repository devops plan", cmd: newRepositoryDevOpsPlanCommand(), args: []string{"repo-1", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodPost, wantPath: "/api/v1/devops/plan", response: `{"plan":{"repositoryId":"repo-1"},"mutated":false}`},
		{name: "workflow cancel", cmd: newWorkflowCancelCommand(), args: []string{"wrun-1", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodPost, wantPath: "/api/v1/workflows/runs/wrun-1/cancel", response: `{"id":"wrun-1","status":"Canceled"}`},
		{name: "gitops plan server catalog", cmd: newGitOpsPlanCommand(), args: []string{gitOpsCatalogFile, "--local=false", "--project-id", "project-a", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodPost, wantPath: "/api/v1/deployments/gitops/plan", wantQuery: "projectId=project-a", response: `{"deploymentRunId":"drun-1","repositoryId":"repo-1","repoURL":"https://example.com/gitops.git"}`},

		{name: "policy attach", cmd: newPolicyAttachCommand(), args: []string{"policy-1", "--scope-type", "project", "--scope-id", "project-a", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodPost, wantPath: "/api/v1/policies/policy-1/attachments", response: `{"id":"attach-1"}`},
		{name: "policy attachments", cmd: newPolicyAttachmentsCommand(), args: []string{"policy-1", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodGet, wantPath: "/api/v1/policies/policy-1/attachments", response: `[]`},
		{name: "policy list", cmd: newPolicyListCommand(), args: []string{"--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodGet, wantPath: "/api/v1/policies", response: `[]`},
		{name: "policy create", cmd: newPolicyCreateCommand(), args: []string{"--name", "deny-critical", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodPost, wantPath: "/api/v1/policies", response: `{"id":"policy-1"}`},
		{name: "policy get", cmd: newPolicyGetCommand(), args: []string{"policy-1", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodGet, wantPath: "/api/v1/policies/policy-1", response: `{"id":"policy-1"}`},
		{name: "policy update", cmd: newPolicyUpdateCommand(), args: []string{"policy-1", "--name", "updated", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodPatch, wantPath: "/api/v1/policies/policy-1", response: `{"id":"policy-1"}`},
		{name: "policy disable", cmd: newPolicyDisableCommand(), args: []string{"policy-1", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodDelete, wantPath: "/api/v1/policies/policy-1", response: `{"id":"policy-1","enabled":false}`},
		{name: "policy evaluate saved", cmd: newPolicyEvaluateCommand(), args: []string{"policy-1", "--subject", "registry.example.invalid/team/app:1.0.0", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodPost, wantPath: "/api/v1/policies/policy-1/evaluate", response: `{"id":"policy-result-1","policyId":"policy-1","decision":"deny"}`},
		{name: "policy results list", cmd: newPolicyResultsListCommand(), args: []string{"--policy-id", "policy-1", "--project-id", "project-a", "--environment-id", "env-a", "--decision", "deny", "--limit", "2", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodGet, wantPath: "/api/v1/policies/results", wantQuery: "decision=deny&environmentId=env-a&limit=2&policyId=policy-1&projectId=project-a", response: `{"results":[{"id":"policy-result-1"}]}`},
		{name: "policy result get", cmd: newPolicyResultGetCommand(), args: []string{"policy-result-1", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodGet, wantPath: "/api/v1/policies/results/policy-result-1", response: `{"id":"policy-result-1"}`},
		{name: "security scans list", cmd: newSecurityScansListCommand(), args: []string{"--project-id", "project-a", "--environment-id", "env-a", "--status", "Succeeded", "--limit", "2", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodGet, wantPath: "/api/v1/security/scans", wantQuery: "environmentId=env-a&limit=2&projectId=project-a&status=Succeeded", response: `{"scans":[{"scan":{"id":"scan-1"}}]}`},
		{name: "security findings list", cmd: newSecurityFindingsListCommand(), args: []string{"--scan-id", "scan-1", "--project-id", "project-a", "--environment-id", "env-a", "--severity", "High", "--category", "misconfiguration", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodGet, wantPath: "/api/v1/security/findings", wantQuery: "category=misconfiguration&environmentId=env-a&projectId=project-a&scanId=scan-1&severity=High", response: `{"findings":[{"id":"finding-1"}]}`},
		{name: "security finding get", cmd: newSecurityFindingGetCommand(), args: []string{"finding-1", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodGet, wantPath: "/api/v1/security/findings/finding-1", response: `{"id":"finding-1"}`},

		{name: "artifact list", cmd: newArtifactListCommand(), args: []string{"--project-id", "project-a", "--environment-id", "env-a", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodGet, wantPath: "/api/v1/artifacts", wantQuery: "environmentId=env-a&projectId=project-a", response: `[]`},
		{name: "artifact create", cmd: newArtifactCreateCommand(), args: []string{"--reference", "registry.example.invalid/team/app:1.0.0", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodPost, wantPath: "/api/v1/artifacts", response: `{"id":"artifact-1"}`},
		{name: "artifact get", cmd: newArtifactGetCommand(), args: []string{"artifact-1", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodGet, wantPath: "/api/v1/artifacts/artifact-1", response: `{"id":"artifact-1"}`},
		{name: "artifact releases", cmd: newArtifactReleasesCommand(), args: []string{"artifact-1", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodGet, wantPath: "/api/v1/artifacts/artifact-1/releases", response: `[]`},
		{name: "artifact registry list", cmd: newArtifactRegistryListCommand(), args: []string{"--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodGet, wantPath: "/api/v1/artifact-registries", response: `[]`},
		{name: "artifact registry create", cmd: newArtifactRegistryCreateCommand(), args: []string{"--name", "local", "--endpoint", "registry.example.invalid", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodPost, wantPath: "/api/v1/artifact-registries", response: `{"id":"reg-1"}`},
		{name: "artifact registry get", cmd: newArtifactRegistryGetCommand(), args: []string{"reg-1", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodGet, wantPath: "/api/v1/artifact-registries/reg-1", response: `{"id":"reg-1"}`},
		{name: "artifact registry update", cmd: newArtifactRegistryUpdateCommand(), args: []string{"reg-1", "--name", "updated", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodPatch, wantPath: "/api/v1/artifact-registries/reg-1", response: `{"id":"reg-1"}`},
		{name: "artifact registry disable", cmd: newArtifactRegistryDisableCommand(), args: []string{"reg-1", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodDelete, wantPath: "/api/v1/artifact-registries/reg-1", response: `{"id":"reg-1","enabled":false}`},
		{name: "artifact registry validate", cmd: newArtifactRegistryValidateCommand(), args: []string{"--name", "local", "--endpoint", "registry.example.invalid", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodPost, wantPath: "/api/v1/artifact-registries/validate", response: `{"valid":true}`},
		{name: "artifact registry validate saved", cmd: newArtifactRegistryValidateCommand(), args: []string{"reg-1", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodPost, wantPath: "/api/v1/artifact-registries/reg-1/validate", response: `{"valid":true,"registryId":"reg-1"}`},
		{name: "artifact registry artifacts", cmd: newArtifactRegistryArtifactsCommand(), args: []string{"reg-1", "--repository", "team/app", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodGet, wantPath: "/api/v1/artifact-registries/reg-1/artifacts", wantQuery: "repository=team%2Fapp", response: `{"registryId":"reg-1","repository":"team/app","artifacts":[]}`},

		{name: "pipeline get", cmd: newPipelineGetCommand(), args: []string{"prun-1", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodGet, wantPath: "/api/v1/pipeline-runs/prun-1", response: runResponse},
		{name: "pipeline logs", cmd: newPipelineInspectCommand("logs", "Get PipelineRun logs", "/logs"), args: []string{"prun-1", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodGet, wantPath: "/api/v1/pipeline-runs/prun-1/logs", response: `[]`},
		{name: "pipeline artifacts", cmd: newPipelineInspectCommand("artifacts", "Get PipelineRun artifact metadata", "/artifacts"), args: []string{"prun-1", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodGet, wantPath: "/api/v1/pipeline-runs/prun-1/artifacts", response: `[]`},
		{name: "pipeline caches", cmd: newPipelineInspectCommand("caches", "Get PipelineRun cache metadata", "/caches"), args: []string{"prun-1", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodGet, wantPath: "/api/v1/pipeline-runs/prun-1/caches", response: `[]`},
		{name: "pipeline annotations", cmd: newPipelineInspectCommand("annotations", "Get PipelineRun annotations", "/annotations"), args: []string{"prun-1", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodGet, wantPath: "/api/v1/pipeline-runs/prun-1/annotations", response: `[]`},
		{name: "pipeline summary", cmd: newPipelineInspectCommand("summary", "Get PipelineRun metadata summary", "/summary"), args: []string{"prun-1", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodGet, wantPath: "/api/v1/pipeline-runs/prun-1/summary", response: `{"pipelineRunId":"prun-1"}`},
		{name: "pipeline cancel", cmd: newPipelineCancelCommand(), args: []string{"prun-1", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodPost, wantPath: "/api/v1/pipeline-runs/prun-1/cancel", response: runResponse},
		{name: "pipeline definition versions", cmd: newPipelineDefinitionVersionsCommand(), args: []string{"pipe-1", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodGet, wantPath: "/api/v1/pipelines/pipe-1/versions", response: `{"pipelineId":"pipe-1","versions":[]}`},
		{name: "pipeline definition run version", cmd: newPipelineDefinitionRunCommand(), args: []string{"pipe-1", "--version", "2", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodPost, wantPath: "/api/v1/pipelines/pipe-1/runs", wantQuery: "version=2", response: runResponse},

		{name: "runner list", cmd: newRunnerListCommand(), args: []string{"--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodGet, wantPath: "/api/v1/runners", response: `[]`},
		{name: "runner group list", cmd: newRunnerGroupListCommand(), args: []string{"--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodGet, wantPath: "/api/v1/runner-groups", response: `[]`},
		{name: "runner group create", cmd: newRunnerGroupCreateCommand(), args: []string{"--id", "rgrp-1", "--name", "prod", "--project-id", "project-a", "--environment-id", "env-prod", "--executor", "shell", "--max-concurrency", "2", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodPost, wantPath: "/api/v1/runner-groups", response: `{"id":"rgrp-1"}`},
		{name: "runner group get", cmd: newRunnerGroupGetCommand(), args: []string{"rgrp-1", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodGet, wantPath: "/api/v1/runner-groups/rgrp-1", response: `{"id":"rgrp-1"}`},
		{name: "runner register", cmd: newRunnerRegisterCommand(), args: []string{"--name", "runner-1", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodPost, wantPath: "/api/v1/runners/register", response: `{"id":"runner-1"}`},
		{name: "runner status", cmd: newRunnerStatusCommand(), args: []string{"runner-1", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodGet, wantPath: "/api/v1/runners/runner-1", response: `{"id":"runner-1"}`},
		{name: "runner token rotate", cmd: newRunnerTokenRotateCommand(), args: []string{"runner-1", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodPost, wantPath: "/api/v1/runners/runner-1/token/rotate", response: `{"runnerId":"runner-1","token":"one-time-token"}`},
		{name: "runner token revoke", cmd: newRunnerTokenRevokeCommand(), args: []string{"runner-1", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodPost, wantPath: "/api/v1/runners/runner-1/token/revoke", response: `{"runnerId":"runner-1","revoked":true}`},
		{name: "runner offline detect", cmd: newRunnerOfflineDetectCommand(), args: []string{"--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"}, wantMethod: http.MethodPost, wantPath: "/api/v1/runners/offline-detect", response: `{"offline":0}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var called bool
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				if r.Method != tt.wantMethod || r.URL.Path != tt.wantPath {
					t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
				}
				if tt.wantQuery != "" && r.URL.RawQuery != tt.wantQuery {
					t.Fatalf("unexpected query %q, want %q", r.URL.RawQuery, tt.wantQuery)
				}
				if got := r.Header.Get("Authorization"); got != "Bearer control-plane-token" {
					t.Fatalf("Authorization header = %q", got)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.response))
			}))
			defer server.Close()

			args := replaceServerURL(tt.args, server.URL)
			var out bytes.Buffer
			tt.cmd.SetOut(&out)
			tt.cmd.SetErr(&out)
			tt.cmd.SetArgs(args)
			if err := tt.cmd.Execute(); err != nil {
				t.Fatalf("%s failed: %v output=%s", tt.name, err, out.String())
			}
			if !called {
				t.Fatalf("%s did not call server", tt.name)
			}
		})
	}
}

func TestRunnerRegisterCommandSendsGroupIDAndExecutors(t *testing.T) {
	t.Setenv("NIVORA_TEST_TOKEN", "control-plane-token")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/runners/register" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["groupId"] != "rgrp-prod" {
			t.Fatalf("groupId = %#v", body["groupId"])
		}
		executors, _ := body["executors"].([]any)
		if len(executors) != 1 || executors[0] != "container" {
			t.Fatalf("executors = %#v", body["executors"])
		}
		_, _ = w.Write([]byte(`{"runner":{"id":"runner-1","groupId":"rgrp-prod"},"token":{"token":"one-time"}}`))
	}))
	defer server.Close()

	cmd := newRunnerRegisterCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--name", "runner-1", "--group-id", "rgrp-prod", "--executor", "container", "--server", server.URL, "--token-env", "NIVORA_TEST_TOKEN"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("runner register failed: %v output=%s", err, out.String())
	}
}

func writePluginManifest(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "plugin.yaml")
	content := `apiVersion: nivora.io/v1alpha1
name: noop-scanner
type: scanner
version: 0.1.0
protocol: http
capabilities: []
compatibility:
  minNivoraVersion: 0.1.0
status: enabled
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write plugin manifest: %v", err)
	}
	return path
}
