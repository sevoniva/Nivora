package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestApprovalsCommandIncludesResume(t *testing.T) {
	cmd := newApprovalsCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("approvals help failed: %v", err)
	}
	for _, command := range []string{"list", "create", "get", "approve", "reject", "cancel", "expire", "resume"} {
		if !strings.Contains(out.String(), command) {
			t.Fatalf("approvals help missing %s command: %s", command, out.String())
		}
	}
}

func TestApprovalCreatePostsServerRoute(t *testing.T) {
	t.Setenv("NIVORA_TEST_TOKEN", "approval-token")
	var called bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/approvals" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer approval-token" {
			t.Fatalf("Authorization header = %q", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if body["subjectType"] != "deployment" || body["subjectId"] != "drun-1" || body["environmentId"] != "prod" {
			t.Fatalf("unexpected body: %#v", body)
		}
		if body["requiredByPolicy"] != true || body["requestedBy"] != "reviewer" || body["reason"] != "prod deploy" {
			t.Fatalf("unexpected metadata: %#v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"appr-1","subjectType":"deployment","subjectId":"drun-1","status":"Pending"}`))
	}))
	defer server.Close()

	cmd := newApprovalCreateCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{
		"--server", server.URL,
		"--token-env", "NIVORA_TEST_TOKEN",
		"--subject-type", "deployment",
		"--subject-id", "drun-1",
		"--env", "prod",
		"--requested-by", "reviewer",
		"--reason", "prod deploy",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("approval create failed: %v output=%s", err, out.String())
	}
	if !called {
		t.Fatal("expected approval create to call server")
	}
	if !strings.Contains(out.String(), "appr-1") {
		t.Fatalf("approval create output missing id: %s", out.String())
	}
}

func TestApprovalCreateRequiresSubject(t *testing.T) {
	cmd := newApprovalCreateCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--subject-id", "drun-1"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--subject-type is required") {
		t.Fatalf("expected subject-type error, got err=%v output=%s", err, out.String())
	}

	cmd = newApprovalCreateCommand()
	out.Reset()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--subject-type", "deployment"})
	err = cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--subject-id is required") {
		t.Fatalf("expected subject-id error, got err=%v output=%s", err, out.String())
	}
}

func TestApprovalGetUsesServerRoute(t *testing.T) {
	t.Setenv("NIVORA_TEST_TOKEN", "approval-read-token")
	var called bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/approvals/appr-1" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer approval-read-token" {
			t.Fatalf("Authorization header = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"appr-1","status":"Pending"}`))
	}))
	defer server.Close()

	cmd := newApprovalGetCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"appr-1", "--server", server.URL, "--token-env", "NIVORA_TEST_TOKEN"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("approval get failed: %v output=%s", err, out.String())
	}
	if !called {
		t.Fatal("expected approval get to call server")
	}
	if !strings.Contains(out.String(), "appr-1") {
		t.Fatalf("approval get output missing id: %s", out.String())
	}
}

func TestApprovalProtectedCommandsUseBearerToken(t *testing.T) {
	t.Setenv("NIVORA_TEST_TOKEN", "approval-command-token")
	tests := []struct {
		name       string
		cmd        *cobra.Command
		args       []string
		wantMethod string
		wantPath   string
		response   string
	}{
		{
			name:       "list",
			cmd:        newApprovalListCommand(),
			args:       []string{"--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"},
			wantMethod: http.MethodGet,
			wantPath:   "/api/v1/approvals",
			response:   `[{"id":"appr-1"}]`,
		},
		{
			name:       "approve",
			cmd:        newApprovalDecisionCommand("approve", "Approve an approval request", "/approve"),
			args:       []string{"appr-1", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN", "--comment", "ok"},
			wantMethod: http.MethodPost,
			wantPath:   "/api/v1/approvals/appr-1/approve",
			response:   `{"id":"appr-1","status":"Approved"}`,
		},
		{
			name:       "resume",
			cmd:        newApprovalResumeSubjectCommand(),
			args:       []string{"appr-1", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"},
			wantMethod: http.MethodPost,
			wantPath:   "/api/v1/approvals/appr-1/resume-subject",
			response:   `{"resumed":true}`,
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
				if got := r.Header.Get("Authorization"); got != "Bearer approval-command-token" {
					t.Fatalf("Authorization header = %q", got)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.response))
			}))
			defer server.Close()

			args := append([]string(nil), tt.args...)
			for i, arg := range args {
				if arg == "SERVER_URL" {
					args[i] = server.URL
				}
			}
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

func TestApprovalResumeHelp(t *testing.T) {
	cmd := newApprovalsCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"resume", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("approvals resume help failed: %v", err)
	}
	help := out.String()
	for _, want := range []string{"resume <id>", "DeploymentRun", "ReleaseExecution", "PipelineRun"} {
		if !strings.Contains(help, want) {
			t.Fatalf("approvals resume help missing %q: %s", want, help)
		}
	}
}
