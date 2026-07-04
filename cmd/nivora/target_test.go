package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestTargetCommandHelpIncludesValidate(t *testing.T) {
	cmd := newTargetCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("target help failed: %v", err)
	}
	if !strings.Contains(out.String(), "validate") {
		t.Fatalf("target help missing validate command: %s", out.String())
	}
}

func TestTargetCreateRequiresEnvironmentNameAndType(t *testing.T) {
	cmd := newTargetCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"create", "--name", "dev-noop", "--type", "noop"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--environment-id is required") {
		t.Fatalf("expected environment id error, got err=%v out=%s", err, out.String())
	}
}

func TestTargetListSupportsProjectAndEnvironmentFilters(t *testing.T) {
	cmd := newTargetCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"list", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("target list help failed: %v", err)
	}
	help := out.String()
	for _, flag := range []string{"--project-id", "--environment-id"} {
		if !strings.Contains(help, flag) {
			t.Fatalf("target list help missing %s: %s", flag, help)
		}
	}
}
