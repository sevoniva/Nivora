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

func TestChangeWindowCommandIncludesManagementCommands(t *testing.T) {
	cmd := newChangeWindowCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("change-window help failed: %v", err)
	}
	for _, want := range []string{"list", "create", "get", "evaluate"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("change-window help missing %q: %s", want, out.String())
		}
	}
}

func TestChangeWindowCreateFromFlagsPostsServerRoute(t *testing.T) {
	t.Setenv("NIVORA_TEST_TOKEN", "change-window-token")
	var called bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/change-windows" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer change-window-token" {
			t.Fatalf("Authorization header = %q", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if body["name"] != "prod-business-hours" || body["environmentId"] != "prod" || body["timezone"] != "UTC" {
			t.Fatalf("unexpected body: %#v", body)
		}
		if body["allowed"] != true || body["startTime"] != "09:00" || body["endTime"] != "17:00" {
			t.Fatalf("unexpected body fields: %#v", body)
		}
		days, _ := body["daysOfWeek"].([]any)
		if len(days) != 2 || days[0] != "monday" || days[1] != "tuesday" {
			t.Fatalf("unexpected days: %#v", days)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"cwin-1","name":"prod-business-hours","environmentId":"prod","allowed":true}`))
	}))
	defer server.Close()

	cmd := newChangeWindowCreateCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{
		"--server", server.URL,
		"--token-env", "NIVORA_TEST_TOKEN",
		"--name", "prod-business-hours",
		"--env", "prod",
		"--timezone", "UTC",
		"--start", "09:00",
		"--end", "17:00",
		"--day", "monday",
		"--day", "tuesday",
		"--metadata", "owner=platform",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("change-window create failed: %v output=%s", err, out.String())
	}
	if !called {
		t.Fatal("expected change-window create to call server")
	}
	if !strings.Contains(out.String(), "cwin-1") {
		t.Fatalf("output missing window id: %s", out.String())
	}
}

func TestChangeWindowListAndGetUseServerRoutes(t *testing.T) {
	t.Setenv("NIVORA_TEST_TOKEN", "change-window-read-token")
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		if got := r.Header.Get("Authorization"); got != "Bearer change-window-read-token" {
			t.Fatalf("Authorization header = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/change-windows":
			_, _ = w.Write([]byte(`[{"id":"cwin-1","name":"prod-business-hours"}]`))
		case "/api/v1/change-windows/cwin-1":
			_, _ = w.Write([]byte(`{"id":"cwin-1","name":"prod-business-hours"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	listCmd := newChangeWindowListCommand()
	var out bytes.Buffer
	listCmd.SetOut(&out)
	listCmd.SetErr(&out)
	listCmd.SetArgs([]string{"--server", server.URL, "--token-env", "NIVORA_TEST_TOKEN"})
	if err := listCmd.Execute(); err != nil {
		t.Fatalf("change-window list failed: %v output=%s", err, out.String())
	}

	getCmd := newChangeWindowGetCommand()
	out.Reset()
	getCmd.SetOut(&out)
	getCmd.SetErr(&out)
	getCmd.SetArgs([]string{"cwin-1", "--server", server.URL, "--token-env", "NIVORA_TEST_TOKEN"})
	if err := getCmd.Execute(); err != nil {
		t.Fatalf("change-window get failed: %v output=%s", err, out.String())
	}

	want := []string{"GET /api/v1/change-windows", "GET /api/v1/change-windows/cwin-1"}
	if len(paths) != len(want) {
		t.Fatalf("paths = %#v", paths)
	}
	for i := range want {
		if paths[i] != want[i] {
			t.Fatalf("path[%d] = %q, want %q", i, paths[i], want[i])
		}
	}
}

func TestChangeWindowEvaluateUsesBearerToken(t *testing.T) {
	t.Setenv("NIVORA_TEST_TOKEN", "change-window-evaluate-token")
	var called bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/change-windows/evaluate" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer change-window-evaluate-token" {
			t.Fatalf("Authorization header = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"allowed":true}`))
	}))
	defer server.Close()

	cmd := newChangeWindowEvaluateCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--server", server.URL, "--token-env", "NIVORA_TEST_TOKEN", "--env", "prod"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("change-window evaluate failed: %v output=%s", err, out.String())
	}
	if !called {
		t.Fatal("expected change-window evaluate to call server")
	}
}

func TestChangeWindowCommandsExposeTokenEnv(t *testing.T) {
	for _, tt := range []struct {
		name string
		cmd  *cobra.Command
	}{
		{name: "list", cmd: newChangeWindowListCommand()},
		{name: "create", cmd: newChangeWindowCreateCommand()},
		{name: "get", cmd: newChangeWindowGetCommand()},
		{name: "evaluate", cmd: newChangeWindowEvaluateCommand()},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if flag := tt.cmd.Flags().Lookup("token-env"); flag == nil {
				t.Fatalf("%s missing --token-env flag", tt.name)
			}
		})
	}
}

func TestBuildChangeWindowCreateBodyFromFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "window.yaml")
	content := `apiVersion: nivora.io/v1alpha1
kind: ChangeWindow
metadata:
  name: prod-freeze
spec:
  environmentId: prod
  timezone: UTC
  startTime: "00:00"
  endTime: "23:59"
  daysOfWeek:
    - sunday
  allowed: false
  metadata:
    owner: platform
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	body, err := buildChangeWindowCreateBody(path, "", "", "", "", "", nil, true, nil)
	if err != nil {
		t.Fatalf("build body: %v", err)
	}
	if body["name"] != "prod-freeze" || body["environmentId"] != "prod" || body["allowed"] != false {
		t.Fatalf("unexpected body: %#v", body)
	}
	days, _ := body["daysOfWeek"].([]string)
	if len(days) != 1 || days[0] != "sunday" {
		t.Fatalf("unexpected days: %#v", body["daysOfWeek"])
	}
}

func TestBuildChangeWindowCreateBodyRequiresNameAndEnv(t *testing.T) {
	_, err := buildChangeWindowCreateBody("", "", "prod", "UTC", "", "", nil, true, nil)
	if err == nil || !strings.Contains(err.Error(), "--name is required") {
		t.Fatalf("expected name error, got %v", err)
	}
	_, err = buildChangeWindowCreateBody("", "prod-window", "", "UTC", "", "", nil, true, nil)
	if err == nil || !strings.Contains(err.Error(), "--env is required") {
		t.Fatalf("expected env error, got %v", err)
	}
}
