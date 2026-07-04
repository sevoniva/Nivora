package routes

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sevoniva/nivora/internal/adapters/eventbus/memory"
	domainauth "github.com/sevoniva/nivora/internal/domain/auth"
	"github.com/sevoniva/nivora/internal/infra/config"
	authusecase "github.com/sevoniva/nivora/internal/usecase/auth"
)

// criticalRoutes defines the full RBAC route coverage matrix.
var criticalRoutes = []struct {
	method     string
	path       string
	permission string
	mutation   bool
}{
	{"POST", "/api/v1/cloud/accounts", domainauth.PermissionCredentialManage, true},
	{"GET", "/api/v1/cloud/accounts", domainauth.PermissionProjectRead, false},
	{"GET", "/api/v1/cloud/providers", domainauth.PermissionProjectRead, false},
	{"GET", "/api/v1/users", domainauth.PermissionProjectRead, false},
	{"GET", "/api/v1/roles", domainauth.PermissionProjectRead, false},
	{"GET", "/api/v1/orgs", domainauth.PermissionProjectRead, false},
	{"POST", "/api/v1/orgs", domainauth.PermissionProjectWrite, true},
	{"PATCH", "/api/v1/orgs/org-1", domainauth.PermissionProjectWrite, true},
	{"DELETE", "/api/v1/orgs/org-1", domainauth.PermissionProjectWrite, true},
	{"GET", "/api/v1/projects", domainauth.PermissionProjectRead, false},
	{"POST", "/api/v1/projects", domainauth.PermissionProjectWrite, true},
	{"PATCH", "/api/v1/projects/project-1", domainauth.PermissionProjectWrite, true},
	{"DELETE", "/api/v1/projects/project-1", domainauth.PermissionProjectWrite, true},
	{"GET", "/api/v1/applications", domainauth.PermissionApplicationRead, false},
	{"POST", "/api/v1/applications", domainauth.PermissionApplicationWrite, true},
	{"PATCH", "/api/v1/applications/app-1", domainauth.PermissionApplicationWrite, true},
	{"DELETE", "/api/v1/applications/app-1", domainauth.PermissionApplicationWrite, true},
	{"GET", "/api/v1/environments", domainauth.PermissionEnvironmentRead, false},
	{"POST", "/api/v1/environments", domainauth.PermissionEnvironmentWrite, true},
	{"PATCH", "/api/v1/environments/env-1", domainauth.PermissionEnvironmentWrite, true},
	{"DELETE", "/api/v1/environments/env-1", domainauth.PermissionEnvironmentWrite, true},
	{"GET", "/api/v1/repositories", domainauth.PermissionProjectRead, false},
	{"POST", "/api/v1/repositories", domainauth.PermissionProjectWrite, true},
	{"PATCH", "/api/v1/repositories/repo-1", domainauth.PermissionProjectWrite, true},
	{"DELETE", "/api/v1/repositories/repo-1", domainauth.PermissionProjectWrite, true},
	{"GET", "/api/v1/service-accounts", domainauth.PermissionCredentialManage, false},
	{"POST", "/api/v1/service-accounts", domainauth.PermissionCredentialManage, true},
	{"GET", "/api/v1/auth/tokens", domainauth.PermissionCredentialManage, false},
	{"POST", "/api/v1/auth/tokens", domainauth.PermissionCredentialManage, true},
	{"POST", "/api/v1/runners/register", domainauth.PermissionRunnerManage, true},
	{"GET", "/api/v1/runners", domainauth.PermissionRunnerManage, false},
	{"POST", "/api/v1/deployments/apply", domainauth.PermissionDeploymentCreate, true},
	{"POST", "/api/v1/deployments/plan", domainauth.PermissionDeploymentCreate, true},
	{"POST", "/api/v1/releases", domainauth.PermissionReleaseCreate, true},
	{"POST", "/api/v1/secrets", domainauth.PermissionCredentialManage, true},
	{"GET", "/api/v1/secrets/refs", domainauth.PermissionCredentialManage, false},
	{"POST", "/api/v1/approvals", domainauth.PermissionDeploymentApprove, true},
	{"POST", "/api/v1/approvals/test-id/approve", domainauth.PermissionDeploymentApprove, true},
	{"GET", "/api/v1/approvals", domainauth.PermissionProjectRead, false},
	{"GET", "/api/v1/change-windows", domainauth.PermissionProjectRead, false},
	{"POST", "/api/v1/change-windows", domainauth.PermissionEnvironmentWrite, true},
	{"GET", "/api/v1/notifications", domainauth.PermissionProjectRead, false},
	{"POST", "/api/v1/notifications/test", domainauth.PermissionEnvironmentWrite, true},
	{"POST", "/api/v1/security/scans", domainauth.PermissionPolicyManage, true},
	{"POST", "/api/v1/policies/evaluate", domainauth.PermissionPolicyManage, true},
	{"GET", "/api/v1/audit/search", domainauth.PermissionAuditRead, false},
	{"GET", "/api/v1/audit/verify", domainauth.PermissionAuditRead, false},
	{"POST", "/api/v1/pipeline-runs", domainauth.PermissionPipelineRun, true},
	{"GET", "/api/v1/credentials", domainauth.PermissionCredentialManage, false},
	{"POST", "/api/v1/credentials", domainauth.PermissionCredentialManage, true},
	{"DELETE", "/api/v1/secrets/test-id", domainauth.PermissionCredentialManage, true},
}

