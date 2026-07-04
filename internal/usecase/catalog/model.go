package catalog

import (
	"context"
	"errors"

	domainapp "github.com/sevoniva/nivora/internal/domain/application"
	domainenv "github.com/sevoniva/nivora/internal/domain/environment"
	domainorg "github.com/sevoniva/nivora/internal/domain/org"
	domainproject "github.com/sevoniva/nivora/internal/domain/project"
)

var (
	ErrInvalid       = errors.New("catalog input is invalid")
	ErrNotFound      = errors.New("catalog resource not found")
	ErrAlreadyExists = errors.New("catalog resource already exists")
)

type Store interface {
	CreateOrg(ctx context.Context, org domainorg.Org) (domainorg.Org, error)
	GetOrg(ctx context.Context, id string) (domainorg.Org, error)
	ListOrgs(ctx context.Context) ([]domainorg.Org, error)
	UpdateOrg(ctx context.Context, org domainorg.Org) (domainorg.Org, error)

	CreateProject(ctx context.Context, project domainproject.Project) (domainproject.Project, error)
	GetProject(ctx context.Context, id string) (domainproject.Project, error)
	ListProjects(ctx context.Context, orgID string) ([]domainproject.Project, error)
	UpdateProject(ctx context.Context, project domainproject.Project) (domainproject.Project, error)

	CreateApplication(ctx context.Context, app domainapp.Application) (domainapp.Application, error)
	GetApplication(ctx context.Context, id string) (domainapp.Application, error)
	ListApplications(ctx context.Context, projectID string) ([]domainapp.Application, error)
	UpdateApplication(ctx context.Context, app domainapp.Application) (domainapp.Application, error)

	CreateEnvironment(ctx context.Context, environment domainenv.Environment) (domainenv.Environment, error)
	GetEnvironment(ctx context.Context, id string) (domainenv.Environment, error)
	ListEnvironments(ctx context.Context, projectID string) ([]domainenv.Environment, error)
	UpdateEnvironment(ctx context.Context, environment domainenv.Environment) (domainenv.Environment, error)

	CreateRepository(ctx context.Context, repository domainapp.Repository) (domainapp.Repository, error)
	GetRepository(ctx context.Context, id string) (domainapp.Repository, error)
	ListRepositories(ctx context.Context, projectID string) ([]domainapp.Repository, error)
	UpdateRepository(ctx context.Context, repository domainapp.Repository) (domainapp.Repository, error)

	CreateReleaseTarget(ctx context.Context, target domainenv.ReleaseTarget) (domainenv.ReleaseTarget, error)
	GetReleaseTarget(ctx context.Context, id string) (domainenv.ReleaseTarget, error)
	ListReleaseTargets(ctx context.Context, projectID string, environmentID string) ([]domainenv.ReleaseTarget, error)
	UpdateReleaseTarget(ctx context.Context, target domainenv.ReleaseTarget) (domainenv.ReleaseTarget, error)
}

type CreateOrgInput struct {
	ID          string            `json:"id,omitempty" yaml:"id,omitempty"`
	Name        string            `json:"name" yaml:"name"`
	Slug        string            `json:"slug,omitempty" yaml:"slug,omitempty"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Enabled     *bool             `json:"enabled,omitempty" yaml:"enabled,omitempty"`
}

type UpdateOrgInput struct {
	Name        *string           `json:"name,omitempty" yaml:"name,omitempty"`
	Slug        *string           `json:"slug,omitempty" yaml:"slug,omitempty"`
	Description *string           `json:"description,omitempty" yaml:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Enabled     *bool             `json:"enabled,omitempty" yaml:"enabled,omitempty"`
}

type CreateProjectInput struct {
	ID          string            `json:"id,omitempty" yaml:"id,omitempty"`
	OrgID       string            `json:"orgId" yaml:"orgId"`
	Name        string            `json:"name" yaml:"name"`
	Slug        string            `json:"slug,omitempty" yaml:"slug,omitempty"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Enabled     *bool             `json:"enabled,omitempty" yaml:"enabled,omitempty"`
}

type UpdateProjectInput struct {
	Name        *string           `json:"name,omitempty" yaml:"name,omitempty"`
	Slug        *string           `json:"slug,omitempty" yaml:"slug,omitempty"`
	Description *string           `json:"description,omitempty" yaml:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Enabled     *bool             `json:"enabled,omitempty" yaml:"enabled,omitempty"`
}

type CreateApplicationInput struct {
	ID          string            `json:"id,omitempty" yaml:"id,omitempty"`
	ProjectID   string            `json:"projectId" yaml:"projectId"`
	Name        string            `json:"name" yaml:"name"`
	Slug        string            `json:"slug,omitempty" yaml:"slug,omitempty"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Enabled     *bool             `json:"enabled,omitempty" yaml:"enabled,omitempty"`
}

type UpdateApplicationInput struct {
	Name        *string           `json:"name,omitempty" yaml:"name,omitempty"`
	Slug        *string           `json:"slug,omitempty" yaml:"slug,omitempty"`
	Description *string           `json:"description,omitempty" yaml:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Enabled     *bool             `json:"enabled,omitempty" yaml:"enabled,omitempty"`
}

