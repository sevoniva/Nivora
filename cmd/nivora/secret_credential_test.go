package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestSecretPutUsesBearerTokenAndDoesNotPrintValue(t *testing.T) {
	t.Setenv("NIVORA_TEST_TOKEN", "secret-api-token")
	t.Setenv("NIVORA_TEST_SECRET_VALUE", "should-not-leak-secret-value")
	var called bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/secrets" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer secret-api-token" {
			t.Fatalf("Authorization header = %q", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if body["value"] != "should-not-leak-secret-value" {
			t.Fatalf("secret value was not read from env: %#v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"sec-1","name":"local-token","provider":"builtin","key":"example/token"}`))
	}))
	defer server.Close()

	cmd := newSecretPutCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{
		"--server", server.URL,
		"--token-env", "NIVORA_TEST_TOKEN",
		"--name", "local-token",
		"--value-env", "NIVORA_TEST_SECRET_VALUE",
		"--key", "example/token",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("secret put failed: %v output=%s", err, out.String())
	}
	if !called {
		t.Fatal("expected secret put to call server")
	}
	if strings.Contains(out.String(), "should-not-leak-secret-value") {
		t.Fatalf("secret value leaked in output: %s", out.String())
	}
}

func TestSecretProtectedCommandsUseBearerToken(t *testing.T) {
	t.Setenv("NIVORA_TEST_TOKEN", "secret-command-token")
	t.Setenv("NIVORA_TEST_SECRET_VALUE", "rotate-value-not-for-output")
	tests := []struct {
		name       string
		cmd        *cobra.Command
		args       []string
		wantMethod string
		wantPath   string
		response   string
	}{
		{
			name:       "rotate",
			cmd:        newSecretRotateCommand(),
			args:       []string{"sec-1", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN", "--value-env", "NIVORA_TEST_SECRET_VALUE"},
			wantMethod: http.MethodPost,
			wantPath:   "/api/v1/secrets/sec-1/rotate",
			response:   `{"id":"sec-1","version":"v2"}`,
		},
		{
			name:       "list",
			cmd:        newSecretListCommand(),
			args:       []string{"--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"},
			wantMethod: http.MethodGet,
			wantPath:   "/api/v1/secrets/refs",
			response:   `[{"id":"sec-1","name":"local-token"}]`,
		},
		{
			name:       "provider validate",
			cmd:        newSecretProviderValidateCommand(),
			args:       []string{"--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"},
			wantMethod: http.MethodPost,
			wantPath:   "/api/v1/secrets/provider/validate",
			response:   `{"valid":true}`,
		},
		{
			name:       "delete",
			cmd:        newSecretDeleteCommand(),
			args:       []string{"sec-1", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"},
			wantMethod: http.MethodDelete,
			wantPath:   "/api/v1/secrets/sec-1",
			response:   `{"deleted":true}`,
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
				if got := r.Header.Get("Authorization"); got != "Bearer secret-command-token" {
					t.Fatalf("Authorization header = %q", got)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.response))
			}))
			defer server.Close()

			args := replaceServerURL(tt.args, server.URL)
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
			if strings.Contains(out.String(), "rotate-value-not-for-output") {
				t.Fatalf("secret value leaked in output: %s", out.String())
			}
		})
	}
}

func TestCredentialCommandsUseBearerToken(t *testing.T) {
	t.Setenv("NIVORA_TEST_TOKEN", "credential-token")
	credentialFile := filepath.Join(t.TempDir(), "credential.yaml")
	content := `apiVersion: nivora.io/v1alpha1
kind: Credential
metadata:
  name: registry-placeholder
spec:
  type: registry
  scopeType: project
  scopeId: project-a
  secretRef:
    provider: builtin
    key: examples/registry/token
  metadata:
    registry: registry.example.invalid
`
	if err := os.WriteFile(credentialFile, []byte(content), 0o600); err != nil {
		t.Fatalf("write credential file: %v", err)
	}

	tests := []struct {
		name       string
		cmd        *cobra.Command
		args       []string
		wantMethod string
		wantPath   string
		response   string
	}{
		{
			name:       "create",
			cmd:        newCredentialCreateCommand(),
			args:       []string{"--file", credentialFile, "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"},
			wantMethod: http.MethodPost,
			wantPath:   "/api/v1/credentials",
			response:   `{"id":"cred-1","name":"registry-placeholder","secretRef":{"provider":"builtin","key":"examples/registry/token"}}`,
		},
		{
			name:       "validate",
			cmd:        newCredentialValidateCommand(),
			args:       []string{"cred-1", "--server", "SERVER_URL", "--token-env", "NIVORA_TEST_TOKEN"},
			wantMethod: http.MethodPost,
			wantPath:   "/api/v1/credentials/cred-1/validate",
			response:   `{"credentialId":"cred-1","valid":true}`,
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
				if got := r.Header.Get("Authorization"); got != "Bearer credential-token" {
					t.Fatalf("Authorization header = %q", got)
				}
				if tt.name == "create" {
					var body map[string]any
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Fatalf("decode credential body: %v", err)
					}
					if _, ok := body["value"]; ok {
						t.Fatalf("credential create request included a secret value: %#v", body)
					}
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.response))
			}))
			defer server.Close()

			args := replaceServerURL(tt.args, server.URL)
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

func replaceServerURL(args []string, serverURL string) []string {
	replaced := append([]string(nil), args...)
	for i, arg := range replaced {
		if arg == "SERVER_URL" {
			replaced[i] = serverURL
		}
	}
	return replaced
}
