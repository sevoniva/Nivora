package policy

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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

func defaultString(input string, fallback string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return fallback
	}
	return input
}
