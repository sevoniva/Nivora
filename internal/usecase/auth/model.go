package auth

import (
	"context"
	"time"

	"github.com/sevoniva/nivora/internal/domain/audit"
	domainauth "github.com/sevoniva/nivora/internal/domain/auth"
	"github.com/sevoniva/nivora/internal/domain/event"
)

const (
	EventPermissionDenied      = "devops.auth.permission.denied"
	EventMembershipCreated     = "devops.membership.created"
	EventMembershipUpdated     = "devops.membership.updated"
	EventMembershipDeleted     = "devops.membership.deleted"
	EventServiceAccountCreated = "devops.service_account.created"
	EventAPITokenCreated       = "devops.api_token.created"
	EventAPITokenRotated       = "devops.api_token.rotated"
	EventAPITokenRevoked       = "devops.api_token.revoked"
)

type AuthenticateInput struct {
	Mode         string
	DevUser      string
	Token        string
	StaticToken  string
	OIDCIssuer   string
	OIDCAudience string
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
	TokenID       string    `json:"tokenId,omitempty"`
	IssuedAt      time.Time `json:"issuedAt,omitempty"`
}

type ServiceAccountInput struct {
	Name      string `json:"name"`
	ScopeType string `json:"scopeType"`
	ScopeID   string `json:"scopeId,omitempty"`
	Role      string `json:"role"`
}

type APITokenInput struct {
	Name      string     `json:"name"`
	SubjectID string     `json:"subjectId"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

type APITokenResult struct {
	Metadata domainauth.TokenMetadata `json:"metadata"`
	Token    string                   `json:"token,omitempty"`
}

type OIDCClaims struct {
	Subject     string   `json:"subject"`
	Username    string   `json:"username,omitempty"`
	DisplayName string   `json:"displayName,omitempty"`
	Email       string   `json:"email,omitempty"`
	Roles       []string `json:"roles,omitempty"`
	Issuer      string   `json:"issuer,omitempty"`
	Audience    string   `json:"audience,omitempty"`
}

type OIDCProvider interface {
	Validate(ctx context.Context, token string, issuer string, audience string) (OIDCClaims, error)
}

type Record struct {
	Users       []domainauth.User       `json:"users,omitempty"`
	Memberships []domainauth.Membership `json:"memberships,omitempty"`
	Events      []event.Event           `json:"events,omitempty"`
	Audits      []audit.AuditLog        `json:"auditLogs,omitempty"`
}
