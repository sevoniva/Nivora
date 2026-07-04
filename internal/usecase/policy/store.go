package policy

import (
	"context"
	"fmt"
	"sort"
	"sync"

	domainpolicy "github.com/sevoniva/nivora/internal/domain/policy"
)

type Store interface {
	Create(ctx context.Context, policy domainpolicy.Policy) (domainpolicy.Policy, error)
	Get(ctx context.Context, id string) (domainpolicy.Policy, error)
	List(ctx context.Context, projectID string, environmentID string) ([]domainpolicy.Policy, error)
	Update(ctx context.Context, policy domainpolicy.Policy) (domainpolicy.Policy, error)
	CreateAttachment(ctx context.Context, attachment domainpolicy.PolicyAttachment) (domainpolicy.PolicyAttachment, error)
	ListAttachments(ctx context.Context, input AttachmentListInput) ([]domainpolicy.PolicyAttachment, error)
}

type MemoryStore struct {
	mu          sync.RWMutex
	policies    map[string]domainpolicy.Policy
	attachments map[string]domainpolicy.PolicyAttachment
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		policies:    map[string]domainpolicy.Policy{},
		attachments: map[string]domainpolicy.PolicyAttachment{},
	}
}

func (s *MemoryStore) Create(ctx context.Context, policy domainpolicy.Policy) (domainpolicy.Policy, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.policies[policy.ID]; ok {
		return domainpolicy.Policy{}, fmt.Errorf("%w: id %q", ErrAlreadyExists, policy.ID)
	}
	s.policies[policy.ID] = copyPolicy(policy)
	return copyPolicy(policy), nil
}

func (s *MemoryStore) Get(ctx context.Context, id string) (domainpolicy.Policy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	policy, ok := s.policies[id]
	if !ok {
		return domainpolicy.Policy{}, fmt.Errorf("%w: %q", ErrNotFound, id)
	}
	return copyPolicy(policy), nil
}

func (s *MemoryStore) List(ctx context.Context, projectID string, environmentID string) ([]domainpolicy.Policy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domainpolicy.Policy, 0, len(s.policies))
	for _, policy := range s.policies {
		if projectID != "" && policy.ProjectID != projectID {
			continue
		}
		if environmentID != "" && policy.EnvironmentID != environmentID {
			continue
		}
		out = append(out, copyPolicy(policy))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *MemoryStore) Update(ctx context.Context, policy domainpolicy.Policy) (domainpolicy.Policy, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.policies[policy.ID]; !ok {
		return domainpolicy.Policy{}, fmt.Errorf("%w: %q", ErrNotFound, policy.ID)
	}
	s.policies[policy.ID] = copyPolicy(policy)
	return copyPolicy(policy), nil
}

func (s *MemoryStore) CreateAttachment(ctx context.Context, attachment domainpolicy.PolicyAttachment) (domainpolicy.PolicyAttachment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.attachments[attachment.ID]; ok {
		return domainpolicy.PolicyAttachment{}, fmt.Errorf("%w: attachment id %q", ErrAlreadyExists, attachment.ID)
	}
	for _, existing := range s.attachments {
		if existing.PolicyID == attachment.PolicyID &&
			existing.ScopeType == attachment.ScopeType &&
			existing.ScopeID == attachment.ScopeID {
			return domainpolicy.PolicyAttachment{}, fmt.Errorf("%w: policy %q already attached to %s/%s", ErrAlreadyExists, attachment.PolicyID, attachment.ScopeType, attachment.ScopeID)
		}
	}
	s.attachments[attachment.ID] = copyAttachment(attachment)
	return copyAttachment(attachment), nil
}

func (s *MemoryStore) ListAttachments(ctx context.Context, input AttachmentListInput) ([]domainpolicy.PolicyAttachment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domainpolicy.PolicyAttachment, 0, len(s.attachments))
	for _, attachment := range s.attachments {
		if input.PolicyID != "" && attachment.PolicyID != input.PolicyID {
			continue
		}
		if input.ScopeType != "" && attachment.ScopeType != input.ScopeType {
			continue
		}
		if input.ScopeID != "" && attachment.ScopeID != input.ScopeID {
			continue
		}
		if input.Enabled != nil && attachment.Enabled != *input.Enabled {
			continue
		}
		out = append(out, copyAttachment(attachment))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].PolicyID == out[j].PolicyID {
			if out[i].ScopeType == out[j].ScopeType {
				return out[i].ScopeID < out[j].ScopeID
			}
			return out[i].ScopeType < out[j].ScopeType
		}
		return out[i].PolicyID < out[j].PolicyID
	})
	return out, nil
}

func copyPolicy(policy domainpolicy.Policy) domainpolicy.Policy {
	policy.Labels = copyMap(policy.Labels)
	policy.Metadata = copyMap(policy.Metadata)
	return policy
}

func copyAttachment(attachment domainpolicy.PolicyAttachment) domainpolicy.PolicyAttachment {
	attachment.Metadata = copyMap(attachment.Metadata)
	return attachment
}

func copyMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
