package catalog

import (
	"context"
	"fmt"
	"sort"
	"sync"

	domainapp "github.com/sevoniva/nivora/internal/domain/application"
	domainenv "github.com/sevoniva/nivora/internal/domain/environment"
	domainorg "github.com/sevoniva/nivora/internal/domain/org"
	domainproject "github.com/sevoniva/nivora/internal/domain/project"
)

type MemoryStore struct {
	mu           sync.RWMutex
	orgs         map[string]domainorg.Org
	projects     map[string]domainproject.Project
	applications map[string]domainapp.Application
	environments map[string]domainenv.Environment
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		orgs:         map[string]domainorg.Org{},
		projects:     map[string]domainproject.Project{},
		applications: map[string]domainapp.Application{},
		environments: map[string]domainenv.Environment{},
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
