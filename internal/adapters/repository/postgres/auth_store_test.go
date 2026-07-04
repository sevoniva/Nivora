package postgres

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	domainauth "github.com/sevoniva/nivora/internal/domain/auth"
	authusecase "github.com/sevoniva/nivora/internal/usecase/auth"
)

func TestPostgresIntegrationAuthAPITokenRotationPersistsHash(t *testing.T) {
	db := newPostgresIntegration(t, true)
	defer db.cleanup()
	ctx := context.Background()

	service := authusecase.NewService(NewAuthStore(db.pool), nil)
	account, err := service.CreateServiceAccount(ctx, authusecase.ServiceAccountInput{
		Name:      "ci-postgres",
		Role:      domainauth.RoleDeveloper,
		ScopeType: "project",
		ScopeID:   "project-auth",
	}, "admin")
	if err != nil {
		t.Fatalf("create service account: %v", err)
	}
	created, err := service.CreateAPIToken(ctx, authusecase.APITokenInput{Name: "ci-postgres-token", SubjectID: account.ID}, "admin")
	if err != nil {
		t.Fatalf("create api token: %v", err)
	}
	if created.Token == "" {
		t.Fatal("expected one-time raw token")
	}
	if created.Metadata.TokenHash != "" {
		t.Fatalf("create response leaked token hash: %#v", created.Metadata)
	}
	if _, err := service.Authenticate(ctx, authusecase.AuthenticateInput{Mode: "token", Token: created.Token}); err != nil {
		t.Fatalf("created token should authenticate: %v", err)
	}

	rotated, err := service.RotateAPIToken(ctx, created.Metadata.ID, "admin")
	if err != nil {
		t.Fatalf("rotate api token: %v", err)
	}
	if rotated.Token == "" || rotated.Token == created.Token {
		t.Fatalf("rotated token was not regenerated")
	}
	if rotated.Metadata.TokenHash != "" {
		t.Fatalf("rotate response leaked token hash: %#v", rotated.Metadata)
	}
	if _, err := service.Authenticate(ctx, authusecase.AuthenticateInput{Mode: "token", Token: created.Token}); err == nil {
		t.Fatal("old token authenticated after rotation")
	}
	subject, err := service.Authenticate(ctx, authusecase.AuthenticateInput{Mode: "token", Token: rotated.Token})
	if err != nil {
		t.Fatalf("rotated token should authenticate: %v", err)
	}
	if subject.ID != account.ID || subject.ScopeType != "project" || subject.ScopeID != "project-auth" {
		t.Fatalf("rotated token subject lost scope: %#v", subject)
	}

	service = authusecase.NewService(NewAuthStore(db.restart(t)), nil)
	if _, err := service.Authenticate(ctx, authusecase.AuthenticateInput{Mode: "token", Token: rotated.Token}); err != nil {
		t.Fatalf("rotated token should authenticate after pool restart: %v", err)
	}
	tokens, err := service.ListAPITokens(ctx, account.ID)
	if err != nil {
		t.Fatalf("list tokens: %v", err)
	}
	if len(tokens) != 1 || tokens[0].ID != created.Metadata.ID {
		t.Fatalf("listed tokens = %#v", tokens)
	}
	rendered := strings.ToLower(mustMarshalAuthTestJSON(t, tokens))
	for _, forbidden := range []string{"token_hash", "tokenhash", created.Token, rotated.Token} {
		if strings.Contains(rendered, strings.ToLower(forbidden)) {
			t.Fatalf("token listing leaked %q: %s", forbidden, rendered)
		}
	}

	revoked, err := service.RevokeAPIToken(ctx, rotated.Metadata.ID, "admin")
	if err != nil {
		t.Fatalf("revoke api token: %v", err)
	}
	if revoked.TokenHash != "" {
		t.Fatalf("revoke response leaked token hash: %#v", revoked)
	}
	if _, err := service.Authenticate(ctx, authusecase.AuthenticateInput{Mode: "token", Token: rotated.Token}); err == nil {
		t.Fatal("revoked token authenticated")
	}
}

func mustMarshalAuthTestJSON(t *testing.T, value any) string {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal auth test value: %v", err)
	}
	return string(body)
}
