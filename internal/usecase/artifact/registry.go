package artifact

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	domainartifact "github.com/sevoniva/nivora/internal/domain/artifact"
	portartifact "github.com/sevoniva/nivora/internal/ports/artifact"
)

var (
	ErrRegistryInvalid       = errors.New("artifact registry input is invalid")
	ErrRegistryNotFound      = errors.New("artifact registry not found")
	ErrRegistryAlreadyExists = errors.New("artifact registry already exists")
)

type RegistryCreateInput struct {
	ID            string            `json:"id,omitempty" yaml:"id,omitempty"`
	ProjectID     string            `json:"projectId,omitempty" yaml:"projectId,omitempty"`
	Name          string            `json:"name" yaml:"name"`
	Type          string            `json:"type,omitempty" yaml:"type,omitempty"`
	URL           string            `json:"url,omitempty" yaml:"url,omitempty"`
	Endpoint      string            `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
	Insecure      bool              `json:"insecure,omitempty" yaml:"insecure,omitempty"`
	CredentialRef string            `json:"credentialRef,omitempty" yaml:"credentialRef,omitempty"`
	Capabilities  []string          `json:"capabilities,omitempty" yaml:"capabilities,omitempty"`
	Labels        map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Enabled       *bool             `json:"enabled,omitempty" yaml:"enabled,omitempty"`
}

type RegistryUpdateInput struct {
	ProjectID     *string           `json:"projectId,omitempty" yaml:"projectId,omitempty"`
	Name          *string           `json:"name,omitempty" yaml:"name,omitempty"`
	Type          *string           `json:"type,omitempty" yaml:"type,omitempty"`
	URL           *string           `json:"url,omitempty" yaml:"url,omitempty"`
	Endpoint      *string           `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
	Insecure      *bool             `json:"insecure,omitempty" yaml:"insecure,omitempty"`
	CredentialRef *string           `json:"credentialRef,omitempty" yaml:"credentialRef,omitempty"`
	Capabilities  []string          `json:"capabilities,omitempty" yaml:"capabilities,omitempty"`
	Labels        map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Enabled       *bool             `json:"enabled,omitempty" yaml:"enabled,omitempty"`
}

type RegistryValidationResult struct {
	Valid      bool     `json:"valid"`
	RegistryID string   `json:"registryId,omitempty"`
	Name       string   `json:"name,omitempty"`
	Type       string   `json:"type,omitempty"`
	Endpoint   string   `json:"endpoint,omitempty"`
	Insecure   bool     `json:"insecure,omitempty"`
	Enabled    bool     `json:"enabled"`
	Warnings   []string `json:"warnings,omitempty"`
}

type RegistryRepositoryListInput struct {
	RegistryID string `json:"registryId,omitempty" yaml:"registryId,omitempty"`
	Repository string `json:"repository,omitempty" yaml:"repository,omitempty"`
	ProjectID  string `json:"projectId,omitempty" yaml:"projectId,omitempty"`
}

type RegistryRepositoryArtifacts struct {
	RegistryID string                    `json:"registryId"`
	Name       string                    `json:"name,omitempty"`
	Repository string                    `json:"repository"`
	Artifacts  []domainartifact.Artifact `json:"artifacts"`
	Warnings   []string                  `json:"warnings,omitempty"`
}

type RegistryStore interface {
	CreateRegistry(ctx context.Context, registry domainartifact.ArtifactRegistry) (domainartifact.ArtifactRegistry, error)
	GetRegistry(ctx context.Context, id string) (domainartifact.ArtifactRegistry, error)
	ListRegistries(ctx context.Context, projectID string) ([]domainartifact.ArtifactRegistry, error)
	UpdateRegistry(ctx context.Context, registry domainartifact.ArtifactRegistry) (domainartifact.ArtifactRegistry, error)
}

type RegistryMemoryStore struct {
	mu         sync.RWMutex
	registries map[string]domainartifact.ArtifactRegistry
}

