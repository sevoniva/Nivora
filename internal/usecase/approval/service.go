package approval

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	domainapproval "github.com/sevoniva/nivora/internal/domain/approval"
	"github.com/sevoniva/nivora/internal/domain/audit"
	"github.com/sevoniva/nivora/internal/domain/event"
	domainnotification "github.com/sevoniva/nivora/internal/domain/notification"
	"github.com/sevoniva/nivora/internal/ports/eventbus"
	portnotification "github.com/sevoniva/nivora/internal/ports/notification"
)

type Service struct {
	store    Store
	notifier portnotification.Provider
	eventBus eventbus.EventBus
	now      func() time.Time
}

func NewService(store Store, notifier portnotification.Provider, bus eventbus.EventBus) *Service {
	return &Service{store: store, notifier: notifier, eventBus: bus, now: time.Now}
}

func (s *Service) CreateApprovalRequest(ctx context.Context, input ApprovalCreateInput) (domainapproval.ApprovalRequest, error) {
	if input.SubjectType == "" {
		return domainapproval.ApprovalRequest{}, errors.New("approval subjectType is required")
	}
	if input.SubjectID == "" {
		return domainapproval.ApprovalRequest{}, errors.New("approval subjectId is required")
	}
	now := s.now()
	request := domainapproval.ApprovalRequest{
		ID:               newID("appr"),
		SubjectType:      input.SubjectType,
		SubjectID:        input.SubjectID,
		EnvironmentID:    input.EnvironmentID,
		TargetType:       input.TargetType,
		TargetID:         input.TargetID,
		Severity:         input.Severity,
		PolicyResultID:   input.PolicyResultID,
		RequiredByPolicy: input.RequiredByPolicy,
		Status:           domainapproval.StatusPending,
		RequestedBy:      input.RequestedBy,
		RequestedAt:      now,
		ExpiresAt:        input.ExpiresAt,
		Reason:           input.Reason,
		Participants:     append([]domainapproval.ApprovalParticipant(nil), input.Participants...),
	}
	if err := s.store.SaveApproval(ctx, request); err != nil {
		return domainapproval.ApprovalRequest{}, err
	}
	_ = s.record(ctx, EventApprovalRequested, "Approval requested", input.RequestedBy, request.ID, map[string]any{"subjectType": request.SubjectType, "subjectId": request.SubjectID, "status": request.Status})
	_, _ = s.SendNotification(ctx, domainnotification.Notification{Type: "approval", Channel: "noop", Subject: "Approval requested", Body: request.Reason, Recipients: []string{"approvers"}, Metadata: map[string]string{"approvalId": request.ID}})
	return request, nil
}

func (s *Service) RequestApproval(ctx context.Context, subjectType string, subjectID string, environmentID string, requestedBy string, reason string) (domainapproval.ApprovalRequest, error) {
	return s.CreateApprovalRequest(ctx, ApprovalCreateInput{
		SubjectType:      subjectType,
		SubjectID:        subjectID,
		EnvironmentID:    environmentID,
		RequiredByPolicy: true,
		RequestedBy:      requestedBy,
		Reason:           reason,
	})
}

func (s *Service) EvaluateChangeWindowByEnvironment(ctx context.Context, environmentID string) (domainapproval.ChangeWindowResult, error) {
	return s.EvaluateChangeWindow(ctx, environmentID)
}

func (s *Service) ListApprovals(ctx context.Context) ([]domainapproval.ApprovalRequest, error) {
	return s.store.ListApprovals(ctx)
}

func (s *Service) GetApproval(ctx context.Context, id string) (domainapproval.ApprovalRequest, error) {
	return s.store.GetApproval(ctx, id)
}

func (s *Service) Approve(ctx context.Context, id string, input DecisionInput) (domainapproval.ApprovalRequest, error) {
	return s.decide(ctx, id, domainapproval.DecisionApprove, input)
}

func (s *Service) Reject(ctx context.Context, id string, input DecisionInput) (domainapproval.ApprovalRequest, error) {
	return s.decide(ctx, id, domainapproval.DecisionReject, input)
}

