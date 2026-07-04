package main

import (
	"bytes"
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
	if !strings.Contains(out.String(), "resume") {
		t.Fatalf("approvals help missing resume command: %s", out.String())
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
