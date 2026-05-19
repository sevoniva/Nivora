package approval

import (
	"context"
	"strings"
	"testing"
	"time"

	domainapproval "github.com/sevoniva/nivora/internal/domain/approval"
	"github.com/sevoniva/nivora/internal/domain/event"
	domainnotification "github.com/sevoniva/nivora/internal/domain/notification"
)

func TestApprovalRequestAndDecision(t *testing.T) {
	store := NewMemoryStore()
	service := NewService(store, nil, nil)

	request, err := service.RequestApproval(context.Background(), domainapproval.SubjectDeployment, "drun-1", "prod", "alice", "production deployment")
	if err != nil {
		t.Fatalf("request approval: %v", err)
	}
	if request.Status != domainapproval.StatusPending {
		t.Fatalf("status = %s", request.Status)
	}

	approved, err := service.Approve(context.Background(), request.ID, DecisionInput{Approver: "bob", Comment: "approved for window"})
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	if approved.Status != domainapproval.StatusApproved || len(approved.Decisions) != 1 {
		t.Fatalf("approved request = %#v", approved)
	}

	events, err := store.Events(context.Background())
	if err != nil {
		t.Fatalf("events: %v", err)
	}
	assertApprovalEvent(t, events, EventApprovalRequested)
	assertApprovalEvent(t, events, EventApprovalApproved)

	audits, err := store.Audits(context.Background())
	if err != nil {
		t.Fatalf("audits: %v", err)
	}
	if len(audits) < 2 {
		t.Fatalf("audit count = %d", len(audits))
	}
}

func TestApprovalReject(t *testing.T) {
	service := NewService(NewMemoryStore(), nil, nil)
	request, err := service.RequestApproval(context.Background(), domainapproval.SubjectRelease, "rexec-1", "prod", "alice", "release approval")
	if err != nil {
		t.Fatalf("request approval: %v", err)
	}
	rejected, err := service.Reject(context.Background(), request.ID, DecisionInput{Approver: "bob", Comment: "not ready"})
	if err != nil {
		t.Fatalf("reject: %v", err)
	}
	if rejected.Status != domainapproval.StatusRejected {
		t.Fatalf("status = %s", rejected.Status)
	}
}

func TestApprovalCancelAndExpireLifecycle(t *testing.T) {
	store := NewMemoryStore()
	service := NewService(store, nil, nil)
	now := time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	request, err := service.CreateApprovalRequest(context.Background(), ApprovalCreateInput{
		SubjectType: domainapproval.SubjectDeployment,
		SubjectID:   "drun-cancel",
		ExpiresAt:   ptrTime(now.Add(time.Hour)),
	})
	if err != nil {
		t.Fatalf("create approval: %v", err)
	}
	canceled, err := service.Cancel(context.Background(), request.ID, DecisionInput{Approver: "ops", Comment: "superseded"})
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if canceled.Status != domainapproval.StatusCanceled {
		t.Fatalf("status = %s", canceled.Status)
	}

	expiring, err := service.CreateApprovalRequest(context.Background(), ApprovalCreateInput{
		SubjectType: domainapproval.SubjectDeployment,
		SubjectID:   "drun-expire",
		ExpiresAt:   ptrTime(now.Add(-time.Minute)),
	})
	if err != nil {
		t.Fatalf("create expiring approval: %v", err)
	}
	expired, err := service.Approve(context.Background(), expiring.ID, DecisionInput{Approver: "ops"})
	if err == nil || !strings.Contains(err.Error(), "Expired") {
		t.Fatalf("expected expired error, got approval=%#v err=%v", expired, err)
	}
	if expired.Status != domainapproval.StatusExpired {
		t.Fatalf("expired status = %s", expired.Status)
	}
	events, err := store.Events(context.Background())
	if err != nil {
		t.Fatalf("events: %v", err)
	}
	assertApprovalEvent(t, events, EventApprovalCanceled)
	assertApprovalEvent(t, events, EventApprovalExpired)
}

func TestChangeWindowEvaluation(t *testing.T) {
	service := NewService(NewMemoryStore(), nil, nil)
	_, err := service.CreateChangeWindow(context.Background(), domainapproval.ChangeWindow{Name: "prod freeze", EnvironmentID: "prod", Allowed: false})
	if err != nil {
		t.Fatalf("create window: %v", err)
	}

	result, err := service.EvaluateChangeWindow(context.Background(), "prod")
	if err != nil {
		t.Fatalf("evaluate denied window: %v", err)
	}
	if result.Allowed {
		t.Fatalf("expected denied change window: %#v", result)
	}

	result, err = service.EvaluateChangeWindow(context.Background(), "dev")
	if err != nil {
		t.Fatalf("evaluate default window: %v", err)
	}
	if !result.Allowed {
		t.Fatalf("expected unconfigured env to allow: %#v", result)
	}
}

func TestChangeWindowTimezoneDayAndTimeEvaluation(t *testing.T) {
	service := NewService(NewMemoryStore(), nil, nil)
	_, err := service.CreateChangeWindow(context.Background(), domainapproval.ChangeWindow{
		Name:          "prod monday",
		EnvironmentID: "prod",
		Timezone:      "Asia/Shanghai",
		StartTime:     "09:00",
		EndTime:       "17:00",
		DaysOfWeek:    []string{"Monday"},
		Allowed:       true,
	})
	if err != nil {
		t.Fatalf("create window: %v", err)
	}

	allowed, err := service.EvaluateChangeWindowInput(context.Background(), ChangeWindowEvaluateInput{
		EnvironmentID: "prod",
		At:            "2026-05-18T02:00:00Z",
	})
	if err != nil {
		t.Fatalf("evaluate allowed: %v", err)
	}
	if !allowed.Allowed {
		t.Fatalf("expected allowed window: %#v", allowed)
	}

	outside, err := service.EvaluateChangeWindowInput(context.Background(), ChangeWindowEvaluateInput{
		EnvironmentID: "prod",
		At:            "2026-05-18T12:00:00Z",
	})
	if err != nil {
		t.Fatalf("evaluate outside: %v", err)
	}
	if !outside.Allowed || outside.Reason != "no matching change window for evaluation time" {
		t.Fatalf("expected default allow outside window, got %#v", outside)
	}
}

func TestNotificationNoop(t *testing.T) {
	store := NewMemoryStore()
	service := NewService(store, nil, nil)
	sent, err := service.SendNotification(context.Background(), domainnotification.Notification{Type: "approval", Channel: "noop", Subject: "test"})
	if err != nil {
		t.Fatalf("send notification: %v", err)
	}
	if sent.ID == "" || sent.CreatedAt.IsZero() {
		t.Fatalf("notification metadata = %#v", sent)
	}
	notifications, err := service.ListNotifications(context.Background())
	if err != nil {
		t.Fatalf("notifications: %v", err)
	}
	if len(notifications) != 1 {
		t.Fatalf("notifications = %d", len(notifications))
	}
}

func assertApprovalEvent(t *testing.T, events []event.Event, eventType string) {
	t.Helper()
	for _, evt := range events {
		if evt.Type == eventType {
			return
		}
	}
	t.Fatalf("missing event %s in %#v", eventType, events)
}

func ptrTime(value time.Time) *time.Time {
	return &value
}