func NewRegistryMemoryStore() *RegistryMemoryStore {
	return &RegistryMemoryStore{registries: map[string]domainartifact.ArtifactRegistry{}}
}

func (s *RegistryMemoryStore) CreateRegistry(ctx context.Context, registry domainartifact.ArtifactRegistry) (domainartifact.ArtifactRegistry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.registries[registry.ID]; ok {
		return domainartifact.ArtifactRegistry{}, fmt.Errorf("%w: id %q", ErrRegistryAlreadyExists, registry.ID)
	}
	s.registries[registry.ID] = copyRegistry(registry)
	return copyRegistry(registry), nil
}

func (s *RegistryMemoryStore) GetRegistry(ctx context.Context, id string) (domainartifact.ArtifactRegistry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	registry, ok := s.registries[id]
	if !ok {
		return domainartifact.ArtifactRegistry{}, fmt.Errorf("%w: %q", ErrRegistryNotFound, id)
	}
	return copyRegistry(registry), nil
}

func (s *RegistryMemoryStore) ListRegistries(ctx context.Context, projectID string) ([]domainartifact.ArtifactRegistry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domainartifact.ArtifactRegistry, 0, len(s.registries))
	for _, registry := range s.registries {
		if projectID != "" && registry.ProjectID != projectID {
			continue
		}
		out = append(out, copyRegistry(registry))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *RegistryMemoryStore) UpdateRegistry(ctx context.Context, registry domainartifact.ArtifactRegistry) (domainartifact.ArtifactRegistry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.registries[registry.ID]; !ok {
		return domainartifact.ArtifactRegistry{}, fmt.Errorf("%w: %q", ErrRegistryNotFound, registry.ID)
	}
	s.registries[registry.ID] = copyRegistry(registry)
	return copyRegistry(registry), nil
}

type RegistryService struct {
	store           RegistryStore
	now             func() time.Time
	providerFactory RegistryProviderFactory
}

type RegistryProviderFactory func(registry domainartifact.ArtifactRegistry) portartifact.ArtifactProvider

func NewRegistryService(store RegistryStore) *RegistryService {
	return &RegistryService{store: store, now: time.Now}
}

func NewRegistryServiceWithProviderFactory(store RegistryStore, factory RegistryProviderFactory) *RegistryService {
	service := NewRegistryService(store)
	service.providerFactory = factory
	return service
}

func (s *RegistryService) Create(ctx context.Context, input RegistryCreateInput) (domainartifact.ArtifactRegistry, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return domainartifact.ArtifactRegistry{}, fmt.Errorf("%w: registry name is required", ErrRegistryInvalid)
	}
	registryType := defaultRegistryType(input.Type)
	endpoint := strings.TrimSpace(input.Endpoint)
	if endpoint == "" {
		endpoint = strings.TrimSpace(input.URL)
	}
	if err := validateRegistryEndpoint(registryType, endpoint, input.Insecure); err != nil {
		return domainartifact.ArtifactRegistry{}, err
	}
	projectID := strings.TrimSpace(input.ProjectID)
	existing, _ := s.store.ListRegistries(ctx, projectID)
	for _, candidate := range existing {
		if strings.EqualFold(candidate.Name, name) {
			return domainartifact.ArtifactRegistry{}, fmt.Errorf("%w: registry %q", ErrRegistryAlreadyExists, name)
		}
	}
	now := s.now().UTC()
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	registry := domainartifact.ArtifactRegistry{
		ID:            defaultRegistryID(input.ID),
		ProjectID:     projectID,
		Name:          name,
		Type:          registryType,
		URL:           strings.TrimSpace(input.URL),
		Endpoint:      endpoint,
		Insecure:      input.Insecure,
		CredentialRef: strings.TrimSpace(input.CredentialRef),
		Capabilities:  normalizeRegistryCapabilities(input.Capabilities),
		Labels:        copyStringMap(input.Labels),
		Metadata:      copyStringMap(input.Metadata),
		Enabled:       enabled,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if len(registry.Capabilities) == 0 {
		registry.Capabilities = []string{"resolve_digest", "inspect_manifest"}
	}
	return s.store.CreateRegistry(ctx, registry)
}

