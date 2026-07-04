package postgres

import (
	"context"
	"strings"
	"testing"
	"time"

	domainapproval "github.com/sevoniva/nivora/internal/domain/approval"
	"github.com/sevoniva/nivora/internal/domain/audit"
	"github.com/sevoniva/nivora/internal/domain/event"
	domainnotification "github.com/sevoniva/nivora/internal/domain/notification"
	approvalusecase "github.com/sevoniva/nivora/internal/usecase/approval"
)

func TestApprovalStoreImplementsGovernanceInterface(t *testing.T) {
	var _ approvalusecase.Store = (*ApprovalStore)(nil)
}

func TestPostgresIntegrationApprovalGovernanceRecovery(t *testing.T) {
	db := newPostgresIntegration(t, true)
	defer db.cleanup()

	ctx := context.Background()
	now := fixedIntegrationTime()
	expiresAt := now.Add(2 * time.Hour)
	store := NewApprovalStore(db.pool)

	request := domainapproval.ApprovalRequest{
		ID:               "appr-recover",
		SubjectType:      domainapproval.SubjectDeployment,
		SubjectID:        "deploy-recover",
		EnvironmentID:    "prod",
		TargetType:       "kubernetes-yaml",
		TargetID:         "target-prod",
		Severity:         "high",
		PolicyResultID:   "policy-result-1",
		RequiredByPolicy: true,
		Status:           domainapproval.StatusPending,
		RequestedBy:      "maintainer-1",
		RequestedAt:      now,
		ExpiresAt:        &expiresAt,
		Reason:           "deployment requires approval",
		Participants: []domainapproval.ApprovalParticipant{
			{UserID: "approver-1", Role: "maintainer"},
		},
	}
	if err := store.SaveApproval(ctx, request); err != nil {
		t.Fatalf("save approval: %v", err)
	}

	window := domainapproval.ChangeWindow{
		ID:            "cwin-recover",
		Name:          "prod-window",
		EnvironmentID: "prod",
		Timezone:      "UTC",
		StartTime:     "09:00",
		EndTime:       "17:00",
		DaysOfWeek:    []string{"mon", "tue"},
		Allowed:       true,
		Metadata:      map[string]string{"scope": "prod"},
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := store.SaveChangeWindow(ctx, window); err != nil {
		t.Fatalf("save change window: %v", err)
	}

	notification := domainnotification.Notification{
		ID:         "ntf-recover",
		Type:       "approval",
		Channel:    "noop",
		Subject:    "Approval requested",
		Body:       "no secret values here",
		Recipients: []string{"approvers"},
		Metadata:   map[string]string{"approvalId": request.ID},
		CreatedAt:  now,
	}
	if err := store.SaveNotification(ctx, notification); err != nil {
		t.Fatalf("save notification: %v", err)
	}

	evt := event.Event{
		SpecVersion:     "1.0",
		ID:              "evt-approval-recover",
		Type:            approvalusecase.EventApprovalRequested,
		Source:          "nivora.approval",
		Subject:         request.ID,
		Time:            now,
		DataContentType: "application/json",
		Data:            map[string]any{"subjectId": request.SubjectID},
	}
	if err := store.AppendEvent(ctx, evt); err != nil {
		t.Fatalf("append event: %v", err)
	}

	auditEntry := audit.AuditLog{
		ID:        "audit-approval-recover",
		ActorID:   "maintainer-1",
		Action:    "approval.requested",
		Subject:   request.ID,
		Reason:    "approval recovery test",
		CreatedAt: now,
	}
	if err := store.AppendAudit(ctx, auditEntry); err != nil {
		t.Fatalf("append audit: %v", err)
	}

	store = NewApprovalStore(db.restart(t))

	loadedApproval, err := store.GetApproval(ctx, request.ID)
	if err != nil {
		t.Fatalf("reload approval: %v", err)
	}
	if loadedApproval.SubjectID != request.SubjectID || loadedApproval.Status != domainapproval.StatusPending || len(loadedApproval.Participants) != 1 {
		t.Fatalf("loaded approval mismatch: %#v", loadedApproval)
	}

	loadedWindow, err := store.GetChangeWindow(ctx, window.ID)
	if err != nil {
		t.Fatalf("reload change window: %v", err)
	}
	if loadedWindow.EnvironmentID != "prod" || len(loadedWindow.DaysOfWeek) != 2 || loadedWindow.Metadata["scope"] != "prod" {
		t.Fatalf("loaded change window mismatch: %#v", loadedWindow)
	}

	notifications, err := store.ListNotifications(ctx)
	if err != nil {
		t.Fatalf("reload notifications: %v", err)
	}
	if len(notifications) != 1 || notifications[0].ID != notification.ID || notifications[0].Metadata["approvalId"] != request.ID {
		t.Fatalf("loaded notifications mismatch: %#v", notifications)
	}
	for _, value := range []string{notifications[0].Body, strings.Join(notifications[0].Recipients, ","), notifications[0].Metadata["approvalId"]} {
		if strings.Contains(strings.ToLower(value), "password") || strings.Contains(strings.ToLower(value), "token") {
			t.Fatalf("notification recovery payload contains secret-like value: %q", value)
		}
	}

	events, err := store.Events(ctx)
	if err != nil {
		t.Fatalf("reload approval events: %v", err)
	}
	if len(events) != 1 || events[0].ID != evt.ID || events[0].Subject != request.ID {
		t.Fatalf("loaded events mismatch: %#v", events)
	}

	audits, err := store.Audits(ctx)
	if err != nil {
		t.Fatalf("reload approval audits: %v", err)
	}
	if len(audits) != 1 || audits[0].ID != auditEntry.ID || audits[0].Action != auditEntry.Action {
		t.Fatalf("loaded audits mismatch: %#v", audits)
	}

	compliance := NewComplianceStore(db.pool)
	valid, broken, err := compliance.VerifyAuditChain(ctx, "approval", "")
	if err != nil {
		t.Fatalf("verify approval audit chain: %v", err)
	}
	if !valid || len(broken) != 0 {
		t.Fatalf("approval audit chain invalid: valid=%v broken=%#v", valid, broken)
	}
}
