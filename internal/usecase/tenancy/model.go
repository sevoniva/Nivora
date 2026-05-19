package tenancy

import domaintenant "github.com/sevoniva/nivora/internal/domain/tenant"

type ScopeInput struct {
	ScopeType string `json:"scopeType,omitempty"`
	ScopeID   string `json:"scopeId,omitempty"`
}

type QuotaUpdateInput struct {
	ScopeType                   string `json:"scopeType,omitempty"`
	ScopeID                     string `json:"scopeId,omitempty"`
	MaxConcurrentPipelineRuns   int    `json:"maxConcurrentPipelineRuns,omitempty"`
	MaxConcurrentDeploymentRuns int    `json:"maxConcurrentDeploymentRuns,omitempty"`
	MaxRunners                  int    `json:"maxRunners,omitempty"`
	MaxArtifactsTracked         int    `json:"maxArtifactsTracked,omitempty"`
	MaxLogStorageBytes          int64  `json:"maxLogStorageBytes,omitempty"`
	APITokenRequestsPerMinute   int    `json:"apiTokenRequestsPerMinute,omitempty"`
	RunnerHeartbeatPerMinute    int    `json:"runnerHeartbeatPerMinute,omitempty"`
	JobClaimRequestsPerMinute   int    `json:"jobClaimRequestsPerMinute,omitempty"`
	DeploymentConcurrency       int    `json:"deploymentConcurrency,omitempty"`
	PipelineConcurrency         int    `json:"pipelineConcurrency,omitempty"`
}

type UsageUpdate struct {
	Scope                    domaintenant.Scope
	ConcurrentPipelineRuns   int
	ConcurrentDeploymentRuns int
	Runners                  int
	ArtifactsTracked         int
	LogStorageBytes          int64
}