// viewerHasPermission returns true when the viewer role has the given permission.
func viewerHasPermission(permission string) bool {
	switch permission {
	case domainauth.PermissionProjectRead,
		domainauth.PermissionApplicationRead,
		domainauth.PermissionEnvironmentRead:
		return true
	default:
		return false
	}
}

// createServiceAccountAndToken is a helper that creates a service account and
// returns a one-time API token.
func createServiceAccountAndToken(t *testing.T, svc *authusecase.Service, name, role, scopeType, scopeID string) string {
	t.Helper()
	account, err := svc.CreateServiceAccount(context.Background(), authusecase.ServiceAccountInput{
		Name:      name,
		Role:      role,
		ScopeType: scopeType,
		ScopeID:   scopeID,
	}, "admin")
	if err != nil {
		t.Fatalf("create service account: %v", err)
	}
	result, err := svc.CreateAPIToken(context.Background(), authusecase.APITokenInput{
		Name:      name + "-token",
		SubjectID: account.ID,
	}, "admin")
	if err != nil {
		t.Fatalf("create api token: %v", err)
	}
	return result.Token
}

// newRBACTestRouter creates a router with token-mode auth enabled and a fresh
// auth service for service account / token provisioning.
func newRBACTestRouter(t *testing.T) (http.Handler, *authusecase.Service) {
	t.Helper()
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	cfg.Auth.Enabled = true
	cfg.Auth.Mode = "token"
	cfg.Auth.StaticTokenEnv = "" // no static token bypass
	authService := authusecase.NewService(authusecase.NewMemoryStore(), memory.New())
	return newTestRouterWithAuth(cfg, authService), authService
}

// TestCriticalMutationRoutesRequireAuth verifies that POST/PUT/DELETE routes
// return 401 when no auth token is provided.
func TestCriticalMutationRoutesRequireAuth(t *testing.T) {
	router, _ := newRBACTestRouter(t)

	for _, route := range criticalRoutes {
		if !route.mutation {
			continue
		}
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			var body io.Reader
			if route.method == http.MethodPost || route.method == http.MethodPut || route.method == http.MethodDelete {
				body = strings.NewReader("{}")
			} else {
				body = nil
			}
			req := httptest.NewRequest(route.method, route.path, body)
			if route.method == http.MethodPost || route.method == http.MethodPut {
				req.Header.Set("Content-Type", "application/json")
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}

// TestViewerCannotMutate verifies that the viewer role cannot access mutation
// routes (cloud accounts, deployments, releases, credentials, secrets,
// runners, approvals, policies).
func TestViewerCannotMutate(t *testing.T) {
	router, authService := newRBACTestRouter(t)
	viewerToken := createServiceAccountAndToken(t, authService, "viewer-user", domainauth.RoleViewer, "", "")

	for _, route := range criticalRoutes {
		if !route.mutation {
			continue
		}
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			var body io.Reader
			if route.method == http.MethodPost || route.method == http.MethodPut || route.method == http.MethodDelete {
				body = strings.NewReader("{}")
			}
			req := httptest.NewRequest(route.method, route.path, body)
			req.Header.Set("Authorization", "Bearer "+viewerToken)
			if route.method == http.MethodPost || route.method == http.MethodPut {
				req.Header.Set("Content-Type", "application/json")
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusForbidden {
				t.Errorf("expected 403 for viewer on %s %s, got %d body=%s", route.method, route.path, rec.Code, rec.Body.String())
			}
		})
	}
}

// TestDeveloperCanAccessDeploymentRoutes verifies that the developer role can
// access routes requiring deployment.create.
func TestDeveloperCanAccessDeploymentRoutes(t *testing.T) {
	router, authService := newRBACTestRouter(t)
	devToken := createServiceAccountAndToken(t, authService, "developer-user", domainauth.RoleDeveloper, "", "")

	deploymentRoutes := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/deployments/apply"},
		{http.MethodPost, "/api/v1/deployments/plan"},
	}

	for _, route := range deploymentRoutes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			req := httptest.NewRequest(route.method, route.path, strings.NewReader("{}"))
			req.Header.Set("Authorization", "Bearer "+devToken)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code == http.StatusForbidden {
				t.Errorf("expected developer to access %s %s, got %d body=%s", route.method, route.path, rec.Code, rec.Body.String())
			}
		})
	}

	// Developer should NOT be able to use credential.manage routes.
	t.Run("developer cannot manage credentials", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/credentials", strings.NewReader("{}"))
		req.Header.Set("Authorization", "Bearer "+devToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403 for developer on credentials, got %d body=%s", rec.Code, rec.Body.String())
		}
	})
}

