package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestPipelineDefinitionCommandIncludesRunAndVersions(t *testing.T) {
	cmd := newPipelineCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"definition", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("pipeline definition help failed: %v", err)
	}
	help := out.String()
	for _, command := range []string{"run", "versions"} {
		if !strings.Contains(help, command) {
			t.Fatalf("pipeline definition help missing %s: %s", command, help)
		}
	}
}

func TestPipelineRunHelpMentionsCatalogMode(t *testing.T) {
	cmd := newPipelineCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"run", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("pipeline run help failed: %v", err)
	}
	help := out.String()
	for _, text := range []string{"--local=false", "--token-env"} {
		if !strings.Contains(help, text) {
			t.Fatalf("pipeline run help missing %s: %s", text, help)
		}
	}
}

func TestPipelineDefinitionRunRejectsInvalidVersion(t *testing.T) {
	cmd := newPipelineDefinitionRunCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"pipe-1", "--version", "0"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "--version must be greater than zero") {
		t.Fatalf("expected invalid version error, got err=%v output=%s", err, out.String())
	}
}
