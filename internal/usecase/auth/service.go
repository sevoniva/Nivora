package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sevoniva/nivora/internal/domain/audit"
	domainauth "github.com/sevoniva/nivora/internal/domain/auth"
	"github.com/sevoniva/nivora/internal/domain/event"
	"github.com/sevoniva/nivora/internal/ports/eventbus"
)

var ErrUnauthorized = errors.New("unauthorized")

type Service struct {
	store    Store
	eventBus eventbus.EventBus
	roles    map[string]domainauth.Role
	now      func() time.Time
	oidc     OIDCProvider
}

func NewService(store Store, bus eventbus.EventBus) *Service {
	return &Service{store: store, eventBus: bus, roles: DefaultRoles(), now: time.Now}
}

func (s *Service) SetOIDCProvider(provider OIDCProvider) {
	s.oidc = provider
}

func DefaultRoles() map[string]domainauth.Role {
	all := DefaultPermissions()
	return map[string]domainauth.Role{
		domainauth.RoleOwner:      {Name: domainauth.RoleOwner, Description: "Full administrative access", Permissions: all},
		domainauth.RoleAdmin:      {Name: domainauth.RoleAdmin, Description: "Administrative access", Permissions: all},
		domainauth.RoleMaintainer: {Name: domainauth.RoleMaintainer, Description: "Delivery operation access", Permissions: permissionsWithout(all, domainauth.PermissionCredentialManage, domainauth.PermissionPolicyManage, domainauth.PermissionAuditRead)},
		domainauth.RoleDeveloper:  {Name: domainauth.RoleDeveloper, Description: "Build and deploy access", Permissions: permissionsOnly(domainauth.PermissionProjectRead, domainauth.PermissionApplicationRead, domainauth.PermissionEnvironmentRead, domainauth.PermissionPipelineRun, domainauth.PermissionDeploymentCreate, domainauth.PermissionDeploymentCancel, domainauth.PermissionReleaseCreate)},
		domainauth.RoleViewer:     {Name: domainauth.RoleViewer, Description: "Read-only access", Permissions: permissionsOnly(domainauth.PermissionProjectRead, domainauth.PermissionApplicationRead, domainauth.PermissionEnvironmentRead)},
		domainauth.RoleAuditor:    {Name: domainauth.RoleAuditor, Description: "Read and audit access", Permissions: permissionsOnly(domainauth.PermissionProjectRead, domainauth.PermissionAuditRead)},
	}
}

func DefaultPermissions() []domainauth.Permission {
	actions := []string{
		domainauth.PermissionProjectRead,
		domainauth.PermissionProjectWrite,
		domainauth.PermissionApplicationRead,
		domainauth.PermissionApplicationWrite,
		domainauth.PermissionEnvironmentRead,
		domainauth.PermissionEnvironmentWrite,
		domainauth.PermissionPipelineRun,
		domainauth.PermissionDeploymentCreate,
		domainauth.PermissionDeploymentApprove,
		domainauth.PermissionDeploymentCancel,
		domainauth.PermissionReleaseCreate,
		domainauth.PermissionCredentialManage,
		domainauth.PermissionRunnerManage,
		domainauth.PermissionPolicyManage,
		domainauth.PermissionAuditRead,
	}
	return permissionsOnly(actions...)
}

func (s *Service) Authenticate(ctx context.Context, input AuthenticateInput) (domainauth.Subject, error) {
	mode := input.Mode
	if mode == "" {
		mode = "dev"
	}
	switch mode {
	case "disabled", "dev":
		username := input.DevUser
		if username == "" {
			username = "local-admin"
		}
		return domainauth.Subject{ID: username, Username: username, DisplayName: username, Roles: []string{domainauth.RoleOwner}, AuthMode: mode}, nil
	case "token":
		if input.Token == "" {
			return domainauth.Subject{}, ErrUnauthorized
		}
		if input.StaticToken != "" && input.Token == input.StaticToken {
			return domainauth.Subject{ID: "service-account", Username: "service-account", DisplayName: "Service Account", Roles: []string{domainauth.RoleOwner}, AuthMode: mode}, nil
		}
		token, err := s.store.FindTokenByHash(ctx, hashToken(input.Token))
		if err != nil {
			return domainauth.Subject{}, ErrUnauthorized
		}
		if token.RevokedAt != nil {
			return domainauth.Subject{}, ErrUnauthorized
		}
		if token.ExpiresAt != nil && !token.ExpiresAt.IsZero() && s.now().After(*token.ExpiresAt) {
			return domainauth.Subject{}, ErrUnauthorized
		}
		now := s.now()
		token.LastUsedAt = &now
		_ = s.store.SaveToken(ctx, token)
		return domainauth.Subject{ID: token.SubjectID, Username: token.SubjectID, DisplayName: token.Name, Roles: token.Roles, AuthMode: mode, ScopeType: token.ScopeType, ScopeID: token.ScopeID, TokenID: token.ID}, nil
	case "oidc", "oidc-placeholder":
		if s.oidc == nil || input.Token == "" {
			return domainauth.Subject{}, ErrUnauthorized
		}
		claims, err := s.oidc.Validate(ctx, input.Token, input.OIDCIssuer, input.OIDCAudience)
		if err != nil {
			return domainauth.Subject{}, ErrUnauthorized
		}
		if claims.Subject == "" {
			return domainauth.Subject{}, ErrUnauthorized
		}
		roles := claims.Roles
		if len(roles) == 0 {
			roles = []string{domainauth.RoleViewer}
		}
		username := claims.Username
		if username == "" {
			username = claims.Subject
		}
		return domainauth.Subject{ID: claims.Subject, Username: username, DisplayName: claims.DisplayName, Roles: roles, AuthMode: mode}, nil
	default:
		return domainauth.Subject{}, fmt.Errorf("unsupported auth mode %q", mode)
	}
}

