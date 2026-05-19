package auth

import (
	"context"
	"testing"
	"time"

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

func TestScopedSubjectDeniedAcrossProjects(t *testing.T) {
	service := NewService(NewMemoryStore(), nil)
	decision := service.Evaluate(EvaluateInput{
		Subject:  domainauth.Subject{ID: "sa-1", Roles: []string{domainauth.RoleDeveloper}, ScopeType: "project", ScopeID: "project-a"},
		Action:   domainauth.PermissionProjectRead,
		Resource: domainauth.Resource{Type: "credential", ScopeType: "project", ScopeID: "project-b"},
	})
	if decision.Allowed {
		t.Fatalf("expected cross-project scoped subject to be denied")
	}
}

func TestScopedSubjectAllowedWithinProject(t *testing.T) {
	service := NewService(NewMemoryStore(), nil)
	decision := service.Evaluate(EvaluateInput{
		Subject:  domainauth.Subject{ID: "sa-1", Roles: []string{domainauth.RoleDeveloper}, ScopeType: "project", ScopeID: "project-a"},
		Action:   domainauth.PermissionProjectRead,
		Resource: domainauth.Resource{Type: "credential", ScopeType: "project", ScopeID: "project-a"},
	})
	if !decision.Allowed {
		t.Fatalf("expected matching project scope to be allowed: %#v", decision)
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

func TestAPITokenHashingAndAuthentication(t *testing.T) {
	service := NewService(NewMemoryStore(), nil)
	account, err := service.CreateServiceAccount(context.Background(), ServiceAccountInput{Name: "ci", Role: domainauth.RoleDeveloper, ScopeType: "project", ScopeID: "project-1"}, "admin")
	if err != nil {
		t.Fatalf("create service account: %v", err)
	}
	result, err := service.CreateAPIToken(context.Background(), APITokenInput{Name: "ci-token", SubjectID: account.ID}, "admin")
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	if result.Token == "" {
		t.Fatalf("expected one-time token")
	}
	if result.Metadata.TokenHash != "" {
		t.Fatalf("token hash leaked in public metadata")
	}
	tokens, err := service.ListAPITokens(context.Background(), account.ID)
	if err != nil {
		t.Fatalf("list tokens: %v", err)
	}
	if len(tokens) != 1 || tokens[0].TokenHash != "" {
		t.Fatalf("token list leaked hash: %#v", tokens)
	}
	subject, err := service.Authenticate(context.Background(), AuthenticateInput{Mode: "token", Token: result.Token})
	if err != nil {
		t.Fatalf("authenticate api token: %v", err)
	}
	if subject.ID != account.ID || subject.TokenID != result.Metadata.ID {
		t.Fatalf("unexpected api token subject: %#v", subject)
	}
	if _, err := service.Authenticate(context.Background(), AuthenticateInput{Mode: "token", Token: "wrong"}); err == nil {
		t.Fatalf("expected wrong API token to fail")
	}
}

func TestAPITokenExpirationAndRevocation(t *testing.T) {
	service := NewService(NewMemoryStore(), nil)
	account, err := service.CreateServiceAccount(context.Background(), ServiceAccountInput{Name: "expired", Role: domainauth.RoleDeveloper}, "admin")
	if err != nil {
		t.Fatalf("create service account: %v", err)
	}
	expiresAt := time.Now().Add(-time.Minute)
	result, err := service.CreateAPIToken(context.Background(), APITokenInput{Name: "expired-token", SubjectID: account.ID, ExpiresAt: &expiresAt}, "admin")
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	if _, err := service.Authenticate(context.Background(), AuthenticateInput{Mode: "token", Token: result.Token}); err == nil {
		t.Fatalf("expected expired token to fail")
	}
	rotated, err := service.RotateAPIToken(context.Background(), result.Metadata.ID, "admin")
	if err != nil {
		t.Fatalf("rotate token: %v", err)
	}
	if _, err := service.RevokeAPIToken(context.Background(), rotated.Metadata.ID, "admin"); err != nil {
		t.Fatalf("revoke token: %v", err)
	}
	if _, err := service.Authenticate(context.Background(), AuthenticateInput{Mode: "token", Token: rotated.Token}); err == nil {
		t.Fatalf("expected revoked token to fail")
	}
}

func TestOIDCProviderAuthentication(t *testing.T) {
	service := NewService(NewMemoryStore(), nil)
	service.SetOIDCProvider(fakeOIDCProvider{})
	subject, err := service.Authenticate(context.Background(), AuthenticateInput{Mode: "oidc", Token: "valid", OIDCIssuer: "https://issuer.example", OIDCAudience: "nivora"})
	if err != nil {
		t.Fatalf("authenticate oidc: %v", err)
	}
	if subject.ID != "user-oidc" || subject.AuthMode != "oidc" || len(subject.Roles) != 1 || subject.Roles[0] != domainauth.RoleMaintainer {
		t.Fatalf("unexpected oidc subject: %#v", subject)
	}
	if _, err := service.Authenticate(context.Background(), AuthenticateInput{Mode: "oidc", Token: "bad"}); err == nil {
		t.Fatalf("expected invalid oidc token to fail")
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

type fakeOIDCProvider struct{}

func (fakeOIDCProvider) Validate(ctx context.Context, token string, issuer string, audience string) (OIDCClaims, error) {
	if token != "valid" || issuer != "https://issuer.example" || audience != "nivora" {
		return OIDCClaims{}, ErrUnauthorized
	}
	return OIDCClaims{Subject: "user-oidc", Username: "oidc-user", Roles: []string{domainauth.RoleMaintainer}}, nil
}
