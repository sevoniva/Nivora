package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
	var called bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/approvals" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
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
	var called bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/approvals/appr-1" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"appr-1","status":"Pending"}`))
	}))
	defer server.Close()

	cmd := newApprovalGetCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"appr-1", "--server", server.URL})
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
