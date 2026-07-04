package postgres

import (
	"context"
	"strings"
	"testing"
	"time"

	domainsecurity "github.com/sevoniva/nivora/internal/domain/security"
	securityusecase "github.com/sevoniva/nivora/internal/usecase/security"
)

func TestSecurityStoreImplementsInterface(t *testing.T) {
	var _ securityusecase.Store = (*SecurityStore)(nil)
}

func TestSecurityPolicyResultMigrationIsReversibleAndIndexed(t *testing.T) {
	up := readMigration(t, "000016_security_policy_results.up.sql")
	down := readMigration(t, "000016_security_policy_results.down.sql")

	if !strings.Contains(up, "CREATE TABLE IF NOT EXISTS security_policy_results") {
		t.Fatal("up migration missing security_policy_results table")
	}
	if !strings.Contains(down, "DROP TABLE IF EXISTS security_policy_results") {
		t.Fatal("down migration missing security_policy_results table drop")
	}
	for _, index := range []string{
		"idx_security_policy_results_policy",
		"idx_security_policy_results_subject",
		"idx_security_policy_results_project",
		"idx_security_policy_results_environment",
	} {
		if !strings.Contains(up, index) {
			t.Fatalf("up migration missing index %s", index)
		}
	}
}

func TestPostgresIntegrationSecurityPolicyResultRecovery(t *testing.T) {
	db := newPostgresIntegration(t, true)
	defer db.cleanup()
	ctx := context.Background()
	store := NewSecurityStore(db.pool)
	evaluatedAt := time.Date(2026, 7, 5, 8, 0, 0, 0, time.UTC)
	result := domainsecurity.PolicyResult{
		ID:            "policy-result-recovery",
		PolicyID:      "policy-digest-required",
		SubjectType:   domainsecurity.SubjectArtifact,
		SubjectID:     "registry.example.invalid/team/app:latest",
		ProjectID:     "project-a",
		EnvironmentID: "env-prod",
		Decision:      domainsecurity.GateDeny,
		Reason:        "artifact digest is required",
		Findings: []domainsecurity.SecurityFinding{{
			ID:       "finding-policy",
			Severity: domainsecurity.SeverityHigh,
			Category: domainsecurity.CategoryPolicy,
			Target:   "registry.example.invalid/team/app:latest",
			Title:    "mutable artifact reference",
		}},
		EvaluatedAt: evaluatedAt,
	}
	if err := store.SavePolicyResult(ctx, result); err != nil {
		t.Fatalf("save policy result: %v", err)
	}

	restartedPool := db.restart(t)
	store = NewSecurityStore(restartedPool)
	loaded, err := store.GetPolicyResult(ctx, result.ID)
	if err != nil {
		t.Fatalf("reload policy result: %v", err)
	}
	if loaded.ID != result.ID || loaded.PolicyID != result.PolicyID || loaded.ProjectID != "project-a" || loaded.Decision != domainsecurity.GateDeny {
		t.Fatalf("loaded policy result = %#v", loaded)
	}
	if len(loaded.Findings) != 1 || loaded.Findings[0].ID != "finding-policy" {
		t.Fatalf("loaded findings = %#v", loaded.Findings)
	}

	results, err := store.ListPolicyResults(ctx, securityusecase.ListPolicyResultsInput{
		PolicyID:  result.PolicyID,
		ProjectID: "project-a",
		Decision:  domainsecurity.GateDeny,
	})
	if err != nil {
		t.Fatalf("list policy results: %v", err)
	}
	if len(results) != 1 || results[0].ID != result.ID {
		t.Fatalf("listed policy results = %#v", results)
	}

	results, err = store.ListPolicyResults(ctx, securityusecase.ListPolicyResultsInput{ProjectID: "project-b"})
	if err != nil {
		t.Fatalf("list policy results for other project: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("cross-project policy results leaked: %#v", results)
	}
}