func (s *Service) Evaluate(input EvaluateInput) domainauth.Decision {
	if input.Subject.ID == "" {
		return domainauth.Decision{Allowed: false, Reason: "subject is not authenticated", Action: input.Action}
	}
	for _, roleName := range input.Subject.Roles {
		role, ok := s.roles[roleName]
		if !ok {
			continue
		}
		for _, permission := range role.Permissions {
			if permission.Action == input.Action {
				return domainauth.Decision{Allowed: true, Reason: "allowed by role " + roleName, Action: input.Action, Roles: input.Subject.Roles}
			}
		}
	}
	return domainauth.Decision{Allowed: false, Reason: "permission denied", Action: input.Action, Roles: input.Subject.Roles}
}

func (s *Service) Roles() []domainauth.Role {
	roles := make([]domainauth.Role, 0, len(s.roles))
	for _, role := range s.roles {
		roles = append(roles, role)
	}
	return roles
}

func (s *Service) Permissions() []domainauth.Permission {
	return DefaultPermissions()
}

func (s *Service) ListUsers(ctx context.Context) ([]domainauth.User, error) {
	return s.store.ListUsers(ctx)
}

func (s *Service) CreateMembership(ctx context.Context, input MembershipInput, actorID string) (domainauth.Membership, error) {
	if input.UserID == "" {
		return domainauth.Membership{}, errors.New("membership userId is required")
	}
	if input.Role == "" {
		return domainauth.Membership{}, errors.New("membership role is required")
	}
	if _, ok := s.roles[input.Role]; !ok {
		return domainauth.Membership{}, fmt.Errorf("unknown role %q", input.Role)
	}
	now := s.now()
	membership := domainauth.Membership{
		ID:        newID("mbr"),
		ScopeType: input.ScopeType,
		ScopeID:   input.ScopeID,
		UserID:    input.UserID,
		Role:      input.Role,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.store.SaveMembership(ctx, membership); err != nil {
		return domainauth.Membership{}, err
	}
	_ = s.record(ctx, EventMembershipCreated, "membership created", actorID, membership.ID, map[string]any{"role": membership.Role, "userId": membership.UserID})
	return membership, nil
}

func (s *Service) ListMemberships(ctx context.Context, scopeType string, scopeID string) ([]domainauth.Membership, error) {
	return s.store.ListMemberships(ctx, scopeType, scopeID)
}

func (s *Service) CreateServiceAccount(ctx context.Context, input ServiceAccountInput, actorID string) (domainauth.ServiceAccount, error) {
	if input.Name == "" {
		return domainauth.ServiceAccount{}, errors.New("service account name is required")
	}
	if input.Role == "" {
		input.Role = domainauth.RoleDeveloper
	}
	if _, ok := s.roles[input.Role]; !ok {
		return domainauth.ServiceAccount{}, fmt.Errorf("unknown role %q", input.Role)
	}
	now := s.now()
	account := domainauth.ServiceAccount{
		ID:        newID("sa"),
		Name:      input.Name,
		ScopeType: input.ScopeType,
		ScopeID:   input.ScopeID,
		Role:      input.Role,
		Status:    "active",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.store.SaveServiceAccount(ctx, account); err != nil {
		return domainauth.ServiceAccount{}, err
	}
	_ = s.record(ctx, EventServiceAccountCreated, "service account created", actorID, account.ID, map[string]any{"role": account.Role, "scopeType": account.ScopeType})
	return account, nil
}

func (s *Service) ListServiceAccounts(ctx context.Context, scopeType string, scopeID string) ([]domainauth.ServiceAccount, error) {
	return s.store.ListServiceAccounts(ctx, scopeType, scopeID)
}

func (s *Service) CreateAPIToken(ctx context.Context, input APITokenInput, actorID string) (APITokenResult, error) {
	if input.SubjectID == "" {
		return APITokenResult{}, errors.New("token subjectId is required")
	}
	account, err := s.store.GetServiceAccount(ctx, input.SubjectID)
	if err != nil {
		return APITokenResult{}, err
	}
	raw := newRawToken()
	now := s.now()
	metadata := domainauth.TokenMetadata{
		ID:          newID("tok"),
		SubjectID:   account.ID,
		SubjectType: "service_account",
		Name:        input.Name,
		ScopeType:   account.ScopeType,
		ScopeID:     account.ScopeID,
		Roles:       []string{account.Role},
		TokenHash:   hashToken(raw),
		IssuedAt:    now,
		ExpiresAt:   input.ExpiresAt,
	}
	if metadata.Name == "" {
		metadata.Name = account.Name
	}
	if err := s.store.SaveToken(ctx, metadata); err != nil {
		return APITokenResult{}, err
	}
	_ = s.record(ctx, EventAPITokenCreated, "api token created", actorID, metadata.ID, map[string]any{"subjectId": metadata.SubjectID})
	public := metadata
	public.TokenHash = ""
	return APITokenResult{Metadata: public, Token: raw}, nil
}

func (s *Service) RotateAPIToken(ctx context.Context, tokenID string, actorID string) (APITokenResult, error) {
	metadata, err := s.store.GetToken(ctx, tokenID)
	if err != nil {
		return APITokenResult{}, err
	}
	raw := newRawToken()
	now := s.now()
	metadata.TokenHash = hashToken(raw)
	metadata.IssuedAt = now
	metadata.RevokedAt = nil
	metadata.LastUsedAt = nil
	if err := s.store.SaveToken(ctx, metadata); err != nil {
		return APITokenResult{}, err
	}
	_ = s.record(ctx, EventAPITokenRotated, "api token rotated", actorID, metadata.ID, map[string]any{"subjectId": metadata.SubjectID})
	public := metadata
	public.TokenHash = ""
	return APITokenResult{Metadata: public, Token: raw}, nil
}

func (s *Service) RevokeAPIToken(ctx context.Context, tokenID string, actorID string) (domainauth.TokenMetadata, error) {
	metadata, err := s.store.GetToken(ctx, tokenID)
	if err != nil {
		return domainauth.TokenMetadata{}, err
	}
	now := s.now()
	metadata.RevokedAt = &now
	if err := s.store.SaveToken(ctx, metadata); err != nil {
		return domainauth.TokenMetadata{}, err
	}
	_ = s.record(ctx, EventAPITokenRevoked, "api token revoked", actorID, metadata.ID, map[string]any{"subjectId": metadata.SubjectID})
	metadata.TokenHash = ""
	return metadata, nil
}

func (s *Service) ListAPITokens(ctx context.Context, subjectID string) ([]domainauth.TokenMetadata, error) {
	return s.store.ListTokens(ctx, subjectID)
}

func (s *Service) RecordDenied(ctx context.Context, subject domainauth.Subject, action string, resource domainauth.Resource) {
	_ = s.record(ctx, EventPermissionDenied, "permission denied", subject.ID, resource.Type+":"+resource.ID, map[string]any{"action": action, "resourceType": resource.Type})
}

func (s *Service) record(ctx context.Context, eventType string, action string, actorID string, subject string, data map[string]any) error {
	evt := event.Event{ID: newID("evt"), SpecVersion: "1.0", Type: eventType, Source: "nivora.auth", Subject: subject, Time: s.now(), DataContentType: "application/json", Data: data}
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

func permissionsOnly(actions ...string) []domainauth.Permission {
	out := make([]domainauth.Permission, 0, len(actions))
	for _, action := range actions {
		out = append(out, domainauth.Permission{Action: action, Description: strings.ReplaceAll(action, ".", " ")})
	}
	return out
}

func permissionsWithout(all []domainauth.Permission, blocked ...string) []domainauth.Permission {
	block := map[string]bool{}
	for _, action := range blocked {
		block[action] = true
	}
	out := make([]domainauth.Permission, 0, len(all))
	for _, permission := range all {
		if !block[permission.Action] {
			out = append(out, permission)
		}
	}
	return out
}

func newID(prefix string) string {
	var b [6]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return prefix + "-" + hex.EncodeToString(b[:])
}

func newRawToken() string {
	var b [24]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("nivora_%d", time.Now().UnixNano())
	}
	return "nivora_" + hex.EncodeToString(b[:])
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return "sha256:" + hex.EncodeToString(sum[:])
}
