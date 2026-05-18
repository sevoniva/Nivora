package artifact

import (
	"errors"
	"fmt"
	"path"
	"strings"
	"time"
)

type ArtifactType string

const (
	ArtifactTypeImage   ArtifactType = "image"
	ArtifactTypeJar     ArtifactType = "jar"
	ArtifactTypeBinary  ArtifactType = "binary"
	ArtifactTypeChart   ArtifactType = "chart"
	ArtifactTypeYAML    ArtifactType = "yaml"
	ArtifactTypeTar     ArtifactType = "tar"
	ArtifactTypeNPM     ArtifactType = "npm"
	ArtifactTypeMaven   ArtifactType = "maven"
	ArtifactTypeUnknown ArtifactType = "unknown"
)

type ArtifactRegistry struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"projectId,omitempty"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	URL       string    `json:"url"`
	Endpoint  string    `json:"endpoint,omitempty"`
	Insecure  bool      `json:"insecure,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Artifact struct {
	ID         string            `json:"id"`
	Type       ArtifactType      `json:"type"`
	Name       string            `json:"name"`
	Version    string            `json:"version,omitempty"`
	Reference  string            `json:"reference"`
	Digest     string            `json:"digest,omitempty"`
	Registry   string            `json:"registry,omitempty"`
	Repository string            `json:"repository,omitempty"`
	MediaType  string            `json:"mediaType,omitempty"`
	SizeBytes  int64             `json:"sizeBytes,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	CreatedAt  time.Time         `json:"createdAt"`
}

type Reference struct {
	Type           ArtifactType `json:"type"`
	Original       string       `json:"original"`
	Normalized     string       `json:"normalized"`
	Registry       string       `json:"registry,omitempty"`
	Repository     string       `json:"repository,omitempty"`
	Name           string       `json:"name,omitempty"`
	Version        string       `json:"version,omitempty"`
	Tag            string       `json:"tag,omitempty"`
	Digest         string       `json:"digest,omitempty"`
	Scheme         string       `json:"scheme,omitempty"`
	Immutable      bool         `json:"immutable"`
	IsDigestPinned bool         `json:"isDigestPinned"`
	IsLatest       bool         `json:"isLatest"`
}

type Warning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Inspection struct {
	Reference Reference `json:"reference"`
	Warnings  []Warning `json:"warnings,omitempty"`
}

type Resolution struct {
	Reference                Reference `json:"reference"`
	Digest                   string    `json:"digest,omitempty"`
	DigestQualifiedReference string    `json:"digestQualifiedReference,omitempty"`
	MediaType                string    `json:"mediaType,omitempty"`
	Resolved                 bool      `json:"resolved"`
	ResolvedAt               time.Time `json:"resolvedAt,omitempty"`
	Warnings                 []Warning `json:"warnings,omitempty"`
}

var ErrInvalidReference = errors.New("invalid artifact reference")

func InspectReference(reference string, artifactType ArtifactType) (Inspection, error) {
	parsed, err := ParseReference(reference, artifactType)
	if err != nil {
		return Inspection{}, err
	}
	return Inspection{Reference: parsed, Warnings: ImmutabilityWarnings(parsed)}, nil
}

func ParseReference(reference string, artifactType ArtifactType) (Reference, error) {
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return Reference{}, fmt.Errorf("%w: reference is required", ErrInvalidReference)
	}
	if artifactType == "" || artifactType == ArtifactTypeUnknown {
		artifactType = inferType(reference)
	}
	if strings.Contains(reference, "://") && artifactType != ArtifactTypeImage {
		return parseGenericReference(reference, artifactType), nil
	}
	return parseOCIReference(reference, artifactType)
}

func ImmutabilityWarnings(reference Reference) []Warning {
	var warnings []Warning
	if reference.Digest != "" {
		return warnings
	}
	if reference.Tag == "latest" {
		warnings = append(warnings, Warning{Code: "mutable_latest_tag", Message: "latest is mutable; prefer a digest reference"})
	}
	if reference.Tag == "" && reference.Version == "" {
		warnings = append(warnings, Warning{Code: "missing_version_or_digest", Message: "artifact reference has no tag, version, or digest"})
	}
	if reference.Tag != "" && reference.Digest == "" {
		warnings = append(warnings, Warning{Code: "tag_without_digest", Message: "tag references can move; prefer sha256 digest binding for releases"})
	}
	return warnings
}

func parseOCIReference(reference string, artifactType ArtifactType) (Reference, error) {
	namePart := reference
	var digest string
	if before, after, ok := strings.Cut(reference, "@"); ok {
		namePart = before
		digest = after
		if !strings.HasPrefix(digest, "sha256:") || strings.TrimPrefix(digest, "sha256:") == "" {
			return Reference{}, fmt.Errorf("%w: unsupported digest %q", ErrInvalidReference, digest)
		}
	}
	lastSlash := strings.LastIndex(namePart, "/")
	lastColon := strings.LastIndex(namePart, ":")
	tag := ""
	repositoryPart := namePart
	if lastColon > lastSlash {
		tag = namePart[lastColon+1:]
		repositoryPart = namePart[:lastColon]
	}
	if repositoryPart == "" {
		return Reference{}, fmt.Errorf("%w: repository is required", ErrInvalidReference)
	}
	parts := strings.Split(repositoryPart, "/")
	registry := ""
	repository := repositoryPart
	if len(parts) > 1 && looksLikeRegistry(parts[0]) {
		registry = parts[0]
		repository = strings.Join(parts[1:], "/")
	}
	normalized := repositoryPart
	if tag != "" {
		normalized += ":" + tag
	}
	if digest != "" {
		normalized += "@" + digest
	}
	version := tag
	if digest != "" {
		version = digest
	}
	return Reference{
		Type:           artifactType,
		Original:       reference,
		Normalized:     normalized,
		Registry:       registry,
		Repository:     repository,
		Name:           path.Base(repository),
		Version:        version,
		Tag:            tag,
		Digest:         digest,
		Immutable:      digest != "",
		IsDigestPinned: digest != "",
		IsLatest:       tag == "latest",
	}, nil
}

func parseGenericReference(reference string, artifactType ArtifactType) Reference {
	scheme, rest, _ := strings.Cut(reference, "://")
	name := path.Base(rest)
	return Reference{
		Type:           artifactType,
		Original:       reference,
		Normalized:     reference,
		Name:           name,
		Version:        name,
		Scheme:         scheme,
		Immutable:      !strings.Contains(name, "latest"),
		IsDigestPinned: false,
		IsLatest:       strings.Contains(name, "latest"),
	}
}

func DigestQualifiedReference(ref Reference, digest string) string {
	if digest == "" {
		return ref.Normalized
	}
	base := ref.Normalized
	if before, _, ok := strings.Cut(base, "@"); ok {
		base = before
	}
	return base + "@" + digest
}

func inferType(reference string) ArtifactType {
	switch {
	case strings.HasPrefix(reference, "s3://"):
		return ArtifactTypeTar
	case strings.HasPrefix(reference, "file://"):
		return ArtifactTypeUnknown
	case strings.HasSuffix(reference, ".jar"):
		return ArtifactTypeJar
	case strings.HasSuffix(reference, ".tgz"), strings.HasSuffix(reference, ".tar.gz"):
		return ArtifactTypeTar
	case strings.HasSuffix(reference, ".yaml"), strings.HasSuffix(reference, ".yml"):
		return ArtifactTypeYAML
	default:
		return ArtifactTypeImage
	}
}

func looksLikeRegistry(component string) bool {
	return strings.Contains(component, ".") || strings.Contains(component, ":") || component == "localhost"
}
