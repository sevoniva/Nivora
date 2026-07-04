package catalog

import (
	"context"
	"errors"
	"testing"
)

func TestCatalogCreatesHierarchyAndDisablesResources(t *testing.T) {
	ctx := context.Background()
	service := NewService(NewMemoryStore())

	org, err := service.CreateOrg(ctx, CreateOrgInput{Name: "Platform"})
	if err != nil {
		t.Fatalf("create org: %v", err)
	}
	if !org.Enabled || org.Slug != "platform" {
		t.Fatalf("unexpected org: %+v", org)
	}

	project, err := service.CreateProject(ctx, CreateProjectInput{OrgID: org.ID, Name: "Delivery"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	if project.OrgID != org.ID || !project.Enabled {
		t.Fatalf("unexpected project: %+v", project)
	}

	app, err := service.CreateApplication(ctx, CreateApplicationInput{ProjectID: project.ID, Name: "Control Plane"})
	if err != nil {
		t.Fatalf("create application: %v", err)
	}
	if app.ProjectID != project.ID || app.Slug != "control-plane" {
		t.Fatalf("unexpected application: %+v", app)
	}

	environment, err := service.CreateEnvironment(ctx, CreateEnvironmentInput{ProjectID: project.ID, Name: "Production"})
	if err != nil {
		t.Fatalf("create environment: %v", err)
	}
	if environment.ProjectID != project.ID || environment.Slug != "production" {
		t.Fatalf("unexpected environment: %+v", environment)
	}

	disabled, err := service.DisableEnvironment(ctx, environment.ID)
	if err != nil {
		t.Fatalf("disable environment: %v", err)
	}
	if disabled.Enabled {
		t.Fatalf("delete should disable environment, got %+v", disabled)
	}
}

func TestCatalogRejectsMissingParentsAndDuplicates(t *testing.T) {
	ctx := context.Background()
	service := NewService(NewMemoryStore())

	if _, err := service.CreateProject(ctx, CreateProjectInput{OrgID: "missing", Name: "Delivery"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing org error = %v", err)
	}

	org, err := service.CreateOrg(ctx, CreateOrgInput{Name: "Platform"})
	if err != nil {
		t.Fatalf("create org: %v", err)
	}
	if _, err := service.CreateOrg(ctx, CreateOrgInput{Name: "platform"}); !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("duplicate org error = %v", err)
	}
	project, err := service.CreateProject(ctx, CreateProjectInput{OrgID: org.ID, Name: "Delivery"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	if _, err := service.CreateApplication(ctx, CreateApplicationInput{ProjectID: "missing", Name: "App"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing project application error = %v", err)
	}
	if _, err := service.CreateApplication(ctx, CreateApplicationInput{ProjectID: project.ID, Name: ""}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("invalid application error = %v", err)
	}
}
