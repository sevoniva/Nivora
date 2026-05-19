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
