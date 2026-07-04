package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestQuotaCommandIncludesViewAndSet(t *testing.T) {
	cmd := newQuotaCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("quota help failed: %v", err)
	}
	for _, want := range []string{"view", "set"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("quota help missing %q: %s", want, out.String())
		}
	}
}

func TestQuotaSetPostsScopeAndLimits(t *testing.T) {
	t.Setenv("NIVORA_TEST_TOKEN", "opaque-token")
	var called bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/tenancy/quota" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer opaque-token" {
			t.Fatalf("unexpected authorization header %q", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		wantNumbers := map[string]float64{
			"maxConcurrentPipelineRuns":   3,
			"maxConcurrentDeploymentRuns": 2,
			"maxRunners":                  4,
			"maxArtifactsTracked":         50,
			"maxLogStorageBytes":          1048576,
			"apiTokenRequestsPerMinute":   60,
			"runnerHeartbeatPerMinute":    120,
			"jobClaimRequestsPerMinute":   30,
			"deploymentConcurrency":       2,
			"pipelineConcurrency":         3,
		}
		if body["scopeType"] != "project" || body["scopeId"] != "demo" {
			t.Fatalf("unexpected scope body: %#v", body)
		}
		for key, want := range wantNumbers {
			if body[key] != want {
				t.Fatalf("body[%s] = %#v, want %v; body=%#v", key, body[key], want, body)
			}
		}
		raw, _ := json.Marshal(body)
		for _, forbidden := range []string{"opaque-token", "password", "secret", "privateKey", "kubeconfig", "authorization"} {
			if strings.Contains(strings.ToLower(string(raw)), strings.ToLower(forbidden)) {
				t.Fatalf("quota request leaked forbidden marker %q: %s", forbidden, raw)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"scope":{"type":"project","id":"demo"},"maxConcurrentPipelineRuns":3,"maxConcurrentDeploymentRuns":2}`))
	}))
	defer server.Close()

	cmd := newQuotaSetCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{
		"--server", server.URL,
		"--token-env", "NIVORA_TEST_TOKEN",
		"--scope-type", "project",
		"--scope-id", "demo",
		"--max-concurrent-pipeline-runs", "3",
		"--max-concurrent-deployment-runs", "2",
		"--max-runners", "4",
		"--max-artifacts-tracked", "50",
		"--max-log-storage-bytes", "1048576",
		"--api-token-requests-per-minute", "60",
		"--runner-heartbeat-per-minute", "120",
		"--job-claim-requests-per-minute", "30",
		"--deployment-concurrency", "2",
		"--pipeline-concurrency", "3",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("quota set failed: %v output=%s", err, out.String())
	}
	if !called {
		t.Fatal("expected quota set to call server")
	}
	if !strings.Contains(out.String(), "maxConcurrentPipelineRuns") {
		t.Fatalf("quota set output missing quota payload: %s", out.String())
	}
}
