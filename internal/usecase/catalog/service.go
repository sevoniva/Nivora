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
	repoURL := strings.TrimSpace(input.URL)
	if err := validateRepositoryURL(repoURL); err != nil {
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
		Provider:      defaultProvider(input.Provider),
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
		if err := validateRepositoryURL(repository.URL); err != nil {
			return domainapp.Repository{}, err
		}
	}
	if input.Provider != nil {
		repository.Provider = defaultProvider(*input.Provider)
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

func validateRepositoryURL(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return fmt.Errorf("%w: repository url is required", ErrInvalid)
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" {
		return fmt.Errorf("%w: repository url must include a scheme", ErrInvalid)
	}
	switch parsed.Scheme {
	case "https", "http", "ssh", "git":
		return nil
	default:
		return fmt.Errorf("%w: unsupported repository url scheme %q", ErrInvalid, parsed.Scheme)
	}
}

func defaultProvider(provider string) string {
	provider = strings.TrimSpace(strings.ToLower(provider))
	if provider == "" {
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
