package policy

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	domainpolicy "github.com/sevoniva/nivora/internal/domain/policy"
)

type Service struct {
	store Store
	now   func() time.Time
}

func NewService(store Store) *Service {
	return &Service{store: store, now: time.Now}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (domainpolicy.Policy, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return domainpolicy.Policy{}, fmt.Errorf("%w: policy name is required", ErrInvalid)
	}
	projectID := strings.TrimSpace(input.ProjectID)
	environmentID := strings.TrimSpace(input.EnvironmentID)
	existing, _ := s.store.List(ctx, projectID, environmentID)
	for _, candidate := range existing {
		if strings.EqualFold(candidate.Name, name) {
			return domainpolicy.Policy{}, fmt.Errorf("%w: policy %q", ErrAlreadyExists, name)
		}
	}
	now := s.now().UTC()
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	policy := domainpolicy.Policy{
		ID:                 defaultID(input.ID),
		ProjectID:          projectID,
		EnvironmentID:      environmentID,
		Name:               name,
		Description:        strings.TrimSpace(input.Description),
		Type:               defaultString(input.Type, "security"),
		Mode:               defaultString(input.Mode, "warn"),
		CriticalDeny:       input.CriticalDeny,
		HighWarn:           input.HighWarn,
		RequireDigest:      input.RequireDigest,
		ApprovalOnCritical: input.ApprovalOnCritical,
		Labels:             copyMap(input.Labels),
		Metadata:           copyMap(input.Metadata),
		Enabled:            enabled,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if policy.CriticalDeny == 0 && policy.HighWarn == 0 && !policy.RequireDigest && !policy.ApprovalOnCritical {
		policy.HighWarn = 1
	}
	return s.store.Create(ctx, policy)
}

func (s *Service) Get(ctx context.Context, id string) (domainpolicy.Policy, error) {
	return s.store.Get(ctx, strings.TrimSpace(id))
}

func (s *Service) GetEnabled(ctx context.Context, id string) (domainpolicy.Policy, error) {
	policy, err := s.Get(ctx, id)
	if err != nil {
		return domainpolicy.Policy{}, err
	}
	if !policy.Enabled {
		return domainpolicy.Policy{}, fmt.Errorf("%w: %q", ErrDisabled, policy.ID)
	}
	return policy, nil
}

func (s *Service) List(ctx context.Context, projectID string, environmentID string) ([]domainpolicy.Policy, error) {
	return s.store.List(ctx, strings.TrimSpace(projectID), strings.TrimSpace(environmentID))
}

func (s *Service) Update(ctx context.Context, id string, input UpdateInput) (domainpolicy.Policy, error) {
	policy, err := s.store.Get(ctx, strings.TrimSpace(id))
	if err != nil {
		return domainpolicy.Policy{}, err
	}
	if input.ProjectID != nil {
		policy.ProjectID = strings.TrimSpace(*input.ProjectID)
	}
	if input.EnvironmentID != nil {
		policy.EnvironmentID = strings.TrimSpace(*input.EnvironmentID)
	}
	if input.Name != nil {
		policy.Name = strings.TrimSpace(*input.Name)
		if policy.Name == "" {
			return domainpolicy.Policy{}, fmt.Errorf("%w: policy name is required", ErrInvalid)
		}
	}
	if input.Description != nil {
		policy.Description = strings.TrimSpace(*input.Description)
	}
	if input.Type != nil {
		policy.Type = defaultString(*input.Type, "security")
	}
	if input.Mode != nil {
		policy.Mode = defaultString(*input.Mode, "warn")
	}
	if input.CriticalDeny != nil {
		policy.CriticalDeny = *input.CriticalDeny
	}
	if input.HighWarn != nil {
		policy.HighWarn = *input.HighWarn
	}
	if input.RequireDigest != nil {
		policy.RequireDigest = *input.RequireDigest
	}
	if input.ApprovalOnCritical != nil {
		policy.ApprovalOnCritical = *input.ApprovalOnCritical
	}
	if input.Labels != nil {
		policy.Labels = copyMap(input.Labels)
	}
	if input.Metadata != nil {
		policy.Metadata = copyMap(input.Metadata)
	}
	if input.Enabled != nil {
		policy.Enabled = *input.Enabled
	}
	policy.UpdatedAt = s.now().UTC()
	return s.store.Update(ctx, policy)
}

func (s *Service) Disable(ctx context.Context, id string) (domainpolicy.Policy, error) {
	enabled := false
	return s.Update(ctx, id, UpdateInput{Enabled: &enabled})
}

func (s *Service) Attach(ctx context.Context, policyID string, input AttachInput) (domainpolicy.PolicyAttachment, error) {
	policyID = strings.TrimSpace(policyID)
	if policyID == "" {
		return domainpolicy.PolicyAttachment{}, fmt.Errorf("%w: policy id is required", ErrInvalid)
	}
	if _, err := s.store.Get(ctx, policyID); err != nil {
		return domainpolicy.PolicyAttachment{}, err
	}
	scopeType := normalizeScopeType(input.ScopeType)
	if !isSupportedScopeType(scopeType) {
		return domainpolicy.PolicyAttachment{}, fmt.Errorf("%w: unsupported policy attachment scope type %q", ErrInvalid, input.ScopeType)
	}
	scopeID := strings.TrimSpace(input.ScopeID)
	if scopeType != "global" && scopeID == "" {
		return domainpolicy.PolicyAttachment{}, fmt.Errorf("%w: scopeId is required for %s policy attachments", ErrInvalid, scopeType)
	}
	if scopeType == "global" {
		scopeID = ""
	}
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	now := s.now().UTC()
	attachment := domainpolicy.PolicyAttachment{
		ID:        defaultAttachmentID(input.ID),
		PolicyID:  policyID,
		ScopeType: scopeType,
		ScopeID:   scopeID,
		Metadata:  copyMap(input.Metadata),
		Enabled:   enabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	return s.store.CreateAttachment(ctx, attachment)
}

func (s *Service) ListAttachments(ctx context.Context, input AttachmentListInput) ([]domainpolicy.PolicyAttachment, error) {
	filter := AttachmentListInput{
		PolicyID:  strings.TrimSpace(input.PolicyID),
		ScopeType: normalizeScopeType(input.ScopeType),
		ScopeID:   strings.TrimSpace(input.ScopeID),
		Enabled:   input.Enabled,
	}
	if filter.ScopeType != "" && !isSupportedScopeType(filter.ScopeType) {
		return nil, fmt.Errorf("%w: unsupported policy attachment scope type %q", ErrInvalid, input.ScopeType)
	}
	return s.store.ListAttachments(ctx, filter)
}

func (s *Service) ResolveEnabledForScope(ctx context.Context, input ResolveInput) (domainpolicy.Policy, bool, error) {
	projectID := strings.TrimSpace(input.ProjectID)
	environmentID := strings.TrimSpace(input.EnvironmentID)
	enabled := true
	filters := make([]AttachmentListInput, 0, 3)
	if environmentID != "" {
		filters = append(filters, AttachmentListInput{ScopeType: "environment", ScopeID: environmentID, Enabled: &enabled})
	}
	if projectID != "" {
		filters = append(filters, AttachmentListInput{ScopeType: "project", ScopeID: projectID, Enabled: &enabled})
	}
	filters = append(filters, AttachmentListInput{ScopeType: "global", Enabled: &enabled})

	seen := map[string]struct{}{}
	for _, filter := range filters {
		attachments, err := s.ListAttachments(ctx, filter)
		if err != nil {
			return domainpolicy.Policy{}, false, err
		}
		for _, attachment := range attachments {
			if _, ok := seen[attachment.PolicyID]; ok {
				continue
			}
			seen[attachment.PolicyID] = struct{}{}
			policy, err := s.GetEnabled(ctx, attachment.PolicyID)
			if err != nil {
				if errors.Is(err, ErrDisabled) || errors.Is(err, ErrNotFound) {
					continue
				}
				return domainpolicy.Policy{}, false, err
			}
			if !isSecurityPolicy(policy) {
				continue
			}
			return policy, true, nil
		}
	}
	return domainpolicy.Policy{}, false, nil
}

func defaultID(input string) string {
	input = strings.TrimSpace(input)
	if input != "" {
		return input
	}
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("policy-%d", time.Now().UnixNano())
	}
	return "policy-" + hex.EncodeToString(b[:])
}

func defaultAttachmentID(input string) string {
	input = strings.TrimSpace(input)
	if input != "" {
		return input
	}
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("policy-attachment-%d", time.Now().UnixNano())
	}
	return "policy-attachment-" + hex.EncodeToString(b[:])
}

func defaultString(input string, fallback string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return fallback
	}
	return input
}

func normalizeScopeType(input string) string {
	input = strings.TrimSpace(strings.ToLower(input))
	switch input {
	case "release-target", "release_target", "target":
		return "target"
	default:
		return input
	}
}

func isSupportedScopeType(scopeType string) bool {
	switch scopeType {
	case "global", "org", "project", "application", "environment", "target", "release", "deployment":
		return true
	default:
		return false
	}
}

func isSecurityPolicy(policy domainpolicy.Policy) bool {
	policyType := strings.ToLower(strings.TrimSpace(policy.Type))
	return policyType == "" || policyType == "security"
}
