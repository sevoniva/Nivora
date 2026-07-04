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

func TestCloudAccountCommandIncludesMetadataCommands(t *testing.T) {
	cmd := newCloudAccountCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("cloud account help failed: %v", err)
	}
	for _, want := range []string{"list", "create", "get", "validate"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("cloud account help missing %q: %s", want, out.String())
		}
	}
}

func TestCloudAccountCreatePostsMetadataOnly(t *testing.T) {
	t.Setenv("NIVORA_TEST_TOKEN", "cloud-create-token")
	var called bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/cloud/accounts" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer cloud-create-token" {
			t.Fatalf("Authorization header = %q", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if body["name"] != "dev-aws" || body["provider"] != "aws" || body["credentialRef"] != "cred-cloud-placeholder" {
			t.Fatalf("unexpected body: %#v", body)
		}
		config, _ := body["config"].(map[string]any)
		if config["provider"] != "aws" || config["defaultRegion"] != "us-test-1" || config["endpoint"] != "https://cloud.example.invalid" {
			t.Fatalf("unexpected config: %#v", config)
		}
		regions, _ := config["regions"].([]any)
		if len(regions) != 2 || regions[0] != "us-test-1" || regions[1] != "us-test-2" {
			t.Fatalf("unexpected regions: %#v", regions)
		}
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		for _, forbidden := range []string{"password", "secretKey", "token", "privateKey", "kubeconfig"} {
			if strings.Contains(string(raw), forbidden) {
				t.Fatalf("request body leaked forbidden marker %q: %s", forbidden, raw)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"cloudacct-1","name":"dev-aws","provider":"aws","credentialRef":"cred-cloud-placeholder"}`))
	}))
	defer server.Close()

	cmd := newCloudAccountCreateCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{
		"--server", server.URL,
		"--token-env", "NIVORA_TEST_TOKEN",
		"--name", "dev-aws",
		"--provider", "aws",
		"--credential-ref", "cred-cloud-placeholder",
		"--default-region", "us-test-1",
		"--region", "us-test-1",
		"--region", "us-test-2",
		"--endpoint", "https://cloud.example.invalid",
		"--metadata", "owner=platform",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("cloud account create failed: %v output=%s", err, out.String())
	}
	if !called {
		t.Fatal("expected server to be called")
	}
	if !strings.Contains(out.String(), "cloudacct-1") {
		t.Fatalf("create output missing account id: %s", out.String())
	}
}

func TestCloudAccountCreateRequiresNameAndProvider(t *testing.T) {
	cmd := newCloudAccountCreateCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--provider", "aws"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--name is required") {
		t.Fatalf("expected name validation error, got err=%v output=%s", err, out.String())
	}

	cmd = newCloudAccountCreateCommand()
	out.Reset()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--name", "dev-aws"})
	err = cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--provider is required") {
		t.Fatalf("expected provider validation error, got err=%v output=%s", err, out.String())
	}
}

func TestCloudAccountListAndGetUseServerRoutes(t *testing.T) {
	t.Setenv("NIVORA_TEST_TOKEN", "cloud-read-token")
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		if got := r.Header.Get("Authorization"); got != "Bearer cloud-read-token" {
			t.Fatalf("Authorization header = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/cloud/accounts":
			_, _ = w.Write([]byte(`[{"id":"cloudacct-1","name":"dev-aws"}]`))
		case "/api/v1/cloud/accounts/cloudacct-1":
			_, _ = w.Write([]byte(`{"id":"cloudacct-1","name":"dev-aws"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	listCmd := newCloudAccountListCommand()
	var out bytes.Buffer
	listCmd.SetOut(&out)
	listCmd.SetErr(&out)
	listCmd.SetArgs([]string{"--server", server.URL, "--token-env", "NIVORA_TEST_TOKEN"})
	if err := listCmd.Execute(); err != nil {
		t.Fatalf("cloud account list failed: %v output=%s", err, out.String())
	}

	getCmd := newCloudAccountGetCommand()
	out.Reset()
	getCmd.SetOut(&out)
	getCmd.SetErr(&out)
	getCmd.SetArgs([]string{"cloudacct-1", "--server", server.URL, "--token-env", "NIVORA_TEST_TOKEN"})
	if err := getCmd.Execute(); err != nil {
		t.Fatalf("cloud account get failed: %v output=%s", err, out.String())
	}

	want := []string{"GET /api/v1/cloud/accounts", "GET /api/v1/cloud/accounts/cloudacct-1"}
	if len(paths) != len(want) {
		t.Fatalf("paths = %#v", paths)
	}
	for i := range want {
		if paths[i] != want[i] {
			t.Fatalf("path[%d] = %q, want %q", i, paths[i], want[i])
		}
	}
}

func TestCloudProtectedCommandsUseBearerToken(t *testing.T) {
	t.Setenv("NIVORA_TEST_TOKEN", "cloud-command-token")
	tests := []struct {
		name       string
		cmd        *cobra.Command
		args       []string
		wantMethod string
		wantPath   string
		wantQuery  string
		response   string
	}{
		{
			name:       "providers",
			cmd:        newCloudGetCommand("providers", "List configured cloud provider types", "/api/v1/cloud/providers"),
			args:       []string{"--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"},
			wantMethod: http.MethodGet,
			wantPath:   "/api/v1/cloud/providers",
			response:   `[{"type":"aws"}]`,
		},
		{
			name:       "validate",
			cmd:        newCloudAccountValidateCommand(),
			args:       []string{"cloudacct-1", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"},
			wantMethod: http.MethodPost,
			wantPath:   "/api/v1/cloud/accounts/cloudacct-1/validate",
			response:   `{"valid":true}`,
		},
		{
			name:       "inventory",
			cmd:        newCloudAccountInspectCommand("inventory", "Get a cloud inventory snapshot", "/inventory"),
			args:       []string{"cloudacct-1", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN", "--region", "us test/1"},
			wantMethod: http.MethodGet,
			wantPath:   "/api/v1/cloud/accounts/cloudacct-1/inventory",
			wantQuery:  "region=us+test%2F1",
			response:   `{"accountId":"cloudacct-1"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var called bool
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				if r.Method != tt.wantMethod || r.URL.Path != tt.wantPath {
					t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
				}
				if tt.wantQuery != "" && r.URL.RawQuery != tt.wantQuery {
					t.Fatalf("query = %q, want %q", r.URL.RawQuery, tt.wantQuery)
				}
				if got := r.Header.Get("Authorization"); got != "Bearer cloud-command-token" {
					t.Fatalf("Authorization header = %q", got)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.response))
			}))
			defer server.Close()

			args := append([]string(nil), tt.args...)
			for i, arg := range args {
				if arg == "SERVER_URL" {
					args[i] = server.URL
				}
			}
			var out bytes.Buffer
			tt.cmd.SetOut(&out)
			tt.cmd.SetErr(&out)
			tt.cmd.SetArgs(args)
			if err := tt.cmd.Execute(); err != nil {
				t.Fatalf("%s failed: %v output=%s", tt.name, err, out.String())
			}
			if !called {
				t.Fatalf("%s did not call server", tt.name)
			}
		})
	}
}