func (s *Service) Cancel(ctx context.Context, id string, input DecisionInput) (domainapproval.ApprovalRequest, error) {
	request, err := s.store.GetApproval(ctx, id)
	if err != nil {
		return domainapproval.ApprovalRequest{}, err
	}
	if expired, err := s.expireIfNeeded(ctx, request, input.Approver); err != nil || expired {
		latest, getErr := s.store.GetApproval(ctx, id)
		if getErr == nil {
			return latest, err
		}
		return domainapproval.ApprovalRequest{}, err
	}
	if request.Status != domainapproval.StatusPending {
		return domainapproval.ApprovalRequest{}, fmt.Errorf("approval request is already %s", request.Status)
	}
	now := s.now()
	request.Status = domainapproval.StatusCanceled
	request.Decisions = append(request.Decisions, domainapproval.ApprovalDecision{Approver: input.Approver, Decision: domainapproval.DecisionCancel, Comment: input.Comment, DecidedAt: now})
	if err := s.store.SaveApproval(ctx, request); err != nil {
		return domainapproval.ApprovalRequest{}, err
	}
	_ = s.record(ctx, EventApprovalCanceled, "Approval canceled", input.Approver, request.ID, map[string]any{"status": request.Status, "comment": input.Comment})
	return request, nil
}

func (s *Service) Expire(ctx context.Context, id string, input DecisionInput) (domainapproval.ApprovalRequest, error) {
	request, err := s.store.GetApproval(ctx, id)
	if err != nil {
		return domainapproval.ApprovalRequest{}, err
	}
	if request.Status != domainapproval.StatusPending {
		return domainapproval.ApprovalRequest{}, fmt.Errorf("approval request is already %s", request.Status)
	}
	now := s.now()
	request.Status = domainapproval.StatusExpired
	request.Decisions = append(request.Decisions, domainapproval.ApprovalDecision{Approver: input.Approver, Decision: domainapproval.DecisionExpire, Comment: input.Comment, DecidedAt: now})
	if err := s.store.SaveApproval(ctx, request); err != nil {
		return domainapproval.ApprovalRequest{}, err
	}
	_ = s.record(ctx, EventApprovalExpired, "Approval expired", input.Approver, request.ID, map[string]any{"status": request.Status, "comment": input.Comment})
	return request, nil
}

func (s *Service) CreateChangeWindow(ctx context.Context, window domainapproval.ChangeWindow) (domainapproval.ChangeWindow, error) {
	if window.Name == "" {
		return domainapproval.ChangeWindow{}, errors.New("change window name is required")
	}
	if window.EnvironmentID == "" {
		return domainapproval.ChangeWindow{}, errors.New("change window environmentId is required")
	}
	now := s.now()
	if window.ID == "" {
		window.ID = newID("cwin")
	}
	if window.Timezone == "" {
		window.Timezone = "UTC"
	}
	window.CreatedAt = now
	window.UpdatedAt = now
	if err := s.store.SaveChangeWindow(ctx, window); err != nil {
		return domainapproval.ChangeWindow{}, err
	}
	return window, nil
}

func (s *Service) ListChangeWindows(ctx context.Context) ([]domainapproval.ChangeWindow, error) {
	return s.store.ListChangeWindows(ctx)
}

func (s *Service) GetChangeWindow(ctx context.Context, id string) (domainapproval.ChangeWindow, error) {
	return s.store.GetChangeWindow(ctx, id)
}

func (s *Service) EvaluateChangeWindow(ctx context.Context, environmentID string) (domainapproval.ChangeWindowResult, error) {
	return s.EvaluateChangeWindowInput(ctx, ChangeWindowEvaluateInput{EnvironmentID: environmentID})
}

