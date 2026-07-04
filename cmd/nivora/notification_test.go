package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNotificationCommandIncludesList(t *testing.T) {
	cmd := newNotificationCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("notification help failed: %v", err)
	}
	for _, want := range []string{"list", "test"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("notification help missing %q: %s", want, out.String())
		}
	}
}

func TestNotificationListUsesServerRoute(t *testing.T) {
	t.Setenv("NIVORA_TEST_TOKEN", "notification-token")
	var called bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/notifications" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer notification-token" {
			t.Fatalf("Authorization header = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":"notif-1","type":"approval","channel":"noop","subject":"Approval requested"}]`))
	}))
	defer server.Close()

	cmd := newNotificationListCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--server", server.URL, "--token-env", "NIVORA_TEST_TOKEN"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("notification list failed: %v output=%s", err, out.String())
	}
	if !called {
		t.Fatal("expected notification list to call server")
	}
	if !strings.Contains(out.String(), "notif-1") {
		t.Fatalf("notification list output missing record: %s", out.String())
	}
}

func TestNotificationTestUsesBearerToken(t *testing.T) {
	t.Setenv("NIVORA_TEST_TOKEN", "notification-test-token")
	var called bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/notifications/test" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer notification-test-token" {
			t.Fatalf("Authorization header = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"notif-test","channel":"noop"}`))
	}))
	defer server.Close()

	cmd := newNotificationTestCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--server", server.URL, "--token-env", "NIVORA_TEST_TOKEN", "--channel", "noop"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("notification test failed: %v output=%s", err, out.String())
	}
	if !called {
		t.Fatal("expected notification test to call server")
	}
}
