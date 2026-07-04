package policy

import (
	"context"
	"testing"
)

func TestPolicyCatalogCreateUpdateDisable(t *testing.T) {
	service := NewService(NewMemoryStore())
	ctx := context.Background()

	created, err := service.Create(ctx, CreateInput{
		ID:            "policy-digest",
		ProjectID:     "project-a",
		EnvironmentID: "prod",
		Name:          "Require digest",
		RequireDigest: true,
	})
	if err != nil {
		t.Fatalf("create policy: %v", err)
	}
	if created.ID != "policy-digest" || !created.RequireDigest || !created.Enabled {
		t.Fatalf("unexpected policy: %+v", created)
	}

	if _, err := service.Create(ctx, CreateInput{ProjectID: "project-a", EnvironmentID: "prod", Name: "Require digest"}); err == nil {
		t.Fatal("expected duplicate policy error")
	}

	highWarn := 2
	updated, err := service.Update(ctx, created.ID, UpdateInput{HighWarn: &highWarn})
	if err != nil {
		t.Fatalf("update policy: %v", err)
	}
	if updated.HighWarn != 2 {
		t.Fatalf("high warn not updated: %+v", updated)
	}

	disabled, err := service.Disable(ctx, created.ID)
	if err != nil {
		t.Fatalf("disable policy: %v", err)
	}
	if disabled.Enabled {
		t.Fatalf("policy should be disabled: %+v", disabled)
	}

	listed, err := service.List(ctx, "project-a", "prod")
	if err != nil {
		t.Fatalf("list policy: %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("expected one policy, got %d", len(listed))
	}
}

func TestPolicyCatalogValidation(t *testing.T) {
	service := NewService(NewMemoryStore())
	if _, err := service.Create(context.Background(), CreateInput{}); err == nil {
		t.Fatal("expected validation error")
	}
	if _, err := service.Get(context.Background(), "missing"); err == nil {
		t.Fatal("expected missing policy error")
	}
}