func (s *RegistryService) Get(ctx context.Context, id string) (domainartifact.ArtifactRegistry, error) {
	return s.store.GetRegistry(ctx, strings.TrimSpace(id))
}

func (s *RegistryService) List(ctx context.Context, projectID string) ([]domainartifact.ArtifactRegistry, error) {
	return s.store.ListRegistries(ctx, strings.TrimSpace(projectID))
}

func (s *RegistryService) Update(ctx context.Context, id string, input RegistryUpdateInput) (domainartifact.ArtifactRegistry, error) {
	registry, err := s.store.GetRegistry(ctx, strings.TrimSpace(id))
	if err != nil {
		return domainartifact.ArtifactRegistry{}, err
	}
	if input.ProjectID != nil {
		registry.ProjectID = strings.TrimSpace(*input.ProjectID)
	}
	if input.Name != nil {
		registry.Name = strings.TrimSpace(*input.Name)
		if registry.Name == "" {
			return domainartifact.ArtifactRegistry{}, fmt.Errorf("%w: registry name is required", ErrRegistryInvalid)
		}
	}
	if input.Type != nil {
		registry.Type = defaultRegistryType(*input.Type)
	}
	if input.URL != nil {
		registry.URL = strings.TrimSpace(*input.URL)
	}
	if input.Endpoint != nil {
		registry.Endpoint = strings.TrimSpace(*input.Endpoint)
	}
	if input.Insecure != nil {
		registry.Insecure = *input.Insecure
	}
	if input.CredentialRef != nil {
		registry.CredentialRef = strings.TrimSpace(*input.CredentialRef)
	}
	if input.Capabilities != nil {
		registry.Capabilities = normalizeRegistryCapabilities(input.Capabilities)
	}
	if input.Labels != nil {
		registry.Labels = copyStringMap(input.Labels)
	}
	if input.Metadata != nil {
		registry.Metadata = copyStringMap(input.Metadata)
	}
	if input.Enabled != nil {
		registry.Enabled = *input.Enabled
	}
	if registry.Endpoint == "" {
		registry.Endpoint = registry.URL
	}
	if err := validateRegistryEndpoint(registry.Type, registry.Endpoint, registry.Insecure); err != nil {
		return domainartifact.ArtifactRegistry{}, err
	}
	registry.UpdatedAt = s.now().UTC()
	return s.store.UpdateRegistry(ctx, registry)
}

func (s *RegistryService) Disable(ctx context.Context, id string) (domainartifact.ArtifactRegistry, error) {
	enabled := false
	return s.Update(ctx, id, RegistryUpdateInput{Enabled: &enabled})
}

func (s *RegistryService) Validate(ctx context.Context, id string) (RegistryValidationResult, error) {
	registry, err := s.store.GetRegistry(ctx, strings.TrimSpace(id))
	if err != nil {
		return RegistryValidationResult{}, err
	}
	result := RegistryValidationResult{
		Valid:      true,
		RegistryID: registry.ID,
		Name:       registry.Name,
		Type:       registry.Type,
		Endpoint:   registry.Endpoint,
		Insecure:   registry.Insecure,
		Enabled:    registry.Enabled,
	}
	if !registry.Enabled {
		result.Valid = false
		result.Warnings = append(result.Warnings, "artifact registry is disabled and cannot be used")
	}
	if registry.Insecure {
		result.Warnings = append(result.Warnings, "insecure OCI registry configuration is for local development only")
	}
	if err := validateRegistryEndpoint(registry.Type, registry.Endpoint, registry.Insecure); err != nil {
		result.Valid = false
		result.Warnings = append(result.Warnings, err.Error())
	}
	return result, nil
}

