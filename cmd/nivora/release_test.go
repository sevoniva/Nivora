package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestReleaseCommandIncludesEvidenceAndCancel(t *testing.T) {
	cmd := newReleaseCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("release help failed: %v", err)
	}
	for _, command := range []string{"evidence", "cancel"} {
		if !strings.Contains(out.String(), command) {
			t.Fatalf("release help missing %s command: %s", command, out.String())
		}
	}
}

func TestReleaseEvidenceHelpIncludesFormat(t *testing.T) {
	cmd := newReleaseCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"evidence", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("release evidence help failed: %v", err)
	}
	if !strings.Contains(out.String(), "--format") {
		t.Fatalf("release evidence help missing format flag: %s", out.String())
	}
}

func TestReleasePlanHelpIncludesReleaseIDModeFlags(t *testing.T) {
	cmd := newReleaseCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"plan", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("release plan help failed: %v", err)
	}
	help := out.String()
	for _, want := range []string{"[release-id]", "--environment", "--target", "--catalog-target", "--file"} {
		if !strings.Contains(help, want) {
			t.Fatalf("release plan help missing %q: %s", want, help)
		}
	}
}

func TestBuildReleaseDefinitionFromCLIFlags(t *testing.T) {
	def, err := buildReleaseDefinitionFromCLIFlags("rel-123", "staging", "", []string{"audit-only", "notify:webhook"}, nil, "plan-only")
	if err != nil {
		t.Fatalf("build definition: %v", err)
	}
	if def.Spec.ReleaseID != "rel-123" || def.Spec.Environment != "staging" || def.Spec.Strategy != "plan-only" {
		t.Fatalf("definition spec = %#v", def.Spec)
	}
	if len(def.Spec.Targets) != 2 {
		t.Fatalf("targets = %#v", def.Spec.Targets)
	}
	if def.Spec.Targets[0].Name != "audit-only" || def.Spec.Targets[0].Type != "noop" {
		t.Fatalf("first target = %#v", def.Spec.Targets[0])
	}
	if def.Spec.Targets[1].Name != "notify" || def.Spec.Targets[1].Type != "webhook" {
		t.Fatalf("second target = %#v", def.Spec.Targets[1])
	}
}

func TestBuildReleaseDefinitionFromCLIFlagsWithCatalogTargets(t *testing.T) {
	def, err := buildReleaseDefinitionFromCLIFlags("rel-123", "staging", "", nil, []string{"target-prod", " target-audit "}, "plan-only")
	if err != nil {
		t.Fatalf("build definition: %v", err)
	}
	if len(def.Spec.Targets) != 2 {
		t.Fatalf("targets = %#v", def.Spec.Targets)
	}
	if def.Spec.Targets[0].TargetID != "target-prod" || def.Spec.Targets[0].Name != "" || def.Spec.Targets[0].Type != "" {
		t.Fatalf("first catalog target = %#v", def.Spec.Targets[0])
	}
	if def.Spec.Targets[1].TargetID != "target-audit" || def.Spec.Targets[1].Order != 2 {
		t.Fatalf("second catalog target = %#v", def.Spec.Targets[1])
	}
}

func TestReleaseIDModeRejectsTargetsThatNeedDeploymentSpec(t *testing.T) {
	_, err := buildReleaseDefinitionFromCLIFlags("rel-123", "staging", "", []string{"cluster:kubernetes-yaml"}, nil, "plan-only")
	if err == nil || !strings.Contains(err.Error(), "use --file") {
		t.Fatalf("expected file mode error, got %v", err)
	}
}

