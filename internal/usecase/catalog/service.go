package catalog

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"time"

	domainapp "github.com/sevoniva/nivora/internal/domain/application"
	domainenv "github.com/sevoniva/nivora/internal/domain/environment"
	domainorg "github.com/sevoniva/nivora/internal/domain/org"
	domainproject "github.com/sevoniva/nivora/internal/domain/project"
)

type Service struct {
	store Store
	now   func() time.Time
}

func NewService(store Store) *Service {
	return &Service{store: store, now: time.Now}
}

func (s *Service) CreateOrg(ctx context.Context, input CreateOrgInput) (domainorg.Org, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return domainorg.Org{}, fmt.Errorf("%w: org name is required", ErrInvalid)
	}
	if exists, _ := s.findOrgByNameOrSlug(ctx, name, input.Slug); exists.ID != "" {
		return domainorg.Org{}, fmt.Errorf("%w: org %q", ErrAlreadyExists, name)
	}
	now := s.now().UTC()
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	org := domainorg.Org{
		ID:          defaultID(input.ID, "org"),
		Name:        name,
		Slug:        defaultSlug(input.Slug, name),
		Description: strings.TrimSpace(input.Description),
		Labels:      copyMap(input.Labels),
		Metadata:    copyMap(input.Metadata),
		Enabled:     enabled,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	return s.store.CreateOrg(ctx, org)
}

func (s *Service) GetOrg(ctx context.Context, id string) (domainorg.Org, error) {
	return s.store.GetOrg(ctx, strings.TrimSpace(id))
}

func (s *Service) ListOrgs(ctx context.Context) ([]domainorg.Org, error) {
	return s.store.ListOrgs(ctx)
}

func (s *Service) UpdateOrg(ctx context.Context, id string, input UpdateOrgInput) (domainorg.Org, error) {
	org, err := s.store.GetOrg(ctx, strings.TrimSpace(id))
	if err != nil {
		return domainorg.Org{}, err
	}
	if input.Name != nil {
		org.Name = strings.TrimSpace(*input.Name)
		if org.Name == "" {
			return domainorg.Org{}, fmt.Errorf("%w: org name is required", ErrInvalid)
		}
	}
	if input.Slug != nil {
		org.Slug = defaultSlug(*input.Slug, org.Name)
	}
	if input.Description != nil {
		org.Description = strings.TrimSpace(*input.Description)
	}
	if input.Labels != nil {
		org.Labels = copyMap(input.Labels)
	}
	if input.Metadata != nil {
		org.Metadata = copyMap(input.Metadata)
	}
	if input.Enabled != nil {
		org.Enabled = *input.Enabled
	}
	org.UpdatedAt = s.now().UTC()
	return s.store.UpdateOrg(ctx, org)
}

func (s *Service) DisableOrg(ctx context.Context, id string) (domainorg.Org, error) {
	disabled := false
	return s.UpdateOrg(ctx, id, UpdateOrgInput{Enabled: &disabled})
}

