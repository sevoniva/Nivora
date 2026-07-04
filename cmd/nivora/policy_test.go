package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestPolicyCommandIncludesAttachmentCommands(t *testing.T) {
	cmd := newPolicyCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("policy help failed: %v", err)
	}
	help := out.String()
	for _, command := range []string{"attach", "attachments"} {
		if !strings.Contains(help, command) {
			t.Fatalf("policy help missing %s: %s", command, help)
		}
	}
}

func TestPolicyAttachHelpIncludesScopeFlags(t *testing.T) {
	cmd := newPolicyCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"attach", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("policy attach help failed: %v", err)
	}
	help := out.String()
	for _, flag := range []string{"--scope-type", "--scope-id"} {
		if !strings.Contains(help, flag) {
			t.Fatalf("policy attach help missing %s: %s", flag, help)
		}
	}
}
