package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestSecurityCommandIncludesScanCatalogQueries(t *testing.T) {
	cmd := newSecurityCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("security help failed: %v", err)
	}
	help := out.String()
	for _, command := range []string{"scans", "findings"} {
		if !strings.Contains(help, command) {
			t.Fatalf("security help missing %s: %s", command, help)
		}
	}
}

func TestSecurityFindingsListHelpIncludesFilters(t *testing.T) {
	cmd := newSecurityCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"findings", "list", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("security findings list help failed: %v", err)
	}
	help := out.String()
	for _, flag := range []string{"--scan-id", "--severity", "--category", "--subject-type"} {
		if !strings.Contains(help, flag) {
			t.Fatalf("security findings list help missing %s: %s", flag, help)
		}
	}
}
