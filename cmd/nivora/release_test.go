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

func TestReleasePlanHelpIncludesReleaseIDModeFlags(t *testing.T) {
	cmd := newReleaseCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"plan", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("release plan help failed: %v", err)
	}
	help := out.String()
	for _, want := range []string{"[release-id]", "--environment", "--target", "--file"} {
		if !strings.Contains(help, want) {
			t.Fatalf("release plan help missing %q: %s", want, help)
		}
	}
}

func TestBuildReleaseDefinitionFromCLIFlags(t *testing.T) {
	def, err := buildReleaseDefinitionFromCLIFlags("rel-123", "staging", "", []string{"audit-only", "notify:webhook"}, "plan-only")
	if err != nil {
		t.Fatalf("build definition: %v", err)
	}
	if def.Spec.ReleaseID != "rel-123" || def.Spec.Environment != "staging" || def.Spec.Strategy != "plan-only" {
		t.Fatalf("definition spec = %#v", def.Spec)
	}
	if len(def.Spec.Targets) != 2 {
		t.Fatalf("targets = %#v", def.Spec.Targets)
	}
	if def.Spec.Targets[0].Name != "audit-only" || def.Spec.Targets[0].Type != "noop" {
		t.Fatalf("first target = %#v", def.Spec.Targets[0])
	}
	if def.Spec.Targets[1].Name != "notify" || def.Spec.Targets[1].Type != "webhook" {
		t.Fatalf("second target = %#v", def.Spec.Targets[1])
	}
}

func TestReleaseIDModeRejectsTargetsThatNeedDeploymentSpec(t *testing.T) {
	_, err := buildReleaseDefinitionFromCLIFlags("rel-123", "staging", "", []string{"cluster:kubernetes-yaml"}, "plan-only")
	if err == nil || !strings.Contains(err.Error(), "use --file") {
		t.Fatalf("expected file mode error, got %v", err)
	}
}

func TestReleaseIDModeRejectsExplicitLocalMode(t *testing.T) {
	cmd := newReleaseCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"plan", "rel-123", "--environment", "staging", "--target", "audit-only", "--local"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "server-backed release state") {
		t.Fatalf("expected server-backed error, got err=%v out=%s", err, out.String())
	}
}
