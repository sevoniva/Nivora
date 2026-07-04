package pipeline

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	domainpipeline "github.com/sevoniva/nivora/internal/domain/pipeline"
)

var (
	ErrPipelineDefinitionNotFound      = errors.New("pipeline definition not found")
	ErrPipelineDefinitionAlreadyExists = errors.New("pipeline definition already exists")
)

type DefinitionRecord struct {
	Pipeline   domainpipeline.Pipeline        `json:"pipeline"`
	Version    domainpipeline.PipelineVersion `json:"version"`
	Definition Definition                     `json:"definition"`
}

type DefinitionVersionRecord struct {
	Version    domainpipeline.PipelineVersion `json:"version"`
	Definition Definition                     `json:"definition"`
}

type DefinitionCreateInput struct {
	ID          string            `json:"id,omitempty"`
	ProjectID   string            `json:"projectId,omitempty"`
	Description string            `json:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Definition  Definition        `json:"definition"`
	Enabled     *bool             `json:"enabled,omitempty"`
}

type DefinitionUpdateInput struct {
	Description *string           `json:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Definition  *Definition       `json:"definition,omitempty"`
	Enabled     *bool             `json:"enabled,omitempty"`
}

type DefinitionCatalogStore interface {
	CreateDefinition(ctx context.Context, record DefinitionRecord) (DefinitionRecord, error)
	GetDefinition(ctx context.Context, id string) (DefinitionRecord, error)
	ListDefinitions(ctx context.Context, projectID string) ([]DefinitionRecord, error)
	ListDefinitionVersions(ctx context.Context, id string) ([]domainpipeline.PipelineVersion, error)
	GetDefinitionVersion(ctx context.Context, id string, version int) (DefinitionVersionRecord, error)
	UpdateDefinition(ctx context.Context, record DefinitionRecord) (DefinitionRecord, error)
}

type DefinitionCatalog struct {
	store DefinitionCatalogStore
	now   func() time.Time
}

func NewDefinitionCatalog(store DefinitionCatalogStore) *DefinitionCatalog {
	return &DefinitionCatalog{store: store, now: time.Now}
}

