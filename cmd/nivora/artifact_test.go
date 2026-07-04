package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestArtifactCommandIncludesCatalogQueries(t *testing.T) {
	cmd := newArtifactCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("artifact help failed: %v", err)
	}
	help := out.String()
	for _, command := range []string{"create", "list", "get", "releases"} {
		if !strings.Contains(help, command) {
			t.Fatalf("artifact help missing %s: %s", command, help)
		}
	}
}

func TestArtifactCreateHelpIncludesCatalogFields(t *testing.T) {
	cmd := newArtifactCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"create", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("artifact create help failed: %v", err)
	}
	help := out.String()
	for _, flag := range []string{"--reference", "--type", "--name", "--server", "--token-env"} {
		if !strings.Contains(help, flag) {
			t.Fatalf("artifact create help missing %s: %s", flag, help)
		}
	}
}

func TestArtifactListHelpIncludesInventoryFilters(t *testing.T) {
	cmd := newArtifactCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"list", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("artifact list help failed: %v", err)
	}
	help := out.String()
	for _, flag := range []string{"--type", "--registry", "--repository", "--digest", "--reference"} {
		if !strings.Contains(help, flag) {
			t.Fatalf("artifact list help missing %s: %s", flag, help)
		}
	}
}
