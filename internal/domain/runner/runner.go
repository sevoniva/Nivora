package runner

import (
	"strings"
	"time"
)

const (
	ExecutorShell         = "shell"
	ExecutorContainer     = "container"
	ExecutorKubernetesJob = "kubernetes-job"
	ExecutorWebhook       = "webhook"
	ExecutorNoop          = "noop"
	ExecutorExternal      = "external"
)

type Runner struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	GroupID         string            `json:"groupId,omitempty"`
	Status          string            `json:"status"`
	Labels          map[string]string `json:"labels,omitempty"`
	Executors       []string          `json:"executors,omitempty"`
	Capabilities    []string          `json:"capabilities,omitempty"`
	MaxConcurrency  int               `json:"maxConcurrency,omitempty"`
	ActiveJobs      int               `json:"activeJobs,omitempty"`
	TokenID         string            `json:"tokenId,omitempty"`
	TokenHash       string            `json:"-"`
	TokenCreatedAt  *time.Time        `json:"tokenCreatedAt,omitempty"`
	TokenRotatedAt  *time.Time        `json:"tokenRotatedAt,omitempty"`
	TokenRevokedAt  *time.Time        `json:"tokenRevokedAt,omitempty"`
	LastHeartbeatAt *time.Time        `json:"lastHeartbeatAt,omitempty"`
	LastSeenAt      *time.Time        `json:"lastSeenAt,omitempty"`
	CreatedAt       time.Time         `json:"createdAt"`
	UpdatedAt       time.Time         `json:"updatedAt"`
}

type RunnerGroup struct {
	ID             string            `json:"id"`
	ProjectID      string            `json:"projectId,omitempty"`
	EnvironmentIDs []string          `json:"environmentIds,omitempty"`
	Name           string            `json:"name"`
	Labels         map[string]string `json:"labels,omitempty"`
	MaxConcurrency int               `json:"maxConcurrency,omitempty"`
	Executors      []string          `json:"executors,omitempty"`
	CreatedAt      time.Time         `json:"createdAt"`
	UpdatedAt      time.Time         `json:"updatedAt"`
}

func NormalizeExecutorCapability(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "kubernetes_job":
		return ExecutorKubernetesJob
	default:
		return normalized
	}
}

func IsSupportedExecutorCapability(value string) bool {
	switch NormalizeExecutorCapability(value) {
	case ExecutorShell, ExecutorContainer, ExecutorKubernetesJob, ExecutorWebhook, ExecutorNoop, ExecutorExternal:
		return true
	default:
		return false
	}
}