func (c *DefinitionCatalog) Create(ctx context.Context, input DefinitionCreateInput) (DefinitionRecord, error) {
	if err := input.Definition.Validate(); err != nil {
		return DefinitionRecord{}, err
	}
	name := strings.TrimSpace(input.Definition.Metadata.Name)
	projectID := strings.TrimSpace(input.ProjectID)
	existing, _ := c.store.ListDefinitions(ctx, projectID)
	for _, record := range existing {
		if strings.EqualFold(record.Pipeline.Name, name) {
			return DefinitionRecord{}, fmt.Errorf("%w: pipeline %q", ErrPipelineDefinitionAlreadyExists, name)
		}
	}
	now := c.now().UTC()
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	pipeline := domainpipeline.Pipeline{
		ID:          defaultDefinitionID(input.ID),
		ProjectID:   projectID,
		Name:        name,
		Description: strings.TrimSpace(input.Description),
		Labels:      cloneMap(input.Labels),
		Metadata:    cloneMap(input.Metadata),
		Enabled:     enabled,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	version := domainpipeline.PipelineVersion{
		ID:             newID("pver"),
		PipelineID:     pipeline.ID,
		Version:        1,
		DefinitionHash: definitionHash(input.Definition),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	return c.store.CreateDefinition(ctx, DefinitionRecord{Pipeline: pipeline, Version: version, Definition: input.Definition})
}

func (c *DefinitionCatalog) Get(ctx context.Context, id string) (DefinitionRecord, error) {
	return c.store.GetDefinition(ctx, strings.TrimSpace(id))
}

func (c *DefinitionCatalog) List(ctx context.Context, projectID string) ([]DefinitionRecord, error) {
	return c.store.ListDefinitions(ctx, strings.TrimSpace(projectID))
}

func (c *DefinitionCatalog) Versions(ctx context.Context, id string) ([]domainpipeline.PipelineVersion, error) {
	record, err := c.store.GetDefinition(ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	versions, err := c.store.ListDefinitionVersions(ctx, record.Pipeline.ID)
	if err != nil {
		return nil, err
	}
	if len(versions) == 0 {
		return []domainpipeline.PipelineVersion{record.Version}, nil
	}
	return versions, nil
}

func (c *DefinitionCatalog) Version(ctx context.Context, id string, version int) (DefinitionVersionRecord, error) {
	if version <= 0 {
		return DefinitionVersionRecord{}, fmt.Errorf("pipeline definition version must be greater than zero")
	}
	record, err := c.store.GetDefinition(ctx, strings.TrimSpace(id))
	if err != nil {
		return DefinitionVersionRecord{}, err
	}
	if record.Version.Version == version {
		return DefinitionVersionRecord{Version: record.Version, Definition: record.Definition}, nil
	}
	return c.store.GetDefinitionVersion(ctx, record.Pipeline.ID, version)
}

func (c *DefinitionCatalog) Update(ctx context.Context, id string, input DefinitionUpdateInput) (DefinitionRecord, error) {
	record, err := c.store.GetDefinition(ctx, strings.TrimSpace(id))
	if err != nil {
		return DefinitionRecord{}, err
	}
	now := c.now().UTC()
	if input.Description != nil {
		record.Pipeline.Description = strings.TrimSpace(*input.Description)
	}
	if input.Labels != nil {
		record.Pipeline.Labels = cloneMap(input.Labels)
	}
	if input.Metadata != nil {
		record.Pipeline.Metadata = cloneMap(input.Metadata)
	}
	if input.Enabled != nil {
		record.Pipeline.Enabled = *input.Enabled
	}
	if input.Definition != nil {
		if err := input.Definition.Validate(); err != nil {
			return DefinitionRecord{}, err
		}
		record.Definition = *input.Definition
		record.Pipeline.Name = strings.TrimSpace(input.Definition.Metadata.Name)
		record.Version = domainpipeline.PipelineVersion{
			ID:             newID("pver"),
			PipelineID:     record.Pipeline.ID,
			Version:        record.Version.Version + 1,
			DefinitionHash: definitionHash(*input.Definition),
			CreatedAt:      now,
			UpdatedAt:      now,
		}
	}
	record.Pipeline.UpdatedAt = now
	return c.store.UpdateDefinition(ctx, record)
}

func (c *DefinitionCatalog) Disable(ctx context.Context, id string) (DefinitionRecord, error) {
	disabled := false
	return c.Update(ctx, id, DefinitionUpdateInput{Enabled: &disabled})
}

type DefinitionMemoryStore struct {
	mu       sync.RWMutex
	records  map[string]DefinitionRecord
	versions map[string][]DefinitionVersionRecord
}

func NewDefinitionMemoryStore() *DefinitionMemoryStore {
	return &DefinitionMemoryStore{
		records:  map[string]DefinitionRecord{},
		versions: map[string][]DefinitionVersionRecord{},
	}
}

func (s *DefinitionMemoryStore) CreateDefinition(ctx context.Context, record DefinitionRecord) (DefinitionRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.records[record.Pipeline.ID]; ok {
		return DefinitionRecord{}, fmt.Errorf("%w: pipeline id %q", ErrPipelineDefinitionAlreadyExists, record.Pipeline.ID)
	}
	s.records[record.Pipeline.ID] = cloneDefinitionRecord(record)
	s.versions[record.Pipeline.ID] = appendVersion(s.versions[record.Pipeline.ID], DefinitionVersionRecord{Version: record.Version, Definition: record.Definition})
	return cloneDefinitionRecord(record), nil
}

func (s *DefinitionMemoryStore) GetDefinition(ctx context.Context, id string) (DefinitionRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, ok := s.records[id]
	if !ok {
		return DefinitionRecord{}, fmt.Errorf("%w: pipeline %q", ErrPipelineDefinitionNotFound, id)
	}
	return cloneDefinitionRecord(record), nil
}

func (s *DefinitionMemoryStore) ListDefinitions(ctx context.Context, projectID string) ([]DefinitionRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]DefinitionRecord, 0, len(s.records))
	for _, record := range s.records {
		if projectID == "" || record.Pipeline.ProjectID == projectID {
			out = append(out, cloneDefinitionRecord(record))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Pipeline.Name < out[j].Pipeline.Name })
	return out, nil
}

func (s *DefinitionMemoryStore) ListDefinitionVersions(ctx context.Context, id string) ([]domainpipeline.PipelineVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.records[id]; !ok {
		return nil, fmt.Errorf("%w: pipeline %q", ErrPipelineDefinitionNotFound, id)
	}
	records := append([]DefinitionVersionRecord(nil), s.versions[id]...)
	sort.Slice(records, func(i, j int) bool { return records[i].Version.Version < records[j].Version.Version })
	versions := make([]domainpipeline.PipelineVersion, 0, len(records))
	for _, record := range records {
		versions = append(versions, record.Version)
	}
	return versions, nil
}

func (s *DefinitionMemoryStore) GetDefinitionVersion(ctx context.Context, id string, version int) (DefinitionVersionRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.records[id]; !ok {
		return DefinitionVersionRecord{}, fmt.Errorf("%w: pipeline %q", ErrPipelineDefinitionNotFound, id)
	}
	for _, record := range s.versions[id] {
		if record.Version.Version == version {
			return cloneDefinitionVersionRecord(record), nil
		}
	}
	return DefinitionVersionRecord{}, fmt.Errorf("%w: pipeline %q version %d", ErrPipelineDefinitionNotFound, id, version)
}

func (s *DefinitionMemoryStore) UpdateDefinition(ctx context.Context, record DefinitionRecord) (DefinitionRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.records[record.Pipeline.ID]; !ok {
		return DefinitionRecord{}, fmt.Errorf("%w: pipeline %q", ErrPipelineDefinitionNotFound, record.Pipeline.ID)
	}
	s.records[record.Pipeline.ID] = cloneDefinitionRecord(record)
	s.versions[record.Pipeline.ID] = appendVersion(s.versions[record.Pipeline.ID], DefinitionVersionRecord{Version: record.Version, Definition: record.Definition})
	return cloneDefinitionRecord(record), nil
}

func appendVersion(versions []DefinitionVersionRecord, version DefinitionVersionRecord) []DefinitionVersionRecord {
	version = cloneDefinitionVersionRecord(version)
	for i, existing := range versions {
		if existing.Version.ID == version.Version.ID || existing.Version.Version == version.Version.Version {
			versions[i] = version
			return versions
		}
	}
	return append(versions, version)
}

func defaultDefinitionID(id string) string {
	if strings.TrimSpace(id) != "" {
		return strings.TrimSpace(id)
	}
	return newID("pipe")
}

func definitionHash(def Definition) string {
	body, err := json.Marshal(def)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(body)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func cloneDefinitionRecord(record DefinitionRecord) DefinitionRecord {
	record.Pipeline.Labels = cloneMap(record.Pipeline.Labels)
	record.Pipeline.Metadata = cloneMap(record.Pipeline.Metadata)
	record.Definition.Spec.Stages = cloneSpecStages(record.Definition.Spec.Stages)
	return record
}

func cloneDefinitionVersionRecord(record DefinitionVersionRecord) DefinitionVersionRecord {
	record.Definition.Spec.Stages = cloneSpecStages(record.Definition.Spec.Stages)
	return record
}
