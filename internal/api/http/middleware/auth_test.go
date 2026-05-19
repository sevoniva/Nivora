package middleware

import (
	"context"
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
