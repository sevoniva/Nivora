package executor

import "context"

type HostDeploymentRequest struct {
	DeploymentRunID string
	HostID          string
	HostName        string
	Address         string
	Artifact        string
	DeployPath      string
	ReleaseDir      string
	ServiceName     string
	HealthCheck     string
	Strategy        string
	DryRun          bool
	Apply           bool
	Confirmed       bool
	AllowRemote     bool
	CredentialRef   string
}

type HostDeploymentResult struct {
	HostID   string
	HostName string
	Status   string
	Message  string
	Stdout   string
	Stderr   string
}

type HostExecutor interface {
	Prepare(ctx context.Context, request HostDeploymentRequest) (HostDeploymentResult, error)
	Upload(ctx context.Context, request HostDeploymentRequest) (HostDeploymentResult, error)
	Execute(ctx context.Context, request HostDeploymentRequest) (HostDeploymentResult, error)
	HealthCheck(ctx context.Context, request HostDeploymentRequest) (HostDeploymentResult, error)
	Rollback(ctx context.Context, request HostDeploymentRequest) (HostDeploymentResult, error)
}