func (s *Service) EvaluateChangeWindowInput(ctx context.Context, input ChangeWindowEvaluateInput) (domainapproval.ChangeWindowResult, error) {
	if input.EnvironmentID == "" {
		return domainapproval.ChangeWindowResult{}, errors.New("environmentId is required")
	}
	evaluatedAt := s.now()
	if input.At != "" {
		parsed, err := time.Parse(time.RFC3339, input.At)
		if err != nil {
			return domainapproval.ChangeWindowResult{}, fmt.Errorf("change window at must be RFC3339: %w", err)
		}
		evaluatedAt = parsed
	}
	windows, err := s.store.ListChangeWindows(ctx)
	if err != nil {
		return domainapproval.ChangeWindowResult{}, err
	}
	matchedEnvironment := false
	for _, window := range windows {
		if window.EnvironmentID != input.EnvironmentID {
			continue
		}
		matchedEnvironment = true
		matches, reason, err := windowMatches(window, evaluatedAt)
		if err != nil {
			return domainapproval.ChangeWindowResult{}, err
		}
		if !matches {
			continue
		}
		result := domainapproval.ChangeWindowResult{WindowID: window.ID, EnvironmentID: input.EnvironmentID, Allowed: window.Allowed, EvaluatedAt: evaluatedAt}
		if window.Allowed {
			result.Reason = "change window allowed"
			_ = s.record(ctx, EventChangeWindowAllowed, "Change window allowed", "", window.ID, map[string]any{"environmentId": input.EnvironmentID})
			return result, nil
		}
		result.Reason = "change window denied: " + reason
		_ = s.record(ctx, EventChangeWindowDenied, "Change window denied", "", window.ID, map[string]any{"environmentId": input.EnvironmentID})
		return result, nil
	}
	reason := "no change window configured"
	if matchedEnvironment {
		reason = "no matching change window for evaluation time"
	}
	result := domainapproval.ChangeWindowResult{EnvironmentID: input.EnvironmentID, Allowed: true, Reason: reason, EvaluatedAt: evaluatedAt}
	_ = s.record(ctx, EventChangeWindowAllowed, "Change window allowed", "", input.EnvironmentID, map[string]any{"environmentId": input.EnvironmentID, "reason": result.Reason})
	return result, nil
}

func (s *Service) SendNotification(ctx context.Context, notification domainnotification.Notification) (domainnotification.Notification, error) {
	if notification.ID == "" {
		notification.ID = newID("ntf")
	}
	if notification.CreatedAt.IsZero() {
		notification.CreatedAt = s.now()
	}
	if notification.Channel == "" {
		notification.Channel = "noop"
	}
	if err := s.store.SaveNotification(ctx, notification); err != nil {
		return domainnotification.Notification{}, err
	}
	if s.notifier != nil {
		if err := s.notifier.Send(ctx, notification); err != nil {
			_ = s.record(ctx, EventNotificationFailed, "Notification failed", "", notification.ID, map[string]any{"channel": notification.Channel})
			return notification, err
		}
	}
	_ = s.record(ctx, EventNotificationSent, "Notification sent", "", notification.ID, map[string]any{"channel": notification.Channel, "type": notification.Type})
	return notification, nil
}

func (s *Service) ListNotifications(ctx context.Context) ([]domainnotification.Notification, error) {
	return s.store.ListNotifications(ctx)
}

func (s *Service) decide(ctx context.Context, id string, decision string, input DecisionInput) (domainapproval.ApprovalRequest, error) {
	request, err := s.store.GetApproval(ctx, id)
	if err != nil {
		return domainapproval.ApprovalRequest{}, err
	}
	if expired, err := s.expireIfNeeded(ctx, request, input.Approver); err != nil || expired {
		latest, getErr := s.store.GetApproval(ctx, id)
		if getErr == nil {
			return latest, err
		}
		return domainapproval.ApprovalRequest{}, err
	}
	if request.Status != domainapproval.StatusPending {
		return domainapproval.ApprovalRequest{}, fmt.Errorf("approval request is already %s", request.Status)
	}
	now := s.now()
	request.Decisions = append(request.Decisions, domainapproval.ApprovalDecision{Approver: input.Approver, Decision: decision, Comment: input.Comment, DecidedAt: now})
	eventType := EventApprovalApproved
	action := "Approval approved"
	request.Status = domainapproval.StatusApproved
	if decision == domainapproval.DecisionReject {
		eventType = EventApprovalRejected
		action = "Approval rejected"
		request.Status = domainapproval.StatusRejected
	}
	if err := s.store.SaveApproval(ctx, request); err != nil {
		return domainapproval.ApprovalRequest{}, err
	}
	_ = s.record(ctx, eventType, action, input.Approver, request.ID, map[string]any{"status": request.Status, "comment": input.Comment})
	return request, nil
}

