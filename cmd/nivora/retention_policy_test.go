package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRetentionPolicyCommandIncludesGetAndSet(t *testing.T) {
	cmd := newRetentionPolicyCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("retention-policy help failed: %v", err)
	}
	for _, want := range []string{"get", "set"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("retention-policy help missing %q: %s", want, out.String())
		}
	}
}

func TestRetentionPolicyGetUsesScopedRoute(t *testing.T) {
	var called bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/retention-policy" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("scopeType") != "project" || r.URL.Query().Get("scopeId") != "demo" {
			t.Fatalf("unexpected query: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"scopeType":"project","scopeId":"demo","logDays":30,"immutableAudit":true}`))
	}))
	defer server.Close()

	cmd := newRetentionPolicyGetCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--server", server.URL, "--scope-type", "project", "--scope-id", "demo"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("retention-policy get failed: %v output=%s", err, out.String())
	}
	if !called {
		t.Fatal("expected retention-policy get to call server")
	}
	if !strings.Contains(out.String(), "logDays") {
		t.Fatalf("retention-policy get output missing payload: %s", out.String())
	}
}

func TestRetentionPolicySetPostsMetadataOnly(t *testing.T) {
	var called bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/retention-policy" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if body["scopeType"] != "project" || body["scopeId"] != "demo" || body["immutableAudit"] != true {
			t.Fatalf("unexpected body: %#v", body)
		}
		if body["logDays"] != float64(14) || body["auditDays"] != float64(365) || body["eventDays"] != float64(30) || body["evidenceDays"] != float64(730) {
			t.Fatalf("unexpected day fields: %#v", body)
		}
		raw, _ := json.Marshal(body)
		for _, forbidden := range []string{"token", "password", "secret", "privateKey", "kubeconfig"} {
			if strings.Contains(string(raw), forbidden) {
				t.Fatalf("retention request leaked forbidden marker %q: %s", forbidden, raw)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"scopeType":"project","scopeId":"demo","logDays":14,"auditDays":365,"eventDays":30,"evidenceDays":730,"immutableAudit":true}`))
	}))
	defer server.Close()

	cmd := newRetentionPolicySetCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{
		"--server", server.URL,
		"--scope-type", "project",
		"--scope-id", "demo",
		"--log-days", "14",
		"--audit-days", "365",
		"--event-days", "30",
		"--evidence-days", "730",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("retention-policy set failed: %v output=%s", err, out.String())
	}
	if !called {
		t.Fatal("expected retention-policy set to call server")
	}
	if !strings.Contains(out.String(), "immutableAudit") {
		t.Fatalf("retention-policy set output missing payload: %s", out.String())
	}
}
