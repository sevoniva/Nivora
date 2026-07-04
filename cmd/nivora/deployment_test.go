package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func TestDeploymentServerCommandsUseBearerToken(t *testing.T) {
	t.Setenv("NIVORA_TEST_TOKEN", "deployment-token")
	deploymentFile := writeDeploymentDefinition(t)
	hostFile := writeHostDryRunDefinition(t)
	summaryResponse := `{"run":{"id":"drun-1","status":"Succeeded"},"plan":{"manifestCount":1},"logs":[{"stream":"system","content":"ok"}],"hostDetails":[{"hostName":"local-noop-host","status":"Succeeded"}],"rollbackPlan":{"strategy":"manifest-restore"}}`

	tests := []struct {
		name       string
		cmd        *cobra.Command
		args       []string
		wantMethod string
		wantPath   string
		response   string
	}{
		{
			name:       "host plan",
			cmd:        newDeploymentHostPlanCommand(),
			args:       []string{"--file", hostFile, "--local=false", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"},
			wantMethod: http.MethodPost,
			wantPath:   "/api/v1/deployments/host/plan",
			response:   `{"deploymentRunId":"drun-host","hosts":[{"name":"local-noop-host"}]}`,
		},
		{
			name:       "host run",
			cmd:        newDeploymentHostRunCommand(),
			args:       []string{"--file", hostFile, "--local=false", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"},
			wantMethod: http.MethodPost,
			wantPath:   "/api/v1/deployments",
			response:   summaryResponse,
		},
		{
			name:       "plan",
			cmd:        newDeploymentPlanCommand(),
			args:       []string{deploymentFile, "--local=false", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"},
			wantMethod: http.MethodPost,
			wantPath:   "/api/v1/deployments/plan",
			response:   `{"id":"dplan-1","manifestCount":1}`,
		},
		{
			name:       "run",
			cmd:        newDeploymentRunCommand(),
			args:       []string{deploymentFile, "--local=false", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"},
			wantMethod: http.MethodPost,
			wantPath:   "/api/v1/deployments",
			response:   summaryResponse,
		},
		{
			name:       "apply",
			cmd:        newDeploymentApplyCommand(),
			args:       []string{deploymentFile, "--local=false", "--confirm", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"},
			wantMethod: http.MethodPost,
			wantPath:   "/api/v1/deployments/apply",
			response:   summaryResponse,
		},
		{
			name:       "rollback",
			cmd:        newDeploymentRollbackCommand(),
			args:       []string{"drun-1", "--confirm", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"},
			wantMethod: http.MethodPost,
			wantPath:   "/api/v1/deployments/drun-1/rollback",
			response:   summaryResponse,
		},
		{
			name:       "get",
			cmd:        newDeploymentGetCommand(),
			args:       []string{"drun-1", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"},
			wantMethod: http.MethodGet,
			wantPath:   "/api/v1/deployments/drun-1",
			response:   summaryResponse,
		},
		{
			name:       "logs",
			cmd:        newDeploymentInspectCommand("logs", "Get DeploymentRun logs", "/logs"),
			args:       []string{"drun-1", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"},
			wantMethod: http.MethodGet,
			wantPath:   "/api/v1/deployments/drun-1/logs",
			response:   `[{"stream":"system","content":"ok"}]`,
		},
		{
			name:       "health",
			cmd:        newDeploymentLocalInspectCommand("health", "Get DeploymentRun health", "/health", nil),
			args:       []string{"drun-1", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"},
			wantMethod: http.MethodGet,
			wantPath:   "/api/v1/deployments/drun-1/health",
			response:   `{"health":"Healthy"}`,
		},
		{
			name:       "cancel",
			cmd:        newDeploymentCancelCommand(),
			args:       []string{"drun-1", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"},
			wantMethod: http.MethodPost,
			wantPath:   "/api/v1/deployments/drun-1/cancel",
			response:   summaryResponse,
		},
		{
			name:       "resume",
			cmd:        newDeploymentResumeCommand(),
			args:       []string{"drun-1", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN", "--approval-status", "Approved"},
			wantMethod: http.MethodPost,
			wantPath:   "/api/v1/deployments/drun-1/resume",
			response:   summaryResponse,
		},
		{
			name:       "sync",
			cmd:        newDeploymentSyncCommand(),
			args:       []string{"drun-1", "--confirm", "--allow-sync", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"},
			wantMethod: http.MethodPost,
			wantPath:   "/api/v1/deployments/drun-1/sync",
			response:   `{"status":"skipped","reason":"test"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var called bool
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				if r.Method != tt.wantMethod || r.URL.Path != tt.wantPath {
					t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
				}
				if got := r.Header.Get("Authorization"); got != "Bearer deployment-token" {
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

func writeDeploymentDefinition(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "deployment.yaml")
	content := `apiVersion: nivora.io/v1alpha1
kind: Deployment
metadata:
  name: demo-yaml-deployment
spec:
  application: demo
  environment: dev
  target:
    type: kubernetes-yaml
    name: local-cluster
    namespace: default
  manifests:
    - examples/yaml/deployment.yaml
  options:
    dryRun: true
    apply: false
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write deployment file: %v", err)
	}
	return path
}
