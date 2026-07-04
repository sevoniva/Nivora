package pipeline

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestDefinitionCatalogCreateUpdateAndDisable(t *testing.T) {
	ctx := context.Background()
	catalog := NewDefinitionCatalog(NewDefinitionMemoryStore())

	record, err := catalog.Create(ctx, DefinitionCreateInput{ProjectID: "project-a", Definition: catalogTestDefinition("build")})
	if err != nil {
		t.Fatalf("create definition: %v", err)
	}
	if record.Pipeline.ProjectID != "project-a" || !record.Pipeline.Enabled || record.Version.Version != 1 || record.Version.DefinitionHash == "" {
		t.Fatalf("unexpected record: %+v", record)
	}

	updated, err := catalog.Update(ctx, record.Pipeline.ID, DefinitionUpdateInput{Definition: ptrDefinition(catalogTestDefinition("build-v2"))})
	if err != nil {
		t.Fatalf("update definition: %v", err)
	}
	if updated.Version.Version != 2 || updated.Pipeline.Name != "build-v2" {
		t.Fatalf("unexpected updated record: %+v", updated)
	}
	versions, err := catalog.Versions(ctx, record.Pipeline.ID)
	if err != nil {
		t.Fatalf("list versions: %v", err)
	}
	if len(versions) != 2 || versions[0].Version != 1 || versions[1].Version != 2 {
		t.Fatalf("unexpected versions: %+v", versions)
	}
	firstVersion, err := catalog.Version(ctx, record.Pipeline.ID, 1)
	if err != nil {
		t.Fatalf("get first version: %v", err)
	}
	if firstVersion.Version.ID != record.Version.ID || firstVersion.Definition.Metadata.Name != "build" {
		t.Fatalf("unexpected first version record: %+v", firstVersion)
	}
	secondVersion, err := catalog.Version(ctx, record.Pipeline.ID, 2)
	if err != nil {
		t.Fatalf("get second version: %v", err)
	}
	if secondVersion.Version.ID != updated.Version.ID || secondVersion.Definition.Metadata.Name != "build-v2" {
		t.Fatalf("unexpected second version record: %+v", secondVersion)
	}

	rollbackDescription := "restored stable definition"
	rolledBack, err := catalog.Rollback(ctx, record.Pipeline.ID, DefinitionRollbackInput{Version: 1, Description: &rollbackDescription})
	if err != nil {
		t.Fatalf("rollback definition: %v", err)
	}
	if rolledBack.Version.Version != 3 || rolledBack.Definition.Metadata.Name != "build" || rolledBack.Pipeline.Description != rollbackDescription {
		t.Fatalf("unexpected rollback record: %+v", rolledBack)
	}
	versions, err = catalog.Versions(ctx, record.Pipeline.ID)
	if err != nil {
		t.Fatalf("list versions after rollback: %v", err)
	}
	if len(versions) != 3 || versions[2].Version != 3 {
		t.Fatalf("rollback should append version 3, got %+v", versions)
	}

	disabled, err := catalog.Disable(ctx, record.Pipeline.ID)
	if err != nil {
		t.Fatalf("disable definition: %v", err)
	}
	if disabled.Pipeline.Enabled {
		t.Fatalf("disable should set enabled=false: %+v", disabled)
	}
}

func TestDefinitionCatalogRejectsInvalidAndDuplicateDefinitions(t *testing.T) {
	ctx := context.Background()
	catalog := NewDefinitionCatalog(NewDefinitionMemoryStore())
	if _, err := catalog.Create(ctx, DefinitionCreateInput{ProjectID: "project-a", Definition: Definition{Kind: "Other"}}); err == nil {
		t.Fatal("expected invalid definition error")
	}
	if _, err := catalog.Create(ctx, DefinitionCreateInput{ProjectID: "project-a", Definition: catalogTestDefinition("build")}); err != nil {
		t.Fatalf("create definition: %v", err)
	}
	if _, err := catalog.Create(ctx, DefinitionCreateInput{ProjectID: "project-a", Definition: catalogTestDefinition("BUILD")}); !errors.Is(err, ErrPipelineDefinitionAlreadyExists) {
		t.Fatalf("duplicate error = %v", err)
	}
}

func TestDefinitionCatalogRollbackRejectsInvalidTargets(t *testing.T) {
	ctx := context.Background()
	catalog := NewDefinitionCatalog(NewDefinitionMemoryStore())
	record, err := catalog.Create(ctx, DefinitionCreateInput{ProjectID: "project-a", Definition: catalogTestDefinition("build")})
	if err != nil {
		t.Fatalf("create definition: %v", err)
	}
	if _, err := catalog.Rollback(ctx, record.Pipeline.ID, DefinitionRollbackInput{Version: 0}); err == nil || !strings.Contains(err.Error(), "greater than zero") {
		t.Fatalf("expected invalid version error, got %v", err)
	}
	if _, err := catalog.Rollback(ctx, record.Pipeline.ID, DefinitionRollbackInput{Version: 1}); err == nil || !strings.Contains(err.Error(), "older than current") {
		t.Fatalf("expected current version rejection, got %v", err)
	}
	if _, err := catalog.Update(ctx, record.Pipeline.ID, DefinitionUpdateInput{Definition: ptrDefinition(catalogTestDefinition("build-v2"))}); err != nil {
		t.Fatalf("update definition: %v", err)
	}
	if _, err := catalog.Rollback(ctx, record.Pipeline.ID, DefinitionRollbackInput{Version: 99}); err == nil || !strings.Contains(err.Error(), "older than current") {
		t.Fatalf("expected future version rejection, got %v", err)
	}
}

func ptrDefinition(def Definition) *Definition {
	return &def
}

func catalogTestDefinition(name string) Definition {
	return Definition{
		APIVersion: "nivora.io/v1alpha1",
		Kind:       "Pipeline",
		Metadata:   Metadata{Name: name},
		Spec: Spec{Stages: []Stage{{
			Name: "build",
			Jobs: []Job{{
				Name:     "test",
				Executor: "shell",
				Steps:    []Step{{Name: "echo", Run: "printf ok"}},
			}},
		}}},
	}
}
