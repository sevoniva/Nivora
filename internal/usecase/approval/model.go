package approval

import (
	domainapproval "github.com/sevoniva/nivora/internal/domain/approval"
	"github.com/sevoniva/nivora/internal/domain/audit"
	"github.com/sevoniva/nivora/internal/domain/event"
	domainnotification "github.com/sevoniva/nivora/internal/domain/notification"
)

const (
	EventApprovalRequested   = "devops.approval.requested"
	EventApprovalApproved    = "devops.approval.approved"
	EventApprovalRejected    = "devops.approval.rejected"
	EventApprovalCanceled    = "devops.approval.canceled"
	EventChangeWindowAllowed = "devops.change_window.allowed"
	EventChangeWindowDenied  = "devops.change_window.denied"
	EventNotificationSent    = "devops.notification.sent"
	EventNotificationFailed  = "devops.notification.failed"
)

type ApprovalCreateInput struct {
	SubjectType      string                               `json:"subjectType" yaml:"subjectType"`
	SubjectID        string                               `json:"subjectId" yaml:"subjectId"`
	EnvironmentID    string                               `json:"environmentId,omitempty" yaml:"environmentId,omitempty"`
	RequiredByPolicy bool                                 `json:"requiredByPolicy" yaml:"requiredByPolicy"`
	RequestedBy      string                               `json:"requestedBy,omitempty" yaml:"requestedBy,omitempty"`
	Reason           string                               `json:"reason,omitempty" yaml:"reason,omitempty"`
	Participants     []domainapproval.ApprovalParticipant `json:"participants,omitempty" yaml:"participants,omitempty"`
}

type DecisionInput struct {
	Approver string `json:"approver" yaml:"approver"`
	Comment  string `json:"comment,omitempty" yaml:"comment,omitempty"`
}

type ChangeWindowEvaluateInput struct {
	EnvironmentID string `json:"environmentId" yaml:"environmentId"`
	At            string `json:"at,omitempty" yaml:"at,omitempty"`
}

type Record struct {
	Approvals     []domainapproval.ApprovalRequest  `json:"approvals,omitempty"`
	ChangeWindows []domainapproval.ChangeWindow     `json:"changeWindows,omitempty"`
	Notifications []domainnotification.Notification `json:"notifications,omitempty"`
	Events        []event.Event                     `json:"events,omitempty"`
	Audits        []audit.AuditLog                  `json:"audits,omitempty"`
}