func (s *RegistryService) ListRepositoryArtifacts(ctx context.Context, input RegistryRepositoryListInput) (RegistryRepositoryArtifacts, error) {
	registryID := strings.TrimSpace(input.RegistryID)
	if registryID == "" {
		return RegistryRepositoryArtifacts{}, fmt.Errorf("%w: registry id is required", ErrRegistryInvalid)
	}
	repository := strings.TrimSpace(input.Repository)
	if repository == "" {
		return RegistryRepositoryArtifacts{}, fmt.Errorf("%w: repository is required", ErrRegistryInvalid)
	}
	registry, err := s.store.GetRegistry(ctx, registryID)
	if err != nil {
		return RegistryRepositoryArtifacts{}, err
	}
	if projectID := strings.TrimSpace(input.ProjectID); projectID != "" && registry.ProjectID != projectID {
		return RegistryRepositoryArtifacts{}, fmt.Errorf("%w: %q", ErrRegistryNotFound, registryID)
	}
	if !registry.Enabled {
		return RegistryRepositoryArtifacts{}, fmt.Errorf("%w: registry %q is disabled", ErrRegistryInvalid, registry.ID)
	}
	if err := validateRegistryEndpoint(registry.Type, registry.Endpoint, registry.Insecure); err != nil {
		return RegistryRepositoryArtifacts{}, err
	}
	if s.providerFactory == nil {
		return RegistryRepositoryArtifacts{}, fmt.Errorf("%w: registry artifact listing provider is not configured", ErrRegistryInvalid)
	}
	provider := s.providerFactory(registry)
	if provider == nil {
		return RegistryRepositoryArtifacts{}, fmt.Errorf("%w: registry artifact listing provider is not configured", ErrRegistryInvalid)
	}
	if !provider.Capabilities().SupportsListing {
		return RegistryRepositoryArtifacts{}, fmt.Errorf("%w: registry provider does not support artifact listing", ErrRegistryInvalid)
	}
	artifacts, err := provider.ListArtifacts(ctx, repository)
	if err != nil {
		return RegistryRepositoryArtifacts{}, err
	}
	result := RegistryRepositoryArtifacts{
		RegistryID: registry.ID,
		Name:       registry.Name,
		Repository: repository,
		Artifacts:  artifacts,
	}
	if registry.Insecure {
		result.Warnings = append(result.Warnings, "insecure OCI registry configuration is for local development only")
	}
	if registry.CredentialRef != "" {
		result.Warnings = append(result.Warnings, "registry CredentialRef was passed as metadata; secret values are not returned")
	}
	return result, nil
}

func validateRegistryEndpoint(registryType string, endpoint string, insecure bool) error {
	if registryType != "oci" {
		return fmt.Errorf("%w: only generic OCI registry configuration is supported", ErrRegistryInvalid)
	}
	if endpoint == "" {
		return fmt.Errorf("%w: registry endpoint is required", ErrRegistryInvalid)
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Host == "" {
		if strings.Contains(endpoint, "://") {
			return fmt.Errorf("%w: registry endpoint must be a valid URL or host[:port]", ErrRegistryInvalid)
		}
		return nil
	}
	if parsed.Scheme == "http" && !insecure {
		return fmt.Errorf("%w: http registry endpoint requires insecure=true", ErrRegistryInvalid)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("%w: registry endpoint scheme must be http or https", ErrRegistryInvalid)
	}
	return nil
}

func defaultRegistryType(input string) string {
	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" {
		return "oci"
	}
	return input
}

func normalizeRegistryCapabilities(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, item := range in {
		item = strings.TrimSpace(strings.ToLower(item))
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

func defaultRegistryID(input string) string {
	input = strings.TrimSpace(input)
	if input != "" {
		return input
	}
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("areg-%d", time.Now().UnixNano())
	}
	return "areg-" + hex.EncodeToString(b[:])
}

func copyRegistry(in domainartifact.ArtifactRegistry) domainartifact.ArtifactRegistry {
	in.Capabilities = append([]string(nil), in.Capabilities...)
	in.Labels = copyStringMap(in.Labels)
	in.Metadata = copyStringMap(in.Metadata)
	return in
}

func copyStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
