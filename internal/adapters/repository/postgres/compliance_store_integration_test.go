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

	oldEvidence := domaincompliance.EvidenceBundle{ID: "evb-retention-old", SubjectType: "release", SubjectID: "rel-old", ScopeType: "project", ScopeID: "project-a", Summary: "old evidence", GeneratedAt: now.AddDate(0, 0, -60)}
	newEvidence := domaincompliance.EvidenceBundle{ID: "evb-retention-new", SubjectType: "release", SubjectID: "rel-new", ScopeType: "project", ScopeID: "project-a", Summary: "new evidence", GeneratedAt: now.AddDate(0, 0, -5)}
	for _, bundle := range []domaincompliance.EvidenceBundle{oldEvidence, newEvidence} {
		if err := store.SaveEvidenceBundle(ctx, bundle); err != nil {
			t.Fatalf("save retention evidence %s: %v", bundle.ID, err)
		}
	}
	if err := store.AppendAuditRecord(ctx, AuditRecord{ID: "audit-retention-old", ActorID: "ops", Action: "old audit", SubjectType: "release", SubjectID: "rel-old", Subject: "release/rel-old", ScopeType: "project", ScopeID: "project-a", CreatedAt: now.AddDate(0, 0, -60)}); err != nil {
		t.Fatalf("append retention audit record: %v", err)
	}
	retentionRunPolicy := domaincompliance.RetentionPolicy{ScopeType: "project", ScopeID: "project-a", AuditDays: 30, EvidenceDays: 30, LogDays: 30, EventDays: 30, ImmutableAudit: true}
	preview, err := store.PreviewRetention(ctx, retentionRunPolicy, now)
	if err != nil {
		t.Fatalf("preview retention: %v", err)
	}
	if got := postgresRetentionTarget(t, preview, domaincompliance.RetentionTargetEvidence); got.Candidates != 1 || got.Deleted != 0 {
		t.Fatalf("evidence preview = %#v", got)
	}
	if got := postgresRetentionTarget(t, preview, domaincompliance.RetentionTargetAudit); !got.Immutable || got.Candidates == 0 || got.Deleted != 0 {
		t.Fatalf("audit preview = %#v", got)
	}
	applied, err := store.ApplyRetention(ctx, retentionRunPolicy, now)
	if err != nil {
		t.Fatalf("apply retention: %v", err)
	}
	if got := postgresRetentionTarget(t, applied, domaincompliance.RetentionTargetEvidence); got.Candidates != 1 || got.Deleted != 1 {
		t.Fatalf("evidence apply = %#v", got)
	}
	if _, err := store.GetEvidenceBundle(ctx, oldEvidence.ID); err == nil {
		t.Fatalf("expected old evidence to be deleted")
	}
	if _, err := store.GetEvidenceBundle(ctx, newEvidence.ID); err != nil {
		t.Fatalf("new evidence should remain: %v", err)
	}
}

func postgresRetentionTarget(t *testing.T, targets []domaincompliance.RetentionTargetResult, name string) domaincompliance.RetentionTargetResult {
	t.Helper()
	for _, target := range targets {
		if target.Target == name {
			return target
		}
	}
	t.Fatalf("retention target %s not found in %#v", name, targets)
	return domaincompliance.RetentionTargetResult{}
}
