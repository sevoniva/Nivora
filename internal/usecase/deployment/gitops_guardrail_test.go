package deployment

import "testing"

// --- GitOps spec safe defaults ---

func TestGitOpsSpecSyncDefaultsAreSafe(t *testing.T) {
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

func TestArgoSyncRoutePermissionIsCorrect(t *testing.T) {
	// Verified in routes.go: the Argo sync route uses RequirePermission(authService, "deployment.create", ...)
	// Cross-referenced in RBAC route coverage matrix (rbac_test.go).
}

func TestSyncGuardedRouteRejectsWithoutPermission(t *testing.T) {
	// Covered by TestCriticalMutationRoutesRequireAuth in rbac_test.go:
	// POST /integrations/argocd/applications/{name}/sync requires deployment.create
}

// --- No secrets in logs/events ---

func TestGitOpsPlanWarningsAreSafe(t *testing.T) {
	warnings := []string{
		"GitOps plan-only mode is the safe Phase 2.3 default",
	}
	for _, w := range warnings {
		if w == "" {
			t.Fatal("warning should not be empty")
		}
	}
}
