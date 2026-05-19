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
	registry.ObservePipelineDuration(25 * time.Millisecond)
	registry.ObserveDeploymentDuration(40 * time.Millisecond)

	snapshot := registry.Snapshot()
	if snapshot.PipelineRunCount != 1 || snapshot.DeploymentRunCount != 1 || snapshot.FailureCount != 1 || snapshot.RunnerHeartbeatCount != 1 {
		t.Fatalf("unexpected snapshot counts: %#v", snapshot)
	}
	if snapshot.PipelineDuration.TotalMS != 25 || snapshot.DeploymentDuration.TotalMS != 40 {
		t.Fatalf("unexpected duration totals: %#v", snapshot)
	}
	text := snapshot.PrometheusText()
	for _, want := range []string{"nivora_pipeline_run_total 1", "nivora_deployment_run_total 1", "nivora_runner_heartbeat_total 1"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in metrics text:\n%s", want, text)
		}
	}
}
