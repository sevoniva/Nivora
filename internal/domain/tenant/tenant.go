package tenant

import "time"

const (
	ScopeOrg         = "org"
	ScopeProject     = "project"
	ScopeApplication = "application"
	ScopeEnvironment = "environment"
	ScopeGlobal      = "global"

	RateLimitAPIToken        = "api_token"
	RateLimitRunnerHeartbeat = "runner_heartbeat"
	RateLimitJobClaim        = "job_claim"
)

type Scope struct {
	Type string `json:"type" yaml:"type"`
	ID   string `json:"id,omitempty" yaml:"id,omitempty"`
}

type Quota struct {
	Scope                       Scope     `json:"scope" yaml:"scope"`
	MaxConcurrentPipelineRuns   int       `json:"maxConcurrentPipelineRuns" yaml:"maxConcurrentPipelineRuns"`
	MaxConcurrentDeploymentRuns int       `json:"maxConcurrentDeploymentRuns" yaml:"maxConcurrentDeploymentRuns"`
	MaxRunners                  int       `json:"maxRunners" yaml:"maxRunners"`
	MaxArtifactsTracked         int       `json:"maxArtifactsTracked" yaml:"maxArtifactsTracked"`
	MaxLogStorageBytes          int64     `json:"maxLogStorageBytes" yaml:"maxLogStorageBytes"`
	APITokenRequestsPerMinute   int       `json:"apiTokenRequestsPerMinute" yaml:"apiTokenRequestsPerMinute"`
	RunnerHeartbeatPerMinute    int       `json:"runnerHeartbeatPerMinute" yaml:"runnerHeartbeatPerMinute"`
	JobClaimRequestsPerMinute   int       `json:"jobClaimRequestsPerMinute" yaml:"jobClaimRequestsPerMinute"`
	DeploymentConcurrency       int       `json:"deploymentConcurrency" yaml:"deploymentConcurrency"`
	PipelineConcurrency         int       `json:"pipelineConcurrency" yaml:"pipelineConcurrency"`
	UpdatedAt                   time.Time `json:"updatedAt" yaml:"updatedAt"`
}

type UsageSummary struct {
	Scope                    Scope     `json:"scope" yaml:"scope"`
	ConcurrentPipelineRuns   int       `json:"concurrentPipelineRuns" yaml:"concurrentPipelineRuns"`
	ConcurrentDeploymentRuns int       `json:"concurrentDeploymentRuns" yaml:"concurrentDeploymentRuns"`
	Runners                  int       `json:"runners" yaml:"runners"`
	ArtifactsTracked         int       `json:"artifactsTracked" yaml:"artifactsTracked"`
	LogStorageBytes          int64     `json:"logStorageBytes" yaml:"logStorageBytes"`
	UpdatedAt                time.Time `json:"updatedAt" yaml:"updatedAt"`
}

type IsolationDecision struct {
	Allowed bool   `json:"allowed" yaml:"allowed"`
	Reason  string `json:"reason" yaml:"reason"`
}

type QuotaCheck struct {
	Allowed bool   `json:"allowed" yaml:"allowed"`
	Reason  string `json:"reason,omitempty" yaml:"reason,omitempty"`
	Limit   int64  `json:"limit,omitempty" yaml:"limit,omitempty"`
	Used    int64  `json:"used,omitempty" yaml:"used,omitempty"`
}
