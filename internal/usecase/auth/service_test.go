package auth

import (
	"context"
	"testing"

	domainauth "github.com/sevoniva/nivora/internal/domain/auth"
)

func TestRBACAllow(t *testing.T) {
	service := NewService(NewMemoryStore(), nil)
	decision := service.Evaluate(EvaluateInput{
		Subject: domainauth.Subject{ID: "user-1", Roles: []string{domainauth.RoleOwner}},
		Action:  domainauth.PermissionCredentialManage,
	})
	if !decision.Allowed {
		t.Fatalf("expected owner to manage credentials: %#v", decision)
	}
}

func TestRBACDeny(t *testing.T) {
	service := NewService(NewMemoryStore(), nil)
	decision := service.Evaluate(EvaluateInput{
		Subject: domainauth.Subject{ID: "user-1", Roles: []string{domainauth.RoleViewer}},
		Action:  domainauth.PermissionCredentialManage,
	})
	if decision.Allowed {
		t.Fatalf("expected viewer to be denied")
	}
}

func TestTokenAuth(t *testing.T) {
	service := NewService(NewMemoryStore(), nil)
	subject, err := service.Authenticate(context.Background(), AuthenticateInput{Mode: "token", Token: "test-token", StaticToken: "test-token"})
	if err != nil {
		t.Fatalf("authenticate token: %v", err)
	}
	if subject.ID == "" || subject.AuthMode != "token" {
		t.Fatalf("unexpected subject: %#v", subject)
	}
	if _, err := service.Authenticate(context.Background(), AuthenticateInput{Mode: "token", Token: "wrong", StaticToken: "test-token"}); err == nil {
		t.Fatalf("expected wrong token to fail")
	}
}

func TestMembershipAudit(t *testing.T) {
	service := NewService(NewMemoryStore(), nil)
	membership, err := service.CreateMembership(context.Background(), MembershipInput{UserID: "user-1", Role: domainauth.RoleDeveloper, ScopeType: "project", ScopeID: "project-1"}, "admin")
	if err != nil {
		t.Fatalf("create membership: %v", err)
	}
	if membership.ID == "" {
		t.Fatalf("expected membership id")
	}
	memberships, err := service.ListMemberships(context.Background(), "project", "project-1")
	if err != nil {
		t.Fatalf("list memberships: %v", err)
	}
	if len(memberships) != 1 {
		t.Fatalf("memberships len = %d", len(memberships))
	}
}
