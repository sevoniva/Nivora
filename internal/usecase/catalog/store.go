package catalog

import (
	"context"
	"fmt"
	"sort"
	"sync"

	domainapp "github.com/sevoniva/nivora/internal/domain/application"
	"github.com/sevoniva/nivora/internal/domain/audit"
	domainenv "github.com/sevoniva/nivora/internal/domain/environment"
	"github.com/sevoniva/nivora/internal/domain/event"
	domainorg "github.com/sevoniva/nivora/internal/domain/org"
	domainproject "github.com/sevoniva/nivora/internal/domain/project"
)

type MemoryStore struct {
	mu           sync.RWMutex
	orgs         map[string]domainorg.Org
	projects     map[string]domainproject.Project
	applications map[string]domainapp.Application
	environments map[string]domainenv.Environment
	repositories map[string]domainapp.Repository
	targets      map[string]domainenv.ReleaseTarget
	events       map[string][]event.Event
	audits       map[string][]audit.AuditLog
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		orgs:         map[string]domainorg.Org{},
		projects:     map[string]domainproject.Project{},
		applications: map[string]domainapp.Application{},
		environments: map[string]domainenv.Environment{},
		repositories: map[string]domainapp.Repository{},
		targets:      map[string]domainenv.ReleaseTarget{},
		events:       map[string][]event.Event{},
		audits:       map[string][]audit.AuditLog{},
	}
}

func (s *MemoryStore) CreateOrg(ctx context.Context, org domainorg.Org) (domainorg.Org, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.orgs[org.ID]; ok {
		return domainorg.Org{}, fmt.Errorf("%w: org id %q", ErrAlreadyExists, org.ID)
	}
	s.orgs[org.ID] = copyOrg(org)
	return copyOrg(org), nil
}

func (s *MemoryStore) GetOrg(ctx context.Context, id string) (domainorg.Org, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	org, ok := s.orgs[id]
	if !ok {
		return domainorg.Org{}, fmt.Errorf("%w: org %q", ErrNotFound, id)
	}
	return copyOrg(org), nil
}

func (s *MemoryStore) ListOrgs(ctx context.Context) ([]domainorg.Org, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domainorg.Org, 0, len(s.orgs))
	for _, org := range s.orgs {
		out = append(out, copyOrg(org))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *MemoryStore) UpdateOrg(ctx context.Context, org domainorg.Org) (domainorg.Org, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.orgs[org.ID]; !ok {
		return domainorg.Org{}, fmt.Errorf("%w: org %q", ErrNotFound, org.ID)
	}
	s.orgs[org.ID] = copyOrg(org)
	return copyOrg(org), nil
}

func (s *MemoryStore) CreateProject(ctx context.Context, project domainproject.Project) (domainproject.Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.projects[project.ID]; ok {
		return domainproject.Project{}, fmt.Errorf("%w: project id %q", ErrAlreadyExists, project.ID)
	}
	s.projects[project.ID] = copyProject(project)
	return copyProject(project), nil
}

func (s *MemoryStore) GetProject(ctx context.Context, id string) (domainproject.Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	project, ok := s.projects[id]
	if !ok {
		return domainproject.Project{}, fmt.Errorf("%w: project %q", ErrNotFound, id)
	}
	return copyProject(project), nil
}

