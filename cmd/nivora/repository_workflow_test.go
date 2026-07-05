package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepositoryInspectLocalCommand(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.invalid/cli\n\ngo 1.23\n"), 0o600); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("TOKEN=should-not-leak\n"), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	cmd := newRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"repository", "inspect", "--path", root, "--name", "cli-local"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("repository inspect failed: %v output=%s", err, out.String())
	}
	body := out.String()
	for _, want := range []string{"go test ./...", "intelligence", "snapshot", "secret-like environment file"} {
		if !strings.Contains(body, want) {
			t.Fatalf("repository inspect output missing %q: %s", want, body)
		}
	}
	if strings.Contains(body, "should-not-leak") {
		t.Fatalf("repository inspect leaked .env content: %s", body)
	}
}

func TestWorkflowValidateAndPlanCommands(t *testing.T) {
	workflow := filepath.Join(t.TempDir(), "ci.yaml")
	if err := os.WriteFile(workflow, []byte(`
apiVersion: nivora.io/v1alpha1
kind: Workflow
metadata:
  name: cli-ci
on: [manual]
jobs:
  test:
    steps:
      - name: test
        run: go test ./...
`), 0o600); err != nil {
		t.Fatalf("write workflow: %v", err)
	}

	for _, args := range [][]string{
		{"workflow", "validate", "--file", workflow},
		{"workflow", "plan", "--file", workflow},
	} {
		t.Run(strings.Join(args[:2], " "), func(t *testing.T) {
			cmd := newRootCommand()
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&out)
			cmd.SetArgs(args)
			if err := cmd.Execute(); err != nil {
				t.Fatalf("%v failed: %v output=%s", args, err, out.String())
			}
			body := out.String()
			for _, want := range []string{"cli-ci", "workflowId", "go test ./..."} {
				if !strings.Contains(body, want) {
					t.Fatalf("%v output missing %q: %s", args, want, body)
				}
			}
		})
	}
}
