package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestReleaseCommandIncludesEvidence(t *testing.T) {
	cmd := newReleaseCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("release help failed: %v", err)
	}
	if !strings.Contains(out.String(), "evidence") {
		t.Fatalf("release help missing evidence command: %s", out.String())
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
