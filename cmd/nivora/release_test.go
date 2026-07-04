package main

import (
	"bytes"
	"strings"
	"testing"
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