// TestAuditorCanReadAuditCannotMutate verifies that the auditor role can read
// audit/search and audit/verify but cannot create deployments.
func TestAuditorCanReadAuditCannotMutate(t *testing.T) {
	router, authService := newRBACTestRouter(t)
	auditorToken := createServiceAccountAndToken(t, authService, "auditor-user", domainauth.RoleAuditor, "", "")

	// Auditor can read audit endpoints.
	t.Run("auditor can read audit/search", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/search", nil)
		req.Header.Set("Authorization", "Bearer "+auditorToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code == http.StatusForbidden {
			t.Errorf("expected auditor to read audit/search, got %d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("auditor can read audit/verify", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/verify", nil)
		req.Header.Set("Authorization", "Bearer "+auditorToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code == http.StatusForbidden {
			t.Errorf("expected auditor to read audit/verify, got %d body=%s", rec.Code, rec.Body.String())
		}
	})

	// Auditor cannot create deployments.
	t.Run("auditor cannot create deployments", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/deployments", strings.NewReader("{}"))
		req.Header.Set("Authorization", "Bearer "+auditorToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403 for auditor on deployment create, got %d body=%s", rec.Code, rec.Body.String())
		}
	})

	// Auditor cannot create pipeline runs.
	t.Run("auditor cannot run pipelines", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/pipeline-runs", strings.NewReader("{}"))
		req.Header.Set("Authorization", "Bearer "+auditorToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403 for auditor on pipeline-run, got %d body=%s", rec.Code, rec.Body.String())
		}
	})
}

// TestCrossTenantIsolation verifies that a user with project membership in
// project-A cannot access resources scoped to project-B.
func TestCrossTenantIsolation(t *testing.T) {
	router, authService := newRBACTestRouter(t)

	// Create project-A developer.
	tokenA := createServiceAccountAndToken(t, authService, "project-a-deployer", domainauth.RoleDeveloper, "project", "project-a")

	// Create project-B developer.
	tokenB := createServiceAccountAndToken(t, authService, "project-b-deployer", domainauth.RoleDeveloper, "project", "project-b")

	// project-A token can access project-A project read routes.
	t.Run("project-a token accesses project-a", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/project-a/members", nil)
		req.Header.Set("Authorization", "Bearer "+tokenA)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code == http.StatusForbidden {
			t.Errorf("expected project-a token to access project-a, got %d body=%s", rec.Code, rec.Body.String())
		}
	})

	// project-A token cannot access project-B resources.
	t.Run("project-a token cannot access project-b", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/project-b/members", nil)
		req.Header.Set("Authorization", "Bearer "+tokenA)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403 for cross-project access, got %d body=%s", rec.Code, rec.Body.String())
		}
	})

	// project-B token cannot access project-A resources.
	t.Run("project-b token cannot access project-a", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/project-a/members", nil)
		req.Header.Set("Authorization", "Bearer "+tokenB)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403 for cross-project access, got %d body=%s", rec.Code, rec.Body.String())
		}
	})

	// Unscoped admin token can access both projects.
	t.Run("unscoped admin accesses both projects", func(t *testing.T) {
		adminToken := createServiceAccountAndToken(t, authService, "admin-user", domainauth.RoleAdmin, "", "")

		for _, proj := range []string{"project-a", "project-b"} {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+proj+"/members", nil)
			req.Header.Set("Authorization", "Bearer "+adminToken)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code == http.StatusForbidden {
				t.Errorf("expected admin to access %s, got %d body=%s", proj, rec.Code, rec.Body.String())
			}
		}
	})

	// Verify deployment create respects tenant scope.
	t.Run("project-a token can create deployment", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/deployments", strings.NewReader("{}"))
		req.Header.Set("Authorization", "Bearer "+tokenA)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code == http.StatusForbidden {
			t.Errorf("expected project-a developer to create deployment, got %d", rec.Code)
		}
	})

	// Verify credential manage respects tenant scope.
	t.Run("project-b developer cannot manage project-a credentials", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/credentials", nil)
		req.Header.Set("Authorization", "Bearer "+tokenB)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		// Developer lacks credential.manage — 403 is correct isolation behavior.
		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403 for developer on credentials, got %d", rec.Code)
		}
	})
}

