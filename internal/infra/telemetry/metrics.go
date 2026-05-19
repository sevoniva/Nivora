package telemetry

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type Registry struct {
	mu sync.Mutex

	pipelineRunCount     int64
	deploymentRunCount   int64
	failureCount         int64
	runnerHeartbeatCount int64
	jobClaimCount        int64
	policyDenialCount    int64

	pipelineDuration   durationStats
	deploymentDuration durationStats
	queueTime          durationStats
	jobClaimLatency    durationStats
}

type durationStats struct {
	Count   int64 `json:"count"`
	TotalMS int64 `json:"total_ms"`
}

type Snapshot struct {
	PipelineRunCount     int64         `json:"pipeline_run_count"`
	DeploymentRunCount   int64         `json:"deployment_run_count"`
	FailureCount         int64         `json:"failure_count"`
	RunnerHeartbeatCount int64         `json:"runner_heartbeat_count"`
	JobClaimCount        int64         `json:"job_claim_count"`
	PolicyDenialCount    int64         `json:"policy_denial_count"`
	PipelineDuration     durationStats `json:"pipeline_duration"`
	DeploymentDuration   durationStats `json:"deployment_duration"`
	QueueTime            durationStats `json:"queue_time"`
	JobClaimLatency      durationStats `json:"job_claim_latency"`
}

var defaultRegistry = &Registry{}

func DefaultMetrics() *Registry {
	return defaultRegistry
}

func (r *Registry) IncPipelineRun() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pipelineRunCount++
}

func (r *Registry) IncDeploymentRun() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.deploymentRunCount++
}

func (r *Registry) IncFailure() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.failureCount++
}

func (r *Registry) IncRunnerHeartbeat() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.runnerHeartbeatCount++
}

func (r *Registry) IncJobClaim() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.jobClaimCount++
}

func (r *Registry) IncPolicyDenial() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.policyDenialCount++
}

func (r *Registry) ObservePipelineDuration(duration time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pipelineDuration.Count++
	r.pipelineDuration.TotalMS += duration.Milliseconds()
}

func (r *Registry) ObserveDeploymentDuration(duration time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.deploymentDuration.Count++
	r.deploymentDuration.TotalMS += duration.Milliseconds()
}

func (r *Registry) ObserveQueueTime(duration time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.queueTime.Count++
	r.queueTime.TotalMS += duration.Milliseconds()
}

func (r *Registry) ObserveJobClaimLatency(duration time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.jobClaimLatency.Count++
	r.jobClaimLatency.TotalMS += duration.Milliseconds()
}

func (r *Registry) Snapshot() Snapshot {
	r.mu.Lock()
	defer r.mu.Unlock()
	return Snapshot{
		PipelineRunCount:     r.pipelineRunCount,
		DeploymentRunCount:   r.deploymentRunCount,
		FailureCount:         r.failureCount,
		RunnerHeartbeatCount: r.runnerHeartbeatCount,
		JobClaimCount:        r.jobClaimCount,
		PolicyDenialCount:    r.policyDenialCount,
		PipelineDuration:     r.pipelineDuration,
		DeploymentDuration:   r.deploymentDuration,
		QueueTime:            r.queueTime,
		JobClaimLatency:      r.jobClaimLatency,
	}
}

func (s Snapshot) PrometheusText() string {
	var b strings.Builder
	writeCounter(&b, "nivora_pipeline_run_total", "PipelineRuns created through the API.", s.PipelineRunCount)
	writeCounter(&b, "nivora_deployment_run_total", "DeploymentRuns created through the API.", s.DeploymentRunCount)
	writeCounter(&b, "nivora_runtime_failure_total", "Runtime failures observed at API boundaries.", s.FailureCount)
	writeCounter(&b, "nivora_runner_heartbeat_total", "Runner heartbeats observed through the API.", s.RunnerHeartbeatCount)
	writeCounter(&b, "nivora_job_claim_total", "Runner job claim attempts observed through the API.", s.JobClaimCount)
	writeCounter(&b, "nivora_policy_denial_total", "Policy denials observed through security, deployment, or release gates.", s.PolicyDenialCount)
	writeCounter(&b, "nivora_pipeline_run_duration_observations_total", "PipelineRun duration observations.", s.PipelineDuration.Count)
	writeCounter(&b, "nivora_pipeline_run_duration_ms_total", "Total observed PipelineRun duration in milliseconds.", s.PipelineDuration.TotalMS)
	writeCounter(&b, "nivora_deployment_run_duration_observations_total", "DeploymentRun duration observations.", s.DeploymentDuration.Count)
	writeCounter(&b, "nivora_deployment_run_duration_ms_total", "Total observed DeploymentRun duration in milliseconds.", s.DeploymentDuration.TotalMS)
	writeCounter(&b, "nivora_queue_time_observations_total", "Queue time observations for runtime work.", s.QueueTime.Count)
	writeCounter(&b, "nivora_queue_time_ms_total", "Total observed queue time in milliseconds.", s.QueueTime.TotalMS)
	writeCounter(&b, "nivora_job_claim_latency_observations_total", "Job claim latency observations.", s.JobClaimLatency.Count)
	writeCounter(&b, "nivora_job_claim_latency_ms_total", "Total observed job claim latency in milliseconds.", s.JobClaimLatency.TotalMS)
	return b.String()
}

func writeCounter(b *strings.Builder, name string, help string, value int64) {
	fmt.Fprintf(b, "# HELP %s %s\n", name, help)
	fmt.Fprintf(b, "# TYPE %s counter\n", name)
	fmt.Fprintf(b, "%s %d\n", name, value)
}
