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
	var called bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/notifications" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":"notif-1","type":"approval","channel":"noop","subject":"Approval requested"}]`))
	}))
	defer server.Close()

	cmd := newNotificationListCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--server", server.URL})
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