func (s *MemoryStore) ListProjects(ctx context.Context, orgID string) ([]domainproject.Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domainproject.Project, 0, len(s.projects))
	for _, project := range s.projects {
		if orgID == "" || project.OrgID == orgID {
			out = append(out, copyProject(project))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *MemoryStore) UpdateProject(ctx context.Context, project domainproject.Project) (domainproject.Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.projects[project.ID]; !ok {
		return domainproject.Project{}, fmt.Errorf("%w: project %q", ErrNotFound, project.ID)
	}
	s.projects[project.ID] = copyProject(project)
	return copyProject(project), nil
}

func (s *MemoryStore) CreateApplication(ctx context.Context, app domainapp.Application) (domainapp.Application, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.applications[app.ID]; ok {
		return domainapp.Application{}, fmt.Errorf("%w: application id %q", ErrAlreadyExists, app.ID)
	}
	s.applications[app.ID] = copyApplication(app)
	return copyApplication(app), nil
}

func (s *MemoryStore) GetApplication(ctx context.Context, id string) (domainapp.Application, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	app, ok := s.applications[id]
	if !ok {
		return domainapp.Application{}, fmt.Errorf("%w: application %q", ErrNotFound, id)
	}
	return copyApplication(app), nil
}

func (s *MemoryStore) ListApplications(ctx context.Context, projectID string) ([]domainapp.Application, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domainapp.Application, 0, len(s.applications))
	for _, app := range s.applications {
		if projectID == "" || app.ProjectID == projectID {
			out = append(out, copyApplication(app))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *MemoryStore) UpdateApplication(ctx context.Context, app domainapp.Application) (domainapp.Application, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.applications[app.ID]; !ok {
		return domainapp.Application{}, fmt.Errorf("%w: application %q", ErrNotFound, app.ID)
	}
	s.applications[app.ID] = copyApplication(app)
	return copyApplication(app), nil
}

func (s *MemoryStore) CreateEnvironment(ctx context.Context, environment domainenv.Environment) (domainenv.Environment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.environments[environment.ID]; ok {
		return domainenv.Environment{}, fmt.Errorf("%w: environment id %q", ErrAlreadyExists, environment.ID)
	}
	s.environments[environment.ID] = copyEnvironment(environment)
	return copyEnvironment(environment), nil
}

func (s *MemoryStore) GetEnvironment(ctx context.Context, id string) (domainenv.Environment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	environment, ok := s.environments[id]
	if !ok {
		return domainenv.Environment{}, fmt.Errorf("%w: environment %q", ErrNotFound, id)
	}
	return copyEnvironment(environment), nil
}

func (s *MemoryStore) ListEnvironments(ctx context.Context, projectID string) ([]domainenv.Environment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domainenv.Environment, 0, len(s.environments))
	for _, environment := range s.environments {
		if projectID == "" || environment.ProjectID == projectID {
			out = append(out, copyEnvironment(environment))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *MemoryStore) UpdateEnvironment(ctx context.Context, environment domainenv.Environment) (domainenv.Environment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.environments[environment.ID]; !ok {
		return domainenv.Environment{}, fmt.Errorf("%w: environment %q", ErrNotFound, environment.ID)
	}
	s.environments[environment.ID] = copyEnvironment(environment)
	return copyEnvironment(environment), nil
}

func (s *MemoryStore) CreateRepository(ctx context.Context, repository domainapp.Repository) (domainapp.Repository, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.repositories[repository.ID]; ok {
		return domainapp.Repository{}, fmt.Errorf("%w: repository id %q", ErrAlreadyExists, repository.ID)
	}
	s.repositories[repository.ID] = copyRepository(repository)
	return copyRepository(repository), nil
}

func (s *MemoryStore) GetRepository(ctx context.Context, id string) (domainapp.Repository, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	repository, ok := s.repositories[id]
	if !ok {
		return domainapp.Repository{}, fmt.Errorf("%w: repository %q", ErrNotFound, id)
	}
	return copyRepository(repository), nil
}

func (s *MemoryStore) ListRepositories(ctx context.Context, projectID string) ([]domainapp.Repository, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domainapp.Repository, 0, len(s.repositories))
	for _, repository := range s.repositories {
		if projectID == "" || repository.ProjectID == projectID {
			out = append(out, copyRepository(repository))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *MemoryStore) UpdateRepository(ctx context.Context, repository domainapp.Repository) (domainapp.Repository, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.repositories[repository.ID]; !ok {
		return domainapp.Repository{}, fmt.Errorf("%w: repository %q", ErrNotFound, repository.ID)
	}
	s.repositories[repository.ID] = copyRepository(repository)
	return copyRepository(repository), nil
}

func (s *MemoryStore) CreateReleaseTarget(ctx context.Context, target domainenv.ReleaseTarget) (domainenv.ReleaseTarget, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.targets[target.ID]; ok {
		return domainenv.ReleaseTarget{}, fmt.Errorf("%w: release target id %q", ErrAlreadyExists, target.ID)
	}
	s.targets[target.ID] = copyReleaseTarget(target)
	return copyReleaseTarget(target), nil
}

func (s *MemoryStore) GetReleaseTarget(ctx context.Context, id string) (domainenv.ReleaseTarget, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	target, ok := s.targets[id]
	if !ok {
		return domainenv.ReleaseTarget{}, fmt.Errorf("%w: release target %q", ErrNotFound, id)
	}
	return copyReleaseTarget(target), nil
}

func (s *MemoryStore) ListReleaseTargets(ctx context.Context, projectID string, environmentID string) ([]domainenv.ReleaseTarget, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domainenv.ReleaseTarget, 0, len(s.targets))
	for _, target := range s.targets {
		if projectID != "" && target.ProjectID != projectID {
			continue
		}
		if environmentID != "" && target.EnvironmentID != environmentID {
			continue
		}
		out = append(out, copyReleaseTarget(target))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *MemoryStore) UpdateReleaseTarget(ctx context.Context, target domainenv.ReleaseTarget) (domainenv.ReleaseTarget, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.targets[target.ID]; !ok {
		return domainenv.ReleaseTarget{}, fmt.Errorf("%w: release target %q", ErrNotFound, target.ID)
	}
	s.targets[target.ID] = copyReleaseTarget(target)
	return copyReleaseTarget(target), nil
}

func (s *MemoryStore) AppendEvent(ctx context.Context, subject string, evt event.Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events[subject] = append(s.events[subject], evt)
	return nil
}

func (s *MemoryStore) EventsBySubject(ctx context.Context, subject string) ([]event.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	events := append([]event.Event(nil), s.events[subject]...)
	sort.Slice(events, func(i, j int) bool {
		if events[i].Time.Equal(events[j].Time) {
			return events[i].ID < events[j].ID
		}
		return events[i].Time.Before(events[j].Time)
	})
	return events, nil
}

func (s *MemoryStore) AppendAudit(ctx context.Context, subject string, entry audit.AuditLog) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.audits[subject] = append(s.audits[subject], entry)
	return nil
}

func (s *MemoryStore) AuditsBySubject(ctx context.Context, subject string) ([]audit.AuditLog, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	audits := append([]audit.AuditLog(nil), s.audits[subject]...)
	sort.Slice(audits, func(i, j int) bool {
		if audits[i].CreatedAt.Equal(audits[j].CreatedAt) {
			return audits[i].ID < audits[j].ID
		}
		return audits[i].CreatedAt.Before(audits[j].CreatedAt)
	})
	return audits, nil
}

func copyOrg(org domainorg.Org) domainorg.Org {
	org.Labels = copyMap(org.Labels)
	org.Metadata = copyMap(org.Metadata)
	return org
}

func copyProject(project domainproject.Project) domainproject.Project {
	project.Labels = copyMap(project.Labels)
	project.Metadata = copyMap(project.Metadata)
	return project
}

func copyApplication(app domainapp.Application) domainapp.Application {
	app.Labels = copyMap(app.Labels)
	app.Metadata = copyMap(app.Metadata)
	return app
}

func copyEnvironment(environment domainenv.Environment) domainenv.Environment {
	environment.Labels = copyMap(environment.Labels)
	environment.Metadata = copyMap(environment.Metadata)
	return environment
}

func copyRepository(repository domainapp.Repository) domainapp.Repository {
	repository.Labels = copyMap(repository.Labels)
	repository.Metadata = copyMap(repository.Metadata)
	return repository
}

func copyReleaseTarget(target domainenv.ReleaseTarget) domainenv.ReleaseTarget {
	target.Labels = copyMap(target.Labels)
	target.Metadata = copyMap(target.Metadata)
	return target
}
