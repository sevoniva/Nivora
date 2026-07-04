package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestMCPCommandIncludesReadAndCall(t *testing.T) {
	cmd := newMCPCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("mcp help failed: %v", err)
	}
	help := out.String()
	for _, command := range []string{"read-resource", "call-tool", "list-tools", "list-resources"} {
		if !strings.Contains(help, command) {
			t.Fatalf("mcp help missing %s: %s", command, help)
		}
	}
}

func TestMCPReadResourceLocal(t *testing.T) {
	cmd := newMCPCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"read-resource", "nivora://runtime/recovery", "--local", "--config", "../../configs/server.yaml"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("read-resource failed: %v\n%s", err, out.String())
	}
	body := out.String()
	for _, want := range []string{`"pipeline"`, `"deployment"`, `"release"`, `"mutated": false`} {
		if !strings.Contains(body, want) {
			t.Fatalf("read-resource output missing %s: %s", want, body)
		}
	}
}

func TestMCPCallToolLocal(t *testing.T) {
	cmd := newMCPCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"call-tool", "nivora_get_runtime_recovery_status", "--local", "--config", "../../configs/server.yaml"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("call-tool failed: %v\n%s", err, out.String())
	}
	body := out.String()
	if !strings.Contains(body, `"mutated": false`) || !strings.Contains(body, `"readOnly": true`) {
		t.Fatalf("call-tool output = %s", body)
	}
}

func TestMCPCallToolDeniedActionDoesNotLeakArgs(t *testing.T) {
	cmd := newMCPCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"call-tool", "nivora_apply_deployment", "--local", "--config", "../../configs/server.yaml", "--arg", "authorization=Bearer should-not-leak"})
	err := cmd.Execute()
	if err == nil {
		t.Fatalf("blocked action unexpectedly succeeded: %s", out.String())
	}
	body := out.String()
	if !strings.Contains(body, "mcp_action_not_allowed") {
		t.Fatalf("blocked action output missing error code: %s", body)
	}
	if strings.Contains(body, "should-not-leak") || strings.Contains(body, "Authorization: Bearer") {
		t.Fatalf("blocked action leaked sensitive arg: %s", body)
	}
}

func TestParseMCPToolArgs(t *testing.T) {
	args, err := parseMCPToolArgs([]string{"severity=High", "subjectId=demo"}, `{"subjectType":"manifest"}`)
	if err != nil {
		t.Fatalf("parse args: %v", err)
	}
	if args["severity"] != "High" || args["subjectType"] != "manifest" || args["subjectId"] != "demo" {
		t.Fatalf("args = %#v", args)
	}
	if _, err := parseMCPToolArgs([]string{"bad"}, ""); err == nil {
		t.Fatal("expected invalid key=value error")
	}
}
