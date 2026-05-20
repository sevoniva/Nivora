package telemetry

import (
	"strings"
	"testing"
	"time"
)

func TestRegistrySnapshotAndPrometheusText(t *testing.T) {
	registry := &Registry{}
	registry.IncPipelineRun()
	registry.IncDeploymentRun()
	registry.IncFailure()
	registry.IncRunnerHeartbeat()
	registry.IncJobClaim()
	registry.IncPolicyDenial()
	registry.ObservePipelineDuration(25 * time.Millisecond)
	registry.ObserveDeploymentDuration(40 * time.Millisecond)
	registry.ObserveQueueTime(15 * time.Millisecond)
	registry.ObserveJobClaimLatency(5 * time.Millisecond)

	snapshot := registry.Snapshot()
	if snapshot.PipelineRunCount != 1 || snapshot.DeploymentRunCount != 1 || snapshot.FailureCount != 1 || snapshot.RunnerHeartbeatCount != 1 || snapshot.JobClaimCount != 1 || snapshot.PolicyDenialCount != 1 {
		t.Fatalf("unexpected snapshot counts: %#v", snapshot)
	}
	if snapshot.PipelineDuration.TotalMS != 25 || snapshot.DeploymentDuration.TotalMS != 40 || snapshot.QueueTime.TotalMS != 15 || snapshot.JobClaimLatency.TotalMS != 5 {
		t.Fatalf("unexpected duration totals: %#v", snapshot)
	}
	text := snapshot.PrometheusText()
	for _, want := range []string{"nivora_pipeline_run_total 1", "nivora_deployment_run_total 1", "nivora_runner_heartbeat_total 1", "nivora_job_claim_total 1", "nivora_policy_denial_total 1", "nivora_queue_time_ms_total 15", "nivora_job_claim_latency_ms_total 5"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in metrics text:\n%s", want, text)
		}
	}
}

func TestMetricsSnapshotDoesNotLeakSecrets(t *testing.T) {
	registry := &Registry{}
	registry.IncPipelineRun()
	snapshot := registry.Snapshot()
	text := snapshot.PrometheusText()

	secretPatterns := []string{"password", "secret", "token", "credential", "DATABASE_URL", "NIVORA_AUTH"}
	for _, pattern := range secretPatterns {
		if strings.Contains(strings.ToLower(text), strings.ToLower(pattern)) {
			t.Fatalf("metrics contain potential secret pattern: %q", pattern)
		}
	}
}

func TestMetricsAllExpectedCountersPresent(t *testing.T) {
	registry := &Registry{}
	// Exercise all counters.
	registry.IncPipelineRun()
	registry.IncDeploymentRun()
	registry.IncFailure()
	registry.IncRunnerHeartbeat()
	registry.IncJobClaim()
	registry.IncPolicyDenial()
	registry.ObservePipelineDuration(10 * time.Millisecond)
	registry.ObserveDeploymentDuration(20 * time.Millisecond)
	registry.ObserveQueueTime(5 * time.Millisecond)
	registry.ObserveJobClaimLatency(3 * time.Millisecond)

	snapshot := registry.Snapshot()
	text := snapshot.PrometheusText()

	expected := []string{
		"nivora_pipeline_run_total",
		"nivora_deployment_run_total",
		"nivora_runtime_failure_total",
		"nivora_runner_heartbeat_total",
		"nivora_job_claim_total",
		"nivora_policy_denial_total",
		"nivora_pipeline_run_duration_ms_total",
		"nivora_deployment_run_duration_ms_total",
		"nivora_queue_time_ms_total",
		"nivora_job_claim_latency_ms_total",
	}
	for _, metric := range expected {
		if !strings.Contains(text, metric) {
			t.Errorf("missing expected metric: %s", metric)
		}
	}
}

func TestMetricsCounterIncrements(t *testing.T) {
	registry := &Registry{}
	registry.IncPipelineRun()
	registry.IncPipelineRun()
	registry.IncPipelineRun()
	snapshot := registry.Snapshot()
	if snapshot.PipelineRunCount != 3 {
		t.Fatalf("expected 3 pipeline runs, got %d", snapshot.PipelineRunCount)
	}
	registry.IncFailure()
	registry.IncFailure()
	snapshot = registry.Snapshot()
	if snapshot.FailureCount != 2 {
		t.Fatalf("expected 2 failures, got %d", snapshot.FailureCount)
	}
	if snapshot.PipelineRunCount != 3 {
		t.Fatalf("pipeline count changed after failure increment: %d", snapshot.PipelineRunCount)
	}
}
