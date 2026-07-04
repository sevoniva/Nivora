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

func TestGetEnabledRejectsDisabledPolicy(t *testing.T) {
	service := NewService(NewMemoryStore())
	ctx := context.Background()
	policy, err := service.Create(ctx, CreateInput{ID: "policy-disabled", Name: "Disabled policy"})
	if err != nil {
		t.Fatalf("create policy: %v", err)
	}
	if _, err := service.GetEnabled(ctx, policy.ID); err != nil {
		t.Fatalf("enabled policy should be readable: %v", err)
	}
	if _, err := service.Disable(ctx, policy.ID); err != nil {
		t.Fatalf("disable policy: %v", err)
	}
	if _, err := service.GetEnabled(ctx, policy.ID); err == nil {
		t.Fatal("expected disabled policy to be rejected")
	}
}

func TestPolicyAttachmentLifecycle(t *testing.T) {
	service := NewService(NewMemoryStore())
	ctx := context.Background()
	policy, err := service.Create(ctx, CreateInput{ID: "policy-approval", Name: "Approval on critical", ApprovalOnCritical: true})
	if err != nil {
		t.Fatalf("create policy: %v", err)
	}

	attachment, err := service.Attach(ctx, policy.ID, AttachInput{
		ID:        "attach-env-prod",
		ScopeType: "environment",
		ScopeID:   "prod",
	})
	if err != nil {
		t.Fatalf("attach policy: %v", err)
	}
	if attachment.PolicyID != policy.ID || attachment.ScopeType != "environment" || attachment.ScopeID != "prod" || !attachment.Enabled {
		t.Fatalf("unexpected attachment: %+v", attachment)
	}

	if _, err := service.Attach(ctx, policy.ID, AttachInput{ScopeType: "environment", ScopeID: "prod"}); err == nil {
		t.Fatal("expected duplicate attachment error")
	}
	if _, err := service.Attach(ctx, policy.ID, AttachInput{ScopeType: "environment"}); err == nil {
		t.Fatal("expected missing scope id error")
	}
	if _, err := service.Attach(ctx, policy.ID, AttachInput{ScopeType: "unsupported", ScopeID: "x"}); err == nil {
		t.Fatal("expected invalid scope type error")
	}
	if _, err := service.Attach(ctx, "missing", AttachInput{ScopeType: "project", ScopeID: "project-a"}); err == nil {
		t.Fatal("expected missing policy error")
	}

	targetAttachment, err := service.Attach(ctx, policy.ID, AttachInput{ScopeType: "release-target", ScopeID: "target-a"})
	if err != nil {
		t.Fatalf("attach target policy: %v", err)
	}
	if targetAttachment.ScopeType != "target" {
		t.Fatalf("expected normalized target scope, got %+v", targetAttachment)
	}

	listed, err := service.ListAttachments(ctx, AttachmentListInput{PolicyID: policy.ID, ScopeType: "target"})
	if err != nil {
		t.Fatalf("list attachments: %v", err)
	}
	if len(listed) != 1 || listed[0].ScopeID != "target-a" {
		t.Fatalf("unexpected target attachments: %+v", listed)
	}
}

func TestResolveEnabledForScopePrefersEnvironmentProjectGlobal(t *testing.T) {
	service := NewService(NewMemoryStore())
	ctx := context.Background()
	global, err := service.Create(ctx, CreateInput{ID: "policy-global", Name: "Global", RequireDigest: true})
	if err != nil {
		t.Fatalf("create global policy: %v", err)
	}
	project, err := service.Create(ctx, CreateInput{ID: "policy-project", Name: "Project", ProjectID: "project-a", HighWarn: 2})
	if err != nil {
		t.Fatalf("create project policy: %v", err)
	}
	environment, err := service.Create(ctx, CreateInput{ID: "policy-environment", Name: "Environment", EnvironmentID: "prod", ApprovalOnCritical: true})
	if err != nil {
		t.Fatalf("create environment policy: %v", err)
	}
	if _, err := service.Attach(ctx, global.ID, AttachInput{ScopeType: "global"}); err != nil {
		t.Fatalf("attach global policy: %v", err)
	}
	if _, err := service.Attach(ctx, project.ID, AttachInput{ScopeType: "project", ScopeID: "project-a"}); err != nil {
		t.Fatalf("attach project policy: %v", err)
	}
	if _, err := service.Attach(ctx, environment.ID, AttachInput{ScopeType: "environment", ScopeID: "prod"}); err != nil {
		t.Fatalf("attach environment policy: %v", err)
	}

	resolved, ok, err := service.ResolveEnabledForScope(ctx, ResolveInput{ProjectID: "project-a", EnvironmentID: "prod"})
	if err != nil {
		t.Fatalf("resolve environment policy: %v", err)
	}
	if !ok || resolved.ID != environment.ID {
		t.Fatalf("resolved policy = %+v ok=%v, want environment", resolved, ok)
	}

	resolved, ok, err = service.ResolveEnabledForScope(ctx, ResolveInput{ProjectID: "project-a"})
	if err != nil {
		t.Fatalf("resolve project policy: %v", err)
	}
	if !ok || resolved.ID != project.ID {
		t.Fatalf("resolved policy = %+v ok=%v, want project", resolved, ok)
	}

	resolved, ok, err = service.ResolveEnabledForScope(ctx, ResolveInput{ProjectID: "project-b"})
	if err != nil {
		t.Fatalf("resolve global policy: %v", err)
	}
	if !ok || resolved.ID != global.ID {
		t.Fatalf("resolved policy = %+v ok=%v, want global", resolved, ok)
	}
}

func TestResolveEnabledForScopeSkipsDisabledPolicy(t *testing.T) {
	service := NewService(NewMemoryStore())
	ctx := context.Background()
	disabled, err := service.Create(ctx, CreateInput{ID: "policy-disabled", Name: "Disabled", RequireDigest: true})
	if err != nil {
		t.Fatalf("create disabled policy: %v", err)
	}
	fallback, err := service.Create(ctx, CreateInput{ID: "policy-fallback", Name: "Fallback", HighWarn: 2})
	if err != nil {
		t.Fatalf("create fallback policy: %v", err)
	}
	if _, err := service.Attach(ctx, disabled.ID, AttachInput{ScopeType: "environment", ScopeID: "prod"}); err != nil {
		t.Fatalf("attach disabled policy: %v", err)
	}
	if _, err := service.Attach(ctx, fallback.ID, AttachInput{ScopeType: "global"}); err != nil {
		t.Fatalf("attach fallback policy: %v", err)
	}
	if _, err := service.Disable(ctx, disabled.ID); err != nil {
		t.Fatalf("disable policy: %v", err)
	}

	resolved, ok, err := service.ResolveEnabledForScope(ctx, ResolveInput{EnvironmentID: "prod"})
	if err != nil {
		t.Fatalf("resolve policy: %v", err)
	}
	if !ok || resolved.ID != fallback.ID {
		t.Fatalf("resolved policy = %+v ok=%v, want fallback", resolved, ok)
	}
}
