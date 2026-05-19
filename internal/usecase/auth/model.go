package auth

import (
	"time"

	"github.com/sevoniva/nivora/internal/domain/audit"
	domainauth "github.com/sevoniva/nivora/internal/domain/auth"
	"github.com/sevoniva/nivora/internal/domain/event"
)

const (
	EventPermissionDenied  = "devops.auth.permission.denied"
	EventMembershipCreated = "devops.membership.created"
	EventMembershipUpdated = "devops.membership.updated"
	EventMembershipDeleted = "devops.membership.deleted"
)

type AuthenticateInput struct {
	Mode        string
	DevUser     string
	Token       string
	StaticToken string
}

type WhoAmI struct {
	Subject domainauth.Subject `json:"subject"`
}

type EvaluateInput struct {
	Subject  domainauth.Subject  `json:"subject"`
	Action   string              `json:"action"`
	Resource domainauth.Resource `json:"resource"`
}

type MembershipInput struct {
	UserID    string `json:"userId"`
	Role      string `json:"role"`
	ScopeType string `json:"scopeType"`
	ScopeID   string `json:"scopeId,omitempty"`
}

type TokenInfo struct {
	Authenticated bool      `json:"authenticated"`
	Mode          string    `json:"mode"`
	SubjectID     string    `json:"subjectId,omitempty"`
	IssuedAt      time.Time `json:"issuedAt,omitempty"`
}

type Record struct {
	Users       []domainauth.User       `json:"users,omitempty"`
	Memberships []domainauth.Membership `json:"memberships,omitempty"`
	Events      []event.Event           `json:"events,omitempty"`
	Audits      []audit.AuditLog        `json:"auditLogs,omitempty"`
}
