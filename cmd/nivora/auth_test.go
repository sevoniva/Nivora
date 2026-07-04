package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestAuthCommandIncludesDirectoryCommands(t *testing.T) {
	cmd := newAuthCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("auth help failed: %v", err)
	}
	help := out.String()
	for _, want := range []string{"users", "roles", "permissions"} {
		if !strings.Contains(help, want) {
			t.Fatalf("auth help missing %q: %s", want, help)
		}
	}
}

func TestAuthDirectoryCommandsUseCatalogRoutes(t *testing.T) {
	t.Setenv("NIVORA_TEST_TOKEN", "directory-token")
	tests := []struct {
		name string
		path string
	}{
		{name: "users", path: "/api/v1/users"},
		{name: "roles", path: "/api/v1/roles"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var called bool
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				if r.Method != http.MethodGet || r.URL.Path != tt.path {
					t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
				}
				if got := r.Header.Get("Authorization"); got != "Bearer directory-token" {
					t.Fatalf("unexpected authorization header %q", got)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`[{"id":"local-admin","name":"local-admin"}]`))
			}))
			defer server.Close()

			cmd := newAuthCommand()
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&out)
			cmd.SetArgs([]string{tt.name, "--server", server.URL, "--token-env", "NIVORA_TEST_TOKEN"})
			if err := cmd.Execute(); err != nil {
				t.Fatalf("auth %s failed: %v output=%s", tt.name, err, out.String())
			}
			if !called {
				t.Fatalf("expected auth %s to call server", tt.name)
			}
			if !strings.Contains(out.String(), "local-admin") {
				t.Fatalf("auth %s output missing payload: %s", tt.name, out.String())
			}
		})
	}
}

func TestMembershipCommandsExposeListAndAdd(t *testing.T) {
	tests := []struct {
		name string
		cmd  *cobra.Command
	}{
		{name: "org", cmd: newOrgCommand()},
		{name: "project", cmd: newProjectCommand()},
		{name: "environment", cmd: newEnvironmentCommand()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			tt.cmd.SetOut(&out)
			tt.cmd.SetErr(&out)
			tt.cmd.SetArgs([]string{"members", "--help"})
			if err := tt.cmd.Execute(); err != nil {
				t.Fatalf("%s members help failed: %v", tt.name, err)
			}
			help := out.String()
			for _, want := range []string{"list", "add"} {
				if !strings.Contains(help, want) {
					t.Fatalf("%s members help missing %q: %s", tt.name, want, help)
				}
			}
		})
	}
}

func TestMembershipAddCommandsPostUserAndRoleOnly(t *testing.T) {
	t.Setenv("NIVORA_TEST_TOKEN", "membership-token")
	tests := []struct {
		name     string
		cmd      func() *cobra.Command
		scopeID  string
		wantPath string
	}{
		{name: "org", cmd: newOrgCommand, scopeID: "org-1", wantPath: "/api/v1/orgs/org-1/members"},
		{name: "project", cmd: newProjectCommand, scopeID: "project-1", wantPath: "/api/v1/projects/project-1/members"},
		{name: "environment", cmd: newEnvironmentCommand, scopeID: "env-1", wantPath: "/api/v1/environments/env-1/members"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var called bool
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				if r.Method != http.MethodPost || r.URL.Path != tt.wantPath {
					t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
				}
				if got := r.Header.Get("Authorization"); got != "Bearer membership-token" {
					t.Fatalf("unexpected authorization header %q", got)
				}
				var body map[string]string
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatalf("decode request: %v", err)
				}
				want := map[string]string{"userId": "user-1", "role": "maintainer"}
				if len(body) != len(want) || body["userId"] != want["userId"] || body["role"] != want["role"] {
					t.Fatalf("unexpected membership body: %#v", body)
				}
				raw, _ := json.Marshal(body)
				for _, forbidden := range []string{"token", "password", "secret", "privateKey", "kubeconfig"} {
					if strings.Contains(string(raw), forbidden) {
						t.Fatalf("membership request leaked forbidden marker %q: %s", forbidden, raw)
					}
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"id":"mbr-1","userId":"user-1","role":"maintainer"}`))
			}))
			defer server.Close()

			cmd := tt.cmd()
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&out)
			cmd.SetArgs([]string{"members", "add", tt.scopeID, "--user-id", "user-1", "--role", "maintainer", "--server", server.URL, "--token-env", "NIVORA_TEST_TOKEN"})
			if err := cmd.Execute(); err != nil {
				t.Fatalf("%s members add failed: %v output=%s", tt.name, err, out.String())
			}
			if !called {
				t.Fatalf("expected %s members add to call server", tt.name)
			}
			if !strings.Contains(out.String(), "maintainer") {
				t.Fatalf("%s members add output missing payload: %s", tt.name, out.String())
			}
		})
	}
}

func TestMembershipAddRequiresUserAndRole(t *testing.T) {
	cmd := newProjectCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"members", "add", "project-1", "--role", "developer"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--user-id is required") {
		t.Fatalf("expected user id error, got err=%v output=%s", err, out.String())
	}

	cmd = newProjectCommand()
	out.Reset()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"members", "add", "project-1", "--user-id", "user-1"})
	err = cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--role is required") {
		t.Fatalf("expected role error, got err=%v output=%s", err, out.String())
	}
}