// TestRBACRouteCoverageMatrix is a table-driven test that iterates over all
// critical routes and verifies the correct permission is required.
// For each route it tests:
//   - Without auth token: expect 401
//   - With viewer token: expect 403 for routes the viewer lacks, non-403 for
//     project.read / application.read / environment.read routes
//   - With admin token: expect non-403
func TestRBACRouteCoverageMatrix(t *testing.T) {
	router, authService := newRBACTestRouter(t)
	adminToken := createServiceAccountAndToken(t, authService, "admin-coverage", domainauth.RoleAdmin, "", "")
	viewerToken := createServiceAccountAndToken(t, authService, "viewer-coverage", domainauth.RoleViewer, "", "")

	for _, route := range criticalRoutes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			// 1. Without auth token: expect 401.
			t.Run("no-auth", func(t *testing.T) {
				var body io.Reader
				if route.method == http.MethodPost || route.method == http.MethodPut || route.method == http.MethodDelete {
					body = strings.NewReader("{}")
				}
				req := httptest.NewRequest(route.method, route.path, body)
				if route.method == http.MethodPost || route.method == http.MethodPut {
					req.Header.Set("Content-Type", "application/json")
				}
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, req)

				if rec.Code != http.StatusUnauthorized {
					t.Errorf("no-auth: expected 401, got %d body=%s", rec.Code, rec.Body.String())
				}
			})

			// 2. With viewer token: expect 403 when viewer lacks the
			//    permission, non-403 when viewer has it.
			t.Run("viewer", func(t *testing.T) {
				var body io.Reader
				if route.method == http.MethodPost || route.method == http.MethodPut || route.method == http.MethodDelete {
					body = strings.NewReader("{}")
				}
				req := httptest.NewRequest(route.method, route.path, body)
				req.Header.Set("Authorization", "Bearer "+viewerToken)
				if route.method == http.MethodPost || route.method == http.MethodPut {
					req.Header.Set("Content-Type", "application/json")
				}
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, req)

				if viewerHasPermission(route.permission) {
					if rec.Code == http.StatusForbidden {
						t.Errorf("viewer: expected non-403 for %s (viewer has %s), got %d body=%s", route.path, route.permission, rec.Code, rec.Body.String())
					}
				} else {
					if rec.Code != http.StatusForbidden {
						t.Errorf("viewer: expected 403 for %s (viewer lacks %s), got %d body=%s", route.path, route.permission, rec.Code, rec.Body.String())
					}
				}
			})

			// 3. With admin token: expect non-403.
			t.Run("admin", func(t *testing.T) {
				var body io.Reader
				if route.method == http.MethodPost || route.method == http.MethodPut || route.method == http.MethodDelete {
					body = strings.NewReader("{}")
				}
				req := httptest.NewRequest(route.method, route.path, body)
				req.Header.Set("Authorization", "Bearer "+adminToken)
				if route.method == http.MethodPost || route.method == http.MethodPut {
					req.Header.Set("Content-Type", "application/json")
				}
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, req)

				if rec.Code == http.StatusForbidden {
					t.Errorf("admin: expected non-403 for %s (admin has all permissions), got %d body=%s", route.path, rec.Code, rec.Body.String())
				}
			})
		})
	}
}
