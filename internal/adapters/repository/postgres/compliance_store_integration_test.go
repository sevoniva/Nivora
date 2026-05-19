package postgres

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	domaincompliance "github.com/sevoniva/nivora/internal/domain/compliance"
)

func TestPostgresIntegrationComplianceEvidenceAndRetentionRecovery(t *testing.T) {
	db := newPostgresIntegration(t, true)
	defer db.cleanup()
	ctx := context.Background()
	now := fixedIntegrationTime()
	store := NewComplianceStore(db.pool)

	bundle := domaincompliance.EvidenceBundle{
		ID:          "evb-recover",
		SubjectType: "release",
		SubjectID:   "rel-recover",
		ScopeType:   "project",
		ScopeID:     "project-a",
		Summary:     "release evidence",
		Artifacts:   []any{map[string]any{"name": "demo", "token": "[REDACTED]"}},
		GeneratedAt: now,
	}
	if err := store.SaveEvidenceBundle(ctx, bundle); err != nil {
		t.Fatalf("save evidence bundle: %v", err)
	}
	policy := domaincompliance.RetentionPolicy{
		ID:             "retention-project:project-a",
		ScopeType:      "project",
		ScopeID:        "project-a",
		LogDays:        14,
		AuditDays:      730,
		EventDays:      180,
		EvidenceDays:   730,
		ImmutableAudit: true,
		UpdatedAt:      now,
	}
	if err := store.SaveRetentionPolicy(ctx, policy); err != nil {
		t.Fatalf("save retention policy: %v", err)
	}

	store = NewComplianceStore(db.restart(t))
	loaded, err := store.GetEvidenceBundle(ctx, bundle.ID)
	if err != nil {
		t.Fatalf("reload evidence bundle: %v", err)
	}
	if loaded.SubjectID != bundle.SubjectID || loaded.ScopeID != bundle.ScopeID {
		t.Fatalf("loaded evidence mismatch: %#v", loaded)
	}
	body, err := json.Marshal(loaded)
	if err != nil {
		t.Fatalf("marshal loaded evidence: %v", err)
	}
	if strings.Contains(string(body), "placeholder-sensitive-token") {
		t.Fatalf("loaded evidence leaked secret-like value: %s", string(body))
	}
	results, err := store.SearchEvidenceBundles(ctx, "release", "rel-recover")
	if err != nil || len(results) != 1 || results[0].ID != bundle.ID {
		t.Fatalf("search evidence = %#v err=%v", results, err)
	}
	loadedPolicy, err := store.GetRetentionPolicy(ctx, "project", "project-a")
	if err != nil {
		t.Fatalf("reload retention policy: %v", err)
	}
	if loadedPolicy.AuditDays != 730 || !loadedPolicy.ImmutableAudit {
		t.Fatalf("loaded retention mismatch: %#v", loadedPolicy)
	}
}
