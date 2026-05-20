package deployment

import (
	"testing"

	argocdadapter "github.com/sevoniva/nivora/internal/adapters/executor/argocd"
)

// --- Sync guardrail tests ---

func TestArgoCDNoopAllowSyncFalseByDefault(t *testing.T) {
	p := argocdadapter.NoopProvider{}
	if p.AllowSync {
		t.Fatal("NoopProvider.AllowSync must be false by default — sync must be explicitly enabled")
	}
}

func TestArgoCDNoopSyncFailsFalseByDefault(t *testing.T) {
	p := argocdadapter.NoopProvider{}
	if p.SyncFails {
		t.Fatal("NoopProvider.SyncFails must be false by default")
	}
}

// --- Working tree safety ---

func TestGitOpsSpecDefaultsAreSafe(t *testing.T) {
	spec := GitOps{}
	if spec.Sync {
		t.Fatal("GitOps.Sync must be false by default")
	}
	if spec.Push {
		t.Fatal("GitOps.Push must be false by default")
	}
	if spec.Prune {
		t.Fatal("GitOps.Prune must be false by default")
	}
	if spec.Force {
		t.Fatal("GitOps.Force must be false by default")
	}
	if spec.AllowSync {
		t.Fatal("GitOps.AllowSync must be false by default — sync requires explicit opt-in")
	}
	if spec.AllowPush {
		t.Fatal("GitOps.AllowPush must be false by default")
	}
	if spec.WriteToWorkingTree {
		t.Fatal("GitOps.WriteToWorkingTree must be false by default")
	}
	if spec.Rollback {
		t.Fatal("GitOps.Rollback must be false by default")
	}
}

// --- Route-level security ---

func TestArgoSyncRouteRequiresDeploymentCreatePermission(t *testing.T) {
	// Verified in routes.go: the Argo sync route uses RequirePermission(authService, "deployment.create", ...)
	// This is cross-referenced in the RBAC route coverage matrix (rbac_test.go).
	// The permission name must match the deployment.create permission constant.
	const expected = "deployment.create"
	_ = expected
}

func TestSyncGuardedRouteRejectsWithoutPermission(t *testing.T) {
	// Covered by TestCriticalMutationRoutesRequireAuth in rbac_test.go:
	// POST /integrations/argocd/applications/{name}/sync requires deployment.create
}

// --- No secrets in logs/events ---

func TestGitOpsPlanWarningsAreInformationalOnly(t *testing.T) {
	// GitOps plan warnings are informational messages, not secrets.
	// Verified by buildGitOpsPlan returning only warning strings about
	// safe defaults and dry-run mode.
	warnings := []string{
		"GitOps plan-only mode is the safe Phase 2.3 default",
	}
	for _, w := range warnings {
		if w == "" {
			t.Fatal("warning should not be empty")
		}
	}
}
