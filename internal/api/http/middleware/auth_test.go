package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sevoniva/nivora/internal/api/http/dto"
	domainauth "github.com/sevoniva/nivora/internal/domain/auth"
	"github.com/sevoniva/nivora/internal/infra/config"
	authusecase "github.com/sevoniva/nivora/internal/usecase/auth"
)

func TestAuthenticateUnauthorizedInTokenMode(t *testing.T) {
	service := authusecase.NewService(authusecase.NewMemoryStore(), nil)
	cfg := config.AuthConfig{Enabled: true, Mode: "token", StaticTokenEnv: "NIVORA_TEST_AUTH_TOKEN"}
	handler := Authenticate(cfg, service, testErrorWriter)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestAuthenticateOIDCMode(t *testing.T) {
	service := authusecase.NewService(authusecase.NewMemoryStore(), nil)
	service.SetOIDCProvider(testOIDCProvider{})
	cfg := config.AuthConfig{Enabled: true, Mode: "oidc"}
	cfg.OIDC.Issuer = "https://issuer.example"
	cfg.OIDC.ClientID = "nivora"
	handler := Authenticate(cfg, service, testErrorWriter)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		subject := Subject(r.Context())
		if subject.ID != "oidc-user" {
			t.Fatalf("subject = %#v", subject)
		}
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer valid")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestRequirePermissionForbidden(t *testing.T) {
	service := authusecase.NewService(authusecase.NewMemoryStore(), nil)
	handler := RequirePermission(service, domainauth.PermissionCredentialManage, testErrorWriter, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	ctx := context.WithValue(context.Background(), subjectKey{}, domainauth.Subject{ID: "viewer", Roles: []string{domainauth.RoleViewer}})
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d", rec.Code)
	}
}

func testErrorWriter(w http.ResponseWriter, r *http.Request, status int, payload dto.ErrorResponse) {
	w.WriteHeader(status)
}

type testOIDCProvider struct{}

func (testOIDCProvider) Validate(ctx context.Context, token string, issuer string, audience string) (authusecase.OIDCClaims, error) {
	if token != "valid" || issuer != "https://issuer.example" || audience != "nivora" {
		return authusecase.OIDCClaims{}, errors.New("invalid")
	}
	return authusecase.OIDCClaims{Subject: "oidc-user", Roles: []string{domainauth.RoleViewer}}, nil
}
