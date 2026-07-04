package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestEvidenceCommandIncludesList(t *testing.T) {
	cmd := newEvidenceCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("evidence help failed: %v", err)
	}
	if !strings.Contains(out.String(), "list") {
		t.Fatalf("evidence help missing list command: %s", out.String())
	}
}

func TestEvidenceListHelpIncludesFilters(t *testing.T) {
	cmd := newEvidenceCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"list", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("evidence list help failed: %v", err)
	}
	help := out.String()
	for _, flag := range []string{"--subject-type", "--subject-id", "--limit", "--token-env"} {
		if !strings.Contains(help, flag) {
			t.Fatalf("evidence list help missing %s: %s", flag, help)
		}
	}
}
