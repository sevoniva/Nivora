package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCommandIncludesAggregateObservabilityCommands(t *testing.T) {
	cmd := newRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("root help failed: %v", err)
	}
	help := out.String()
	for _, command := range []string{"events", "logs", "audit"} {
		if !strings.Contains(help, command) {
			t.Fatalf("root help missing %s command: %s", command, help)
		}
	}
}

func TestEventsSearchHelpIncludesFilters(t *testing.T) {
	cmd := newEventsCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"search", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("events search help failed: %v", err)
	}
	help := out.String()
	for _, flag := range []string{"--pipeline-run-id", "--deployment-run-id", "--release-id", "--type", "--limit", "--token-env"} {
		if !strings.Contains(help, flag) {
			t.Fatalf("events search help missing %s: %s", flag, help)
		}
	}
}

func TestLogsSearchHelpIncludesFilters(t *testing.T) {
	cmd := newLogsCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"search", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("logs search help failed: %v", err)
	}
	help := out.String()
	for _, flag := range []string{"--pipeline-run-id", "--deployment-run-id", "--job-run-id", "--contains", "--stream", "--limit", "--token-env"} {
		if !strings.Contains(help, flag) {
			t.Fatalf("logs search help missing %s: %s", flag, help)
		}
	}
}

func TestAuditSearchHelpIncludesScopeAndPaginationFilters(t *testing.T) {
	cmd := newAuditCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"search", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("audit search help failed: %v", err)
	}
	help := out.String()
	for _, flag := range []string{"--subject-type", "--subject-id", "--scope-type", "--scope-id", "--request-id", "--correlation-id", "--limit", "--token-env"} {
		if !strings.Contains(help, flag) {
			t.Fatalf("audit search help missing %s: %s", flag, help)
		}
	}
}