func (s *Service) CreateProject(ctx context.Context, input CreateProjectInput) (domainproject.Project, error) {
	orgID := strings.TrimSpace(input.OrgID)
	if orgID == "" {
		return domainproject.Project{}, fmt.Errorf("%w: project orgId is required", ErrInvalid)
	}
	if _, err := s.store.GetOrg(ctx, orgID); err != nil {
		return domainproject.Project{}, fmt.Errorf("%w: org %q", ErrNotFound, orgID)
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return domainproject.Project{}, fmt.Errorf("%w: project name is required", ErrInvalid)
	}
	projects, _ := s.store.ListProjects(ctx, orgID)
	for _, existing := range projects {
		if sameIdentity(existing.Name, existing.Slug, name, input.Slug) {
			return domainproject.Project{}, fmt.Errorf("%w: project %q", ErrAlreadyExists, name)
		}
	}
	now := s.now().UTC()
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	project := domainproject.Project{
		ID:          defaultID(input.ID, "project"),
		OrgID:       orgID,
		Name:        name,
		Slug:        defaultSlug(input.Slug, name),
		Description: strings.TrimSpace(input.Description),
		Labels:      copyMap(input.Labels),
		Metadata:    copyMap(input.Metadata),
		Enabled:     enabled,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	return s.store.CreateProject(ctx, project)
}

func (s *Service) GetProject(ctx context.Context, id string) (domainproject.Project, error) {
	return s.store.GetProject(ctx, strings.TrimSpace(id))
}

func (s *Service) ListProjects(ctx context.Context, orgID string) ([]domainproject.Project, error) {
	return s.store.ListProjects(ctx, strings.TrimSpace(orgID))
}

func (s *Service) UpdateProject(ctx context.Context, id string, input UpdateProjectInput) (domainproject.Project, error) {
	project, err := s.store.GetProject(ctx, strings.TrimSpace(id))
	if err != nil {
		return domainproject.Project{}, err
	}
	if input.Name != nil {
		project.Name = strings.TrimSpace(*input.Name)
		if project.Name == "" {
			return domainproject.Project{}, fmt.Errorf("%w: project name is required", ErrInvalid)
		}
	}
	if input.Slug != nil {
		project.Slug = defaultSlug(*input.Slug, project.Name)
	}
	if input.Description != nil {
		project.Description = strings.TrimSpace(*input.Description)
	}
	if input.Labels != nil {
		project.Labels = copyMap(input.Labels)
	}
	if input.Metadata != nil {
		project.Metadata = copyMap(input.Metadata)
	}
	if input.Enabled != nil {
		project.Enabled = *input.Enabled
	}
	project.UpdatedAt = s.now().UTC()
	return s.store.UpdateProject(ctx, project)
}

func (s *Service) DisableProject(ctx context.Context, id string) (domainproject.Project, error) {
	disabled := false
	return s.UpdateProject(ctx, id, UpdateProjectInput{Enabled: &disabled})
}

func (s *Service) CreateApplication(ctx context.Context, input CreateApplicationInput) (domainapp.Application, error) {
	projectID := strings.TrimSpace(input.ProjectID)
	if projectID == "" {
		return domainapp.Application{}, fmt.Errorf("%w: application projectId is required", ErrInvalid)
	}
	if _, err := s.store.GetProject(ctx, projectID); err != nil {
		return domainapp.Application{}, fmt.Errorf("%w: project %q", ErrNotFound, projectID)
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return domainapp.Application{}, fmt.Errorf("%w: application name is required", ErrInvalid)
	}
	apps, _ := s.store.ListApplications(ctx, projectID)
	for _, existing := range apps {
		if sameIdentity(existing.Name, existing.Slug, name, input.Slug) {
			return domainapp.Application{}, fmt.Errorf("%w: application %q", ErrAlreadyExists, name)
		}
	}
	now := s.now().UTC()
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	app := domainapp.Application{
		ID:          defaultID(input.ID, "app"),
		ProjectID:   projectID,
		Name:        name,
		Slug:        defaultSlug(input.Slug, name),
		Description: strings.TrimSpace(input.Description),
		Labels:      copyMap(input.Labels),
		Metadata:    copyMap(input.Metadata),
		Enabled:     enabled,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	return s.store.CreateApplication(ctx, app)
}

func (s *Service) GetApplication(ctx context.Context, id string) (domainapp.Application, error) {
	return s.store.GetApplication(ctx, strings.TrimSpace(id))
}

func (s *Service) ListApplications(ctx context.Context, projectID string) ([]domainapp.Application, error) {
	return s.store.ListApplications(ctx, strings.TrimSpace(projectID))
}

func (s *Service) UpdateApplication(ctx context.Context, id string, input UpdateApplicationInput) (domainapp.Application, error) {
	app, err := s.store.GetApplication(ctx, strings.TrimSpace(id))
	if err != nil {
		return domainapp.Application{}, err
	}
	if input.Name != nil {
		app.Name = strings.TrimSpace(*input.Name)
		if app.Name == "" {
			return domainapp.Application{}, fmt.Errorf("%w: application name is required", ErrInvalid)
		}
	}
	if input.Slug != nil {
		app.Slug = defaultSlug(*input.Slug, app.Name)
	}
	if input.Description != nil {
		app.Description = strings.TrimSpace(*input.Description)
	}
	if input.Labels != nil {
		app.Labels = copyMap(input.Labels)
	}
	if input.Metadata != nil {
		app.Metadata = copyMap(input.Metadata)
	}
	if input.Enabled != nil {
		app.Enabled = *input.Enabled
	}
	app.UpdatedAt = s.now().UTC()
	return s.store.UpdateApplication(ctx, app)
}

func (s *Service) DisableApplication(ctx context.Context, id string) (domainapp.Application, error) {
	disabled := false
	return s.UpdateApplication(ctx, id, UpdateApplicationInput{Enabled: &disabled})
}

func (s *Service) CreateEnvironment(ctx context.Context, input CreateEnvironmentInput) (domainenv.Environment, error) {
	projectID := strings.TrimSpace(input.ProjectID)
	if projectID == "" {
		return domainenv.Environment{}, fmt.Errorf("%w: environment projectId is required", ErrInvalid)
	}
	if _, err := s.store.GetProject(ctx, projectID); err != nil {
		return domainenv.Environment{}, fmt.Errorf("%w: project %q", ErrNotFound, projectID)
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return domainenv.Environment{}, fmt.Errorf("%w: environment name is required", ErrInvalid)
	}
	environments, _ := s.store.ListEnvironments(ctx, projectID)
	for _, existing := range environments {
		if sameIdentity(existing.Name, existing.Slug, name, input.Slug) {
			return domainenv.Environment{}, fmt.Errorf("%w: environment %q", ErrAlreadyExists, name)
		}
	}
	now := s.now().UTC()
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	environment := domainenv.Environment{
		ID:          defaultID(input.ID, "env"),
		ProjectID:   projectID,
		Name:        name,
		Slug:        defaultSlug(input.Slug, name),
		Description: strings.TrimSpace(input.Description),
		Labels:      copyMap(input.Labels),
		Metadata:    copyMap(input.Metadata),
		Enabled:     enabled,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	return s.store.CreateEnvironment(ctx, environment)
}

func (s *Service) GetEnvironment(ctx context.Context, id string) (domainenv.Environment, error) {
	return s.store.GetEnvironment(ctx, strings.TrimSpace(id))
}

func (s *Service) ListEnvironments(ctx context.Context, projectID string) ([]domainenv.Environment, error) {
	return s.store.ListEnvironments(ctx, strings.TrimSpace(projectID))
}

func (s *Service) UpdateEnvironment(ctx context.Context, id string, input UpdateEnvironmentInput) (domainenv.Environment, error) {
	environment, err := s.store.GetEnvironment(ctx, strings.TrimSpace(id))
	if err != nil {
		return domainenv.Environment{}, err
	}
	if input.Name != nil {
		environment.Name = strings.TrimSpace(*input.Name)
		if environment.Name == "" {
			return domainenv.Environment{}, fmt.Errorf("%w: environment name is required", ErrInvalid)
		}
	}
	if input.Slug != nil {
		environment.Slug = defaultSlug(*input.Slug, environment.Name)
	}
	if input.Description != nil {
		environment.Description = strings.TrimSpace(*input.Description)
	}
	if input.Labels != nil {
		environment.Labels = copyMap(input.Labels)
	}
	if input.Metadata != nil {
		environment.Metadata = copyMap(input.Metadata)
	}
	if input.Enabled != nil {
		environment.Enabled = *input.Enabled
	}
	environment.UpdatedAt = s.now().UTC()
	return s.store.UpdateEnvironment(ctx, environment)
}

func (s *Service) DisableEnvironment(ctx context.Context, id string) (domainenv.Environment, error) {
	disabled := false
	return s.UpdateEnvironment(ctx, id, UpdateEnvironmentInput{Enabled: &disabled})
}

func (s *Service) CreateRepository(ctx context.Context, input CreateRepositoryInput) (domainapp.Repository, error) {
	projectID := strings.TrimSpace(input.ProjectID)
	if projectID == "" {
		return domainapp.Repository{}, fmt.Errorf("%w: repository projectId is required", ErrInvalid)
	}
	if _, err := s.store.GetProject(ctx, projectID); err != nil {
		return domainapp.Repository{}, fmt.Errorf("%w: project %q", ErrNotFound, projectID)
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return domainapp.Repository{}, fmt.Errorf("%w: repository name is required", ErrInvalid)
	}
	provider := defaultProvider(input.Provider)
	if err := validateRepositoryProvider(provider); err != nil {
		return domainapp.Repository{}, err
	}
	repoURL := strings.TrimSpace(input.URL)
	if err := validateRepositoryURL(repoURL, provider); err != nil {
		return domainapp.Repository{}, err
	}
	repositories, _ := s.store.ListRepositories(ctx, projectID)
	for _, existing := range repositories {
		if strings.EqualFold(existing.Name, name) || strings.EqualFold(existing.URL, repoURL) {
			return domainapp.Repository{}, fmt.Errorf("%w: repository %q", ErrAlreadyExists, name)
		}
	}
	now := s.now().UTC()
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	repository := domainapp.Repository{
		ID:            defaultID(input.ID, "repo"),
		ProjectID:     projectID,
		Name:          name,
		URL:           repoURL,
		Provider:      provider,
		DefaultBranch: defaultBranch(input.DefaultBranch),
		CredentialRef: strings.TrimSpace(input.CredentialRef),
		Labels:        copyMap(input.Labels),
		Metadata:      copyMap(input.Metadata),
		Enabled:       enabled,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	return s.store.CreateRepository(ctx, repository)
}

func (s *Service) GetRepository(ctx context.Context, id string) (domainapp.Repository, error) {
	return s.store.GetRepository(ctx, strings.TrimSpace(id))
}

func (s *Service) ListRepositories(ctx context.Context, projectID string) ([]domainapp.Repository, error) {
	return s.store.ListRepositories(ctx, strings.TrimSpace(projectID))
}

func (s *Service) UpdateRepository(ctx context.Context, id string, input UpdateRepositoryInput) (domainapp.Repository, error) {
	repository, err := s.store.GetRepository(ctx, strings.TrimSpace(id))
	if err != nil {
		return domainapp.Repository{}, err
	}
	if input.Name != nil {
		repository.Name = strings.TrimSpace(*input.Name)
		if repository.Name == "" {
			return domainapp.Repository{}, fmt.Errorf("%w: repository name is required", ErrInvalid)
		}
	}
	if input.URL != nil {
		repository.URL = strings.TrimSpace(*input.URL)
	}
	if input.Provider != nil {
		repository.Provider = defaultProvider(*input.Provider)
	}
	if err := validateRepositoryProvider(repository.Provider); err != nil {
		return domainapp.Repository{}, err
	}
	if err := validateRepositoryURL(repository.URL, repository.Provider); err != nil {
		return domainapp.Repository{}, err
	}
	if input.DefaultBranch != nil {
		repository.DefaultBranch = defaultBranch(*input.DefaultBranch)
	}
	if input.CredentialRef != nil {
		repository.CredentialRef = strings.TrimSpace(*input.CredentialRef)
	}
	if input.Labels != nil {
		repository.Labels = copyMap(input.Labels)
	}
	if input.Metadata != nil {
		repository.Metadata = copyMap(input.Metadata)
	}
	if input.Enabled != nil {
		repository.Enabled = *input.Enabled
	}
	repository.UpdatedAt = s.now().UTC()
	return s.store.UpdateRepository(ctx, repository)
}

func (s *Service) DisableRepository(ctx context.Context, id string) (domainapp.Repository, error) {
	disabled := false
	return s.UpdateRepository(ctx, id, UpdateRepositoryInput{Enabled: &disabled})
}

func (s *Service) ValidateRepository(ctx context.Context, id string) (RepositoryValidationResult, error) {
	repository, err := s.store.GetRepository(ctx, strings.TrimSpace(id))
	if err != nil {
		return RepositoryValidationResult{}, err
	}
	return validateRepository(repository), nil
}

func (s *Service) CreateReleaseTarget(ctx context.Context, input CreateReleaseTargetInput) (domainenv.ReleaseTarget, error) {
	environmentID := strings.TrimSpace(input.EnvironmentID)
	if environmentID == "" {
		return domainenv.ReleaseTarget{}, fmt.Errorf("%w: release target environmentId is required", ErrInvalid)
	}
	environment, err := s.store.GetEnvironment(ctx, environmentID)
	if err != nil {
		return domainenv.ReleaseTarget{}, fmt.Errorf("%w: environment %q", ErrNotFound, environmentID)
	}
	projectID := strings.TrimSpace(input.ProjectID)
	if projectID == "" {
		projectID = environment.ProjectID
	}
	if projectID != environment.ProjectID {
		return domainenv.ReleaseTarget{}, fmt.Errorf("%w: release target projectId must match parent environment projectId", ErrInvalid)
	}
	if _, err := s.store.GetProject(ctx, projectID); err != nil {
		return domainenv.ReleaseTarget{}, fmt.Errorf("%w: project %q", ErrNotFound, projectID)
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return domainenv.ReleaseTarget{}, fmt.Errorf("%w: release target name is required", ErrInvalid)
	}
	targetType := normalizeTargetType(input.TargetType)
	if err := validateTargetType(targetType); err != nil {
		return domainenv.ReleaseTarget{}, err
	}
	targets, _ := s.store.ListReleaseTargets(ctx, projectID, environmentID)
	for _, existing := range targets {
		if strings.EqualFold(existing.Name, name) {
			return domainenv.ReleaseTarget{}, fmt.Errorf("%w: release target %q", ErrAlreadyExists, name)
		}
	}
	now := s.now().UTC()
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	target := domainenv.ReleaseTarget{
		ID:                    defaultID(input.ID, "target"),
		ProjectID:             projectID,
		EnvironmentID:         environmentID,
		Name:                  name,
		TargetType:            targetType,
		Context:               strings.TrimSpace(input.Context),
		Namespace:             strings.TrimSpace(input.Namespace),
		ConfigRef:             strings.TrimSpace(input.ConfigRef),
		CredentialRef:         strings.TrimSpace(input.CredentialRef),
		Labels:                copyMap(input.Labels),
		Metadata:              copyMap(input.Metadata),
		AllowApply:            boolValue(input.AllowApply),
		AllowSync:             boolValue(input.AllowSync),
		AllowRemoteHostDeploy: boolValue(input.AllowRemoteHostDeploy),
		Enabled:               enabled,
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	result := validateReleaseTargetShape(target)
	if !result.Valid {
		return domainenv.ReleaseTarget{}, fmt.Errorf("%w: %s", ErrInvalid, strings.Join(result.Errors, "; "))
	}
	return s.store.CreateReleaseTarget(ctx, target)
}

func (s *Service) GetReleaseTarget(ctx context.Context, id string) (domainenv.ReleaseTarget, error) {
	return s.store.GetReleaseTarget(ctx, strings.TrimSpace(id))
}

func (s *Service) ListReleaseTargets(ctx context.Context, projectID string, environmentID string) ([]domainenv.ReleaseTarget, error) {
	return s.store.ListReleaseTargets(ctx, strings.TrimSpace(projectID), strings.TrimSpace(environmentID))
}

func (s *Service) UpdateReleaseTarget(ctx context.Context, id string, input UpdateReleaseTargetInput) (domainenv.ReleaseTarget, error) {
	target, err := s.store.GetReleaseTarget(ctx, strings.TrimSpace(id))
	if err != nil {
		return domainenv.ReleaseTarget{}, err
	}
	if input.Name != nil {
		target.Name = strings.TrimSpace(*input.Name)
		if target.Name == "" {
			return domainenv.ReleaseTarget{}, fmt.Errorf("%w: release target name is required", ErrInvalid)
		}
	}
	if input.TargetType != nil {
		target.TargetType = normalizeTargetType(*input.TargetType)
		if err := validateTargetType(target.TargetType); err != nil {
			return domainenv.ReleaseTarget{}, err
		}
	}
	if input.Context != nil {
		target.Context = strings.TrimSpace(*input.Context)
	}
	if input.Namespace != nil {
		target.Namespace = strings.TrimSpace(*input.Namespace)
	}
	if input.ConfigRef != nil {
		target.ConfigRef = strings.TrimSpace(*input.ConfigRef)
	}
	if input.CredentialRef != nil {
		target.CredentialRef = strings.TrimSpace(*input.CredentialRef)
	}
	if input.Labels != nil {
		target.Labels = copyMap(input.Labels)
	}
	if input.Metadata != nil {
		target.Metadata = copyMap(input.Metadata)
	}
	if input.AllowApply != nil {
		target.AllowApply = *input.AllowApply
	}
	if input.AllowSync != nil {
		target.AllowSync = *input.AllowSync
	}
	if input.AllowRemoteHostDeploy != nil {
		target.AllowRemoteHostDeploy = *input.AllowRemoteHostDeploy
	}
	if input.Enabled != nil {
		target.Enabled = *input.Enabled
	}
	target.UpdatedAt = s.now().UTC()
	result := validateReleaseTargetShape(target)
	if !result.Valid {
		return domainenv.ReleaseTarget{}, fmt.Errorf("%w: %s", ErrInvalid, strings.Join(result.Errors, "; "))
	}
	targets, err := s.store.ListReleaseTargets(ctx, target.ProjectID, target.EnvironmentID)
	if err != nil {
		return domainenv.ReleaseTarget{}, err
	}
	for _, existing := range targets {
		if existing.ID != target.ID && strings.EqualFold(existing.Name, target.Name) {
			return domainenv.ReleaseTarget{}, fmt.Errorf("%w: release target %q", ErrAlreadyExists, target.Name)
		}
	}
	return s.store.UpdateReleaseTarget(ctx, target)
}

func (s *Service) DisableReleaseTarget(ctx context.Context, id string) (domainenv.ReleaseTarget, error) {
	disabled := false
	return s.UpdateReleaseTarget(ctx, id, UpdateReleaseTargetInput{Enabled: &disabled})
}

func (s *Service) ValidateReleaseTarget(ctx context.Context, id string) (ReleaseTargetValidationResult, error) {
	target, err := s.store.GetReleaseTarget(ctx, strings.TrimSpace(id))
	if err != nil {
		return ReleaseTargetValidationResult{}, err
	}
	return validateReleaseTarget(target), nil
}

func (s *Service) findOrgByNameOrSlug(ctx context.Context, name string, slug string) (domainorg.Org, error) {
	orgs, err := s.store.ListOrgs(ctx)
	if err != nil {
		return domainorg.Org{}, err
	}
	for _, existing := range orgs {
		if sameIdentity(existing.Name, existing.Slug, name, slug) {
			return existing, nil
		}
	}
	return domainorg.Org{}, ErrNotFound
}

func sameIdentity(existingName string, existingSlug string, name string, slug string) bool {
	return strings.EqualFold(strings.TrimSpace(existingName), strings.TrimSpace(name)) ||
		(strings.TrimSpace(slug) != "" && strings.EqualFold(strings.TrimSpace(existingSlug), defaultSlug(slug, name)))
}

func defaultID(id string, prefix string) string {
	if strings.TrimSpace(id) != "" {
		return strings.TrimSpace(id)
	}
	random := make([]byte, 6)
	if _, err := rand.Read(random); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return prefix + "-" + hex.EncodeToString(random)
}

func validateRepositoryURL(raw string, provider string) error {
	if strings.TrimSpace(raw) == "" {
		return fmt.Errorf("%w: repository url is required", ErrInvalid)
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" {
		return fmt.Errorf("%w: repository url must include a scheme", ErrInvalid)
	}
	if parsed.User != nil && parsed.User.String() != "" {
		return fmt.Errorf("%w: repository url must not contain inline credentials; use CredentialRef instead", ErrInvalid)
	}
	switch parsed.Scheme {
	case "https", "http", "ssh", "git":
		return nil
	case "file":
		switch defaultProvider(provider) {
		case "local", "archive":
			return nil
		default:
			return fmt.Errorf("%w: file repository urls require local or archive provider metadata", ErrInvalid)
		}
	default:
		return fmt.Errorf("%w: unsupported repository url scheme %q", ErrInvalid, parsed.Scheme)
	}
}

func validateRepositoryProvider(provider string) error {
	switch defaultProvider(provider) {
	case "generic", "github", "gitlab", "gitea", "local", "archive":
		return nil
	default:
		return fmt.Errorf("%w: unsupported repository provider %q", ErrInvalid, provider)
	}
}

func validateRepository(repository domainapp.Repository) RepositoryValidationResult {
	result := validateRepositoryShape(repository)
	if !repository.Enabled {
		result.Errors = append(result.Errors, "repository is disabled")
	}
	if len(result.Errors) > 0 {
		result.Valid = false
	}
	return result
}

func validateRepositoryShape(repository domainapp.Repository) RepositoryValidationResult {
	result := RepositoryValidationResult{Valid: true, RepositoryID: repository.ID}
	if strings.TrimSpace(repository.ProjectID) == "" {
		result.Errors = append(result.Errors, "projectId is required")
	}
	if strings.TrimSpace(repository.Name) == "" {
		result.Errors = append(result.Errors, "name is required")
	}
	if err := validateRepositoryURL(repository.URL, repository.Provider); err != nil {
		result.Errors = append(result.Errors, err.Error())
	}
	if err := validateRepositoryProvider(repository.Provider); err != nil {
		result.Errors = append(result.Errors, err.Error())
	}
	if strings.TrimSpace(repository.DefaultBranch) == "" {
		result.Errors = append(result.Errors, "defaultBranch is required")
	}
	if strings.TrimSpace(repository.CredentialRef) != "" {
		result.Warnings = append(result.Warnings, "credentialRef is metadata only; repository validation does not resolve or return secret values")
	}
	if strings.TrimSpace(repository.Provider) != "" && strings.TrimSpace(repository.Provider) != "generic" {
		result.Warnings = append(result.Warnings, "provider-specific network validation is not implemented; this is metadata validation only")
	}
	if len(result.Errors) > 0 {
		result.Valid = false
	}
	return result
}

func defaultProvider(provider string) string {
	provider = strings.TrimSpace(strings.ToLower(provider))
	if provider == "" {
		return "generic"
	}
	if provider == "generic_git" {
		return "generic"
	}
	return provider
}

func defaultBranch(branch string) string {
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return "main"
	}
	return branch
}

func normalizeTargetType(targetType string) string {
	return strings.TrimSpace(strings.ToLower(targetType))
}

func validateTargetType(targetType string) error {
	switch targetType {
	case "kubernetes-yaml", "argocd", "host", "webhook", "noop":
		return nil
	default:
		return fmt.Errorf("%w: release target type %q is not supported", ErrInvalid, targetType)
	}
}

func validateReleaseTarget(target domainenv.ReleaseTarget) ReleaseTargetValidationResult {
	result := validateReleaseTargetShape(target)
	if !target.Enabled {
		result.Errors = append(result.Errors, "target is disabled")
	}
	if len(result.Errors) > 0 {
		result.Valid = false
	}
	return result
}

func validateReleaseTargetShape(target domainenv.ReleaseTarget) ReleaseTargetValidationResult {
	result := ReleaseTargetValidationResult{Valid: true, TargetID: target.ID}
	if strings.TrimSpace(target.ProjectID) == "" {
		result.Errors = append(result.Errors, "projectId is required")
	}
	if strings.TrimSpace(target.EnvironmentID) == "" {
		result.Errors = append(result.Errors, "environmentId is required")
	}
	if strings.TrimSpace(target.Name) == "" {
		result.Errors = append(result.Errors, "name is required")
	}
	if err := validateTargetType(target.TargetType); err != nil {
		result.Errors = append(result.Errors, err.Error())
	}
	switch target.TargetType {
	case "kubernetes-yaml":
		if target.AllowApply {
			result.Warnings = append(result.Warnings, "kubernetes apply is explicitly allowed for this target")
		}
		if target.Namespace == "" {
			result.Warnings = append(result.Warnings, "namespace is not set; deployment specs must provide an explicit namespace")
		}
	case "argocd":
		if target.AllowSync {
			result.Warnings = append(result.Warnings, "Argo CD sync is explicitly allowed for this target")
		}
	case "host":
		if target.AllowRemoteHostDeploy {
			result.Warnings = append(result.Warnings, "remote host deployment is explicitly allowed for this target")
		}
	}
	if len(result.Errors) > 0 {
		result.Valid = false
	}
	return result
}

func boolValue(value *bool) bool {
	return value != nil && *value
}

func defaultSlug(slug string, name string) string {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		slug = strings.TrimSpace(name)
	}
	slug = strings.ToLower(slug)
	var b strings.Builder
	lastDash := false
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteRune('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func copyMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
