package pipeline

import (
	"context"
	"errors"
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