func TestReleasePlanCommandSendsCatalogTargetID(t *testing.T) {
	t.Setenv("NIVORA_TEST_TOKEN", "release-token")
	var requestBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/releases/rel-1/plan" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer release-token" {
			t.Fatalf("Authorization header = %q", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		requestBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"rplan-1"}`))
	}))
	defer server.Close()

	cmd := newReleasePlanCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{
		"rel-1",
		"--environment", "prod",
		"--catalog-target", "target-prod",
		"--local=false",
		"--server", server.URL,
		"--token-env", "NIVORA_TEST_TOKEN",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("release plan failed: %v output=%s", err, out.String())
	}
	if !strings.Contains(requestBody, `"targetId":"target-prod"`) {
		t.Fatalf("request body missing catalog target id: %s", requestBody)
	}
	if strings.Contains(requestBody, `"type":"noop"`) || strings.Contains(requestBody, `"name":"target-prod"`) {
		t.Fatalf("catalog target should not be expanded by CLI: %s", requestBody)
	}
}

func TestReleaseIDModeRejectsExplicitLocalMode(t *testing.T) {
	cmd := newReleaseCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"plan", "rel-123", "--environment", "staging", "--target", "audit-only", "--local"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "server-backed release state") {
		t.Fatalf("expected server-backed error, got err=%v out=%s", err, out.String())
	}
}

func TestReleaseServerCommandsUseBearerToken(t *testing.T) {
	t.Setenv("NIVORA_TEST_TOKEN", "release-token")
	releaseFile := writeReleaseDefinition(t)
	tests := []struct {
		name       string
		cmd        *cobra.Command
		args       []string
		wantMethod string
		wantPath   string
		wantQuery  string
		response   string
	}{
		{
			name:       "create",
			cmd:        newReleaseCreateCommand(),
			args:       []string{"--file", releaseFile, "--project-id", "project-a", "--local=false", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"},
			wantMethod: http.MethodPost,
			wantPath:   "/api/v1/releases",
			wantQuery:  "projectId=project-a",
			response:   `{"release":{"id":"rel-1","version":"1.0.0"}}`,
		},
		{
			name:       "get",
			cmd:        newReleaseGetCommand(),
			args:       []string{"rel-1", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"},
			wantMethod: http.MethodGet,
			wantPath:   "/api/v1/releases/rel-1",
			response:   `{"id":"rel-1"}`,
		},
		{
			name:       "list",
			cmd:        newReleaseListCommand(),
			args:       []string{"--project-id", "project-a", "--environment-id", "env-a", "--application-id", "app-a", "--status", "Ready", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"},
			wantMethod: http.MethodGet,
			wantPath:   "/api/v1/releases",
			wantQuery:  "applicationId=app-a&environmentId=env-a&projectId=project-a&status=Ready",
			response:   `[{"release":{"id":"rel-1"}}]`,
		},
		{
			name:       "artifacts",
			cmd:        newReleaseArtifactsCommand(),
			args:       []string{"rel-1", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"},
			wantMethod: http.MethodGet,
			wantPath:   "/api/v1/releases/rel-1/artifacts",
			response:   `[{"id":"artifact-1"}]`,
		},
		{
			name:       "cancel",
			cmd:        newReleaseCancelCommand(),
			args:       []string{"rel-1", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"},
			wantMethod: http.MethodPost,
			wantPath:   "/api/v1/releases/rel-1/cancel",
			response:   `{"id":"rel-1","status":"Canceled"}`,
		},
		{
			name:       "security",
			cmd:        newReleaseSecurityCommand(),
			args:       []string{"rel-1", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"},
			wantMethod: http.MethodGet,
			wantPath:   "/api/v1/releases/rel-1/security",
			response:   `{"releaseId":"rel-1"}`,
		},
		{
			name:       "evidence json",
			cmd:        newReleaseEvidenceCommand(),
			args:       []string{"rel-1", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"},
			wantMethod: http.MethodPost,
			wantPath:   "/api/v1/releases/rel-1/evidence",
			response:   `{"id":"evidence-1"}`,
		},
		{
			name:       "evidence markdown",
			cmd:        newReleaseEvidenceCommand(),
			args:       []string{"rel-1", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN", "--format", "markdown"},
			wantMethod: http.MethodGet,
			wantPath:   "/api/v1/evidence/release/rel-1",
			wantQuery:  "format=markdown",
			response:   `# Evidence`,
		},
		{
			name:       "plan",
			cmd:        newReleasePlanCommand(),
			args:       []string{"rel-1", "--environment", "dev", "--target", "audit-only", "--local=false", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"},
			wantMethod: http.MethodPost,
			wantPath:   "/api/v1/releases/rel-1/plan",
			response:   `{"id":"rplan-1"}`,
		},
		{
			name:       "deploy",
			cmd:        newReleaseDeployCommand(),
			args:       []string{"rel-1", "--environment", "dev", "--target", "audit-only", "--local=false", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"},
			wantMethod: http.MethodPost,
			wantPath:   "/api/v1/releases/rel-1/deploy",
			response:   `{"id":"rexec-1"}`,
		},
		{
			name:       "execution get",
			cmd:        newReleaseExecutionInspectCommand("get", "Get a ReleaseExecution", ""),
			args:       []string{"rexec-1", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"},
			wantMethod: http.MethodGet,
			wantPath:   "/api/v1/releases/executions/rexec-1",
			response:   `{"id":"rexec-1"}`,
		},
		{
			name:       "execution cancel",
			cmd:        newReleaseExecutionCancelCommand(),
			args:       []string{"rexec-1", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"},
			wantMethod: http.MethodPost,
			wantPath:   "/api/v1/releases/executions/rexec-1/cancel",
			response:   `{"id":"rexec-1","status":"Canceled"}`,
		},
		{
			name:       "execution resume",
			cmd:        newReleaseExecutionResumeCommand(),
			args:       []string{"rexec-1", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN", "--approval-status", "Approved"},
			wantMethod: http.MethodPost,
			wantPath:   "/api/v1/releases/executions/rexec-1/resume",
			response:   `{"id":"rexec-1","status":"Running"}`,
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
				if tt.wantQuery != "" && r.URL.RawQuery != tt.wantQuery {
					t.Fatalf("query = %q, want %q", r.URL.RawQuery, tt.wantQuery)
				}
				if got := r.Header.Get("Authorization"); got != "Bearer release-token" {
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

func writeReleaseDefinition(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "release.yaml")
	content := `apiVersion: nivora.io/v1alpha1
kind: Release
metadata:
  name: demo-release
spec:
  version: 1.0.0
  application: demo
  environment: dev
  artifacts:
    - name: demo
      type: image
      reference: example.invalid/sevoniva/demo:1.0.0
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write release file: %v", err)
	}
	return path
}