func (s *Service) expireIfNeeded(ctx context.Context, request domainapproval.ApprovalRequest, actorID string) (bool, error) {
	if request.Status != domainapproval.StatusPending || request.ExpiresAt == nil || !s.now().After(*request.ExpiresAt) {
		return false, nil
	}
	request.Status = domainapproval.StatusExpired
	request.Decisions = append(request.Decisions, domainapproval.ApprovalDecision{Approver: actorID, Decision: domainapproval.DecisionExpire, Comment: "approval expired before decision", DecidedAt: s.now()})
	if err := s.store.SaveApproval(ctx, request); err != nil {
		return true, err
	}
	_ = s.record(ctx, EventApprovalExpired, "Approval expired", actorID, request.ID, map[string]any{"status": request.Status})
	return true, fmt.Errorf("approval request is already %s", request.Status)
}

func windowMatches(window domainapproval.ChangeWindow, at time.Time) (bool, string, error) {
	location := time.UTC
	if window.Timezone != "" {
		loaded, err := time.LoadLocation(window.Timezone)
		if err != nil {
			return false, "", fmt.Errorf("invalid change window timezone %q: %w", window.Timezone, err)
		}
		location = loaded
	}
	local := at.In(location)
	if len(window.DaysOfWeek) > 0 && !weekdayAllowed(window.DaysOfWeek, local.Weekday()) {
		return false, "day is outside change window", nil
	}
	if window.StartTime == "" && window.EndTime == "" {
		return true, "all-day change window", nil
	}
	start, err := parseClock(window.StartTime)
	if err != nil {
		return false, "", err
	}
	end, err := parseClock(window.EndTime)
	if err != nil {
		return false, "", err
	}
	current := local.Hour()*60 + local.Minute()
	inWindow := false
	if start <= end {
		inWindow = current >= start && current <= end
	} else {
		inWindow = current >= start || current <= end
	}
	if !inWindow {
		return false, "time is outside change window", nil
	}
	return true, "time is inside change window", nil
}

func parseClock(value string) (int, error) {
	parsed, err := time.Parse("15:04", value)
	if err != nil {
		return 0, fmt.Errorf("change window time %q must use HH:MM: %w", value, err)
	}
	return parsed.Hour()*60 + parsed.Minute(), nil
}

func weekdayAllowed(days []string, weekday time.Weekday) bool {
	want := strings.ToLower(weekday.String())
	for _, day := range days {
		normalized := strings.ToLower(strings.TrimSpace(day))
		if normalized == want || normalized == want[:3] {
			return true
		}
	}
	return false
}

func (s *Service) record(ctx context.Context, eventType string, action string, actorID string, subject string, data map[string]any) error {
	evt := event.Event{ID: newID("evt"), SpecVersion: "1.0", Type: eventType, Source: "nivora.governance", Subject: subject, Time: s.now(), DataContentType: "application/json", Data: data}
	if err := s.store.AppendEvent(ctx, evt); err != nil {
		return err
	}
	if err := s.store.AppendAudit(ctx, audit.AuditLog{ID: newID("audit"), ActorID: actorID, Action: action, Subject: subject, CreatedAt: s.now()}); err != nil {
		return err
	}
	if s.eventBus != nil {
		_ = s.eventBus.Publish(ctx, evt)
	}
	return nil
}

func newID(prefix string) string {
	var b [6]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return prefix + "-" + hex.EncodeToString(b[:])
}