type CreateEnvironmentInput struct {
	ID          string            `json:"id,omitempty" yaml:"id,omitempty"`
	ProjectID   string            `json:"projectId" yaml:"projectId"`
	Name        string            `json:"name" yaml:"name"`
	Slug        string            `json:"slug,omitempty" yaml:"slug,omitempty"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Enabled     *bool             `json:"enabled,omitempty" yaml:"enabled,omitempty"`
}

type UpdateEnvironmentInput struct {
	Name        *string           `json:"name,omitempty" yaml:"name,omitempty"`
	Slug        *string           `json:"slug,omitempty" yaml:"slug,omitempty"`
	Description *string           `json:"description,omitempty" yaml:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Enabled     *bool             `json:"enabled,omitempty" yaml:"enabled,omitempty"`
}

type CreateRepositoryInput struct {
	ID            string            `json:"id,omitempty" yaml:"id,omitempty"`
	ProjectID     string            `json:"projectId" yaml:"projectId"`
	Name          string            `json:"name" yaml:"name"`
	URL           string            `json:"url" yaml:"url"`
	Provider      string            `json:"provider,omitempty" yaml:"provider,omitempty"`
	DefaultBranch string            `json:"defaultBranch,omitempty" yaml:"defaultBranch,omitempty"`
	CredentialRef string            `json:"credentialRef,omitempty" yaml:"credentialRef,omitempty"`
	Labels        map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Enabled       *bool             `json:"enabled,omitempty" yaml:"enabled,omitempty"`
}

type UpdateRepositoryInput struct {
	Name          *string           `json:"name,omitempty" yaml:"name,omitempty"`
	URL           *string           `json:"url,omitempty" yaml:"url,omitempty"`
	Provider      *string           `json:"provider,omitempty" yaml:"provider,omitempty"`
	DefaultBranch *string           `json:"defaultBranch,omitempty" yaml:"defaultBranch,omitempty"`
	CredentialRef *string           `json:"credentialRef,omitempty" yaml:"credentialRef,omitempty"`
	Labels        map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Enabled       *bool             `json:"enabled,omitempty" yaml:"enabled,omitempty"`
}

type CreateReleaseTargetInput struct {
	ID                    string            `json:"id,omitempty" yaml:"id,omitempty"`
	ProjectID             string            `json:"projectId,omitempty" yaml:"projectId,omitempty"`
	EnvironmentID         string            `json:"environmentId" yaml:"environmentId"`
	Name                  string            `json:"name" yaml:"name"`
	TargetType            string            `json:"targetType" yaml:"targetType"`
	Context               string            `json:"context,omitempty" yaml:"context,omitempty"`
	Namespace             string            `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	ConfigRef             string            `json:"configRef,omitempty" yaml:"configRef,omitempty"`
	CredentialRef         string            `json:"credentialRef,omitempty" yaml:"credentialRef,omitempty"`
	Labels                map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Metadata              map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	AllowApply            *bool             `json:"allowApply,omitempty" yaml:"allowApply,omitempty"`
	AllowSync             *bool             `json:"allowSync,omitempty" yaml:"allowSync,omitempty"`
	AllowRemoteHostDeploy *bool             `json:"allowRemoteHostDeploy,omitempty" yaml:"allowRemoteHostDeploy,omitempty"`
	Enabled               *bool             `json:"enabled,omitempty" yaml:"enabled,omitempty"`
}

type UpdateReleaseTargetInput struct {
	Name                  *string           `json:"name,omitempty" yaml:"name,omitempty"`
	TargetType            *string           `json:"targetType,omitempty" yaml:"targetType,omitempty"`
	Context               *string           `json:"context,omitempty" yaml:"context,omitempty"`
	Namespace             *string           `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	ConfigRef             *string           `json:"configRef,omitempty" yaml:"configRef,omitempty"`
	CredentialRef         *string           `json:"credentialRef,omitempty" yaml:"credentialRef,omitempty"`
	Labels                map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Metadata              map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	AllowApply            *bool             `json:"allowApply,omitempty" yaml:"allowApply,omitempty"`
	AllowSync             *bool             `json:"allowSync,omitempty" yaml:"allowSync,omitempty"`
	AllowRemoteHostDeploy *bool             `json:"allowRemoteHostDeploy,omitempty" yaml:"allowRemoteHostDeploy,omitempty"`
	Enabled               *bool             `json:"enabled,omitempty" yaml:"enabled,omitempty"`
}

type ReleaseTargetValidationResult struct {
	Valid    bool     `json:"valid"`
	TargetID string   `json:"targetId,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
	Errors   []string `json:"errors,omitempty"`
}
