package artifact

import (
	domainartifact "github.com/sevoniva/nivora/internal/domain/artifact"
	"github.com/sevoniva/nivora/internal/domain/audit"
	"github.com/sevoniva/nivora/internal/domain/event"
	"github.com/sevoniva/nivora/internal/domain/release"
)

type ReleaseDefinition struct {
	APIVersion string          `json:"apiVersion" yaml:"apiVersion"`
	Kind       string          `json:"kind" yaml:"kind"`
	Metadata   ReleaseMetadata `json:"metadata" yaml:"metadata"`
	Spec       ReleaseSpec     `json:"spec" yaml:"spec"`
}

type ReleaseMetadata struct {
	Name string `json:"name" yaml:"name"`
}

type ReleaseSpec struct {
	Version             string                `json:"version" yaml:"version"`
	Application         string                `json:"application,omitempty" yaml:"application,omitempty"`
	Environment         string                `json:"environment,omitempty" yaml:"environment,omitempty"`
	SourcePipelineRunID string                `json:"sourcePipelineRunId,omitempty" yaml:"sourcePipelineRunId,omitempty"`
	Commit              string                `json:"commit,omitempty" yaml:"commit,omitempty"`
	ResolveDigest       bool                  `json:"resolveDigest,omitempty" yaml:"resolveDigest,omitempty"`
	RequireDigest       bool                  `json:"requireDigest,omitempty" yaml:"requireDigest,omitempty"`
	BlockMutable        bool                  `json:"blockMutable,omitempty" yaml:"blockMutable,omitempty"`
	Artifacts           []ReleaseArtifactSpec `json:"artifacts" yaml:"artifacts"`
}

type ReleaseArtifactSpec struct {
	Name          string            `json:"name" yaml:"name"`
	Type          string            `json:"type" yaml:"type"`
	Role          string            `json:"role,omitempty" yaml:"role,omitempty"`
	Required      bool              `json:"required" yaml:"required"`
	Reference     string            `json:"reference" yaml:"reference"`
	ResolveDigest *bool             `json:"resolveDigest,omitempty" yaml:"resolveDigest,omitempty"`
	RequireDigest *bool             `json:"requireDigest,omitempty" yaml:"requireDigest,omitempty"`
	BlockMutable  *bool             `json:"blockMutable,omitempty" yaml:"blockMutable,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

type CreateReleaseInput struct {
	Definition ReleaseDefinition
	ActorID    string
	ProjectID  string
}

type TrackArtifactInput struct {
	ID        string            `json:"id,omitempty"`
	Name      string            `json:"name,omitempty"`
	Type      string            `json:"type,omitempty"`
	Reference string            `json:"reference"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

type ListArtifactsInput struct {
	Type          string
	Name          string
	Registry      string
	Repository    string
	Digest        string
	Reference     string
	ProjectID     string
	EnvironmentID string
}

type ListReleasesInput struct {
	ProjectID     string
	EnvironmentID string
	ApplicationID string
	Status        string
}

type ArtifactReleaseBinding struct {
	Release release.Release         `json:"release"`
	Binding release.ReleaseArtifact `json:"binding"`
}

type ReleaseRecord struct {
	Release     release.Release             `json:"release"`
	Artifacts   []domainartifact.Artifact   `json:"artifacts"`
	Bindings    []release.ReleaseArtifact   `json:"bindings"`
	Inspections []domainartifact.Inspection `json:"inspections,omitempty"`
	Resolutions []domainartifact.Resolution `json:"resolutions,omitempty"`
	Warnings    []domainartifact.Warning    `json:"warnings,omitempty"`
	Events      []event.Event               `json:"events,omitempty"`
	Audits      []audit.AuditLog            `json:"audits,omitempty"`
}
