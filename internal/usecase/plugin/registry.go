package plugin

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"

	domainplugin "github.com/sevoniva/nivora/internal/domain/plugin"
)

var ErrPluginNotFound = errors.New("plugin not found")

const NivoraPluginHostVersion = "0.1.0-alpha.1"

func NewRegistry(plugins []domainplugin.Manifest) *Registry {
	registry := &Registry{plugins: map[string]domainplugin.Manifest{}}
	for _, manifest := range plugins {
		if manifest.Name == "" {
			continue
		}
		name := strings.ToLower(manifest.Name)
		manifest.Name = name
		registry.plugins[name] = cloneManifest(manifest)
		registry.order = append(registry.order, name)
	}
	sort.Strings(registry.order)
	return registry
}

func NewDefaultRegistry() *Registry {
	now := time.Date(2026, 5, 19, 0, 0, 0, 0, time.UTC)
	return NewRegistry([]domainplugin.Manifest{
		builtin("scm-generic", domainplugin.TypeSCM, []domainplugin.Capability{
			capability("scm.placeholder", "SCM provider adapter skeleton for future Git providers"),
		}, now, boundaryMetadata("foundation", "skeleton", "metadata-only", "credential_ref_only", "none")),
		builtin("artifact-oci", domainplugin.TypeArtifact, []domainplugin.Capability{
			capability("artifact.inspect", "Parse and inspect OCI artifact references"),
			capability("artifact.resolve_digest", "Resolve OCI image digests when configured"),
		}, now, boundaryMetadata("partial", "foundation", "read-only", "credential_ref_only", "optional")),
		builtin("cloud-aws", domainplugin.TypeCloud, []domainplugin.Capability{
			capability("cloud.inventory", "AWS inventory provider skeleton backed by deterministic local behavior"),
		}, now, boundaryMetadata("foundation", "skeleton", "metadata-only", "credential_ref_only", "none")),
		builtin("cloud-aliyun", domainplugin.TypeCloud, []domainplugin.Capability{
			capability("cloud.inventory", "Aliyun inventory provider skeleton backed by deterministic local behavior"),
		}, now, boundaryMetadata("foundation", "skeleton", "metadata-only", "credential_ref_only", "none")),
		builtin("cloud-tencent", domainplugin.TypeCloud, []domainplugin.Capability{
			capability("cloud.inventory", "Tencent Cloud inventory provider skeleton backed by deterministic local behavior"),
		}, now, boundaryMetadata("foundation", "skeleton", "metadata-only", "credential_ref_only", "none")),
		builtin("executor-shell", domainplugin.TypeExecutor, []domainplugin.Capability{
			capability("executor.shell", "Execute safe local shell steps for Phase 1 PipelineRuns"),
		}, now, boundaryMetadata("partial", "foundation", "development-only", "none", "none")),
		builtin("executor-yaml-apply", domainplugin.TypeExecutor, []domainplugin.Capability{
			capability("executor.kubernetes_yaml.plan", "Plan and validate static Kubernetes YAML manifests"),
			capability("executor.kubernetes_yaml.noop_apply", "Run guarded noop apply flow for tests and local development"),
		}, now, boundaryMetadata("experimental", "foundation", "guarded-action", "credential_ref_only", "optional")),
		builtin("executor-argocd", domainplugin.TypeGitOps, []domainplugin.Capability{
			capability("gitops.plan", "Build GitOps change plans"),
			capability("argocd.status", "Read deterministic Argo CD application status through noop provider"),
			capability("argocd.guarded_sync", "Model guarded sync requests without production automation"),
		}, now, boundaryMetadata("experimental", "noop", "guarded-action", "credential_ref_only", "optional")),
		builtin("secret-builtin", domainplugin.TypeSecret, []domainplugin.Capability{
			capability("secret.store_development", "Store local development secrets behind SecretRef metadata"),
		}, now, boundaryMetadata("partial", "foundation", "development-only", "secret_ref_only", "none")),
		builtin("notification-noop", domainplugin.TypeNotification, []domainplugin.Capability{
			capability("notification.noop", "Record notification requests without external delivery"),
		}, now, boundaryMetadata("foundation", "noop", "noop", "none", "none")),
		builtin("policy-builtin", domainplugin.TypePolicy, []domainplugin.Capability{
			capability("policy.evaluate", "Evaluate minimal built-in delivery gate rules"),
		}, now, boundaryMetadata("foundation", "foundation", "read-only", "none", "none")),
		builtin("scanner-noop", domainplugin.TypeScanner, []domainplugin.Capability{
			capability("security.scan_noop", "Return deterministic no-op security scan results for tests"),
		}, now, boundaryMetadata("foundation", "noop", "noop", "none", "none")),
	})
}

func (r *Registry) List(ctx context.Context) ([]domainplugin.Manifest, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	items := make([]domainplugin.Manifest, 0, len(r.order))
	for _, name := range r.order {
		items = append(items, cloneManifest(r.plugins[name]))
	}
	return items, nil
}

func (r *Registry) Get(ctx context.Context, name string) (domainplugin.Manifest, error) {
	if err := ctx.Err(); err != nil {
		return domainplugin.Manifest{}, err
	}
	manifest, ok := r.plugins[strings.ToLower(name)]
	if !ok {
		return domainplugin.Manifest{}, ErrPluginNotFound
	}
	return cloneManifest(manifest), nil
}

func (r *Registry) Capabilities(ctx context.Context, name string) ([]domainplugin.Capability, error) {
	manifest, err := r.Get(ctx, name)
	if err != nil {
		return nil, err
	}
	return append([]domainplugin.Capability(nil), manifest.Capabilities...), nil
}

func (r *Registry) Validate(ctx context.Context, manifest domainplugin.Manifest) (ValidationResult, error) {
	if err := ctx.Err(); err != nil {
		return ValidationResult{}, err
	}
	return ValidateManifest(manifest), nil
}

func ValidateManifest(manifest domainplugin.Manifest) ValidationResult {
	result := ValidationResult{Valid: true, Plugin: manifest.Name}
	if manifest.APIVersion == "" {
		result.Warnings = append(result.Warnings, "apiVersion is recommended and defaults to "+domainplugin.ManifestAPIVersion)
	} else if manifest.APIVersion != domainplugin.ManifestAPIVersion {
		result.Errors = append(result.Errors, "unsupported apiVersion "+manifest.APIVersion)
	}
	if manifest.Name == "" {
		result.Errors = append(result.Errors, "name is required")
	}
	if !validType(manifest.Type) {
		result.Errors = append(result.Errors, "unsupported plugin type "+string(manifest.Type))
	}
	if manifest.Version == "" {
		result.Errors = append(result.Errors, "version is required")
	}
	if !validProtocol(manifest.Protocol) {
		result.Errors = append(result.Errors, "unsupported protocol "+manifest.Protocol)
	}
	if (manifest.Protocol == string(domainplugin.ProtocolHTTP) || manifest.Protocol == string(domainplugin.ProtocolGRPC)) && manifest.Endpoint == "" {
		result.Errors = append(result.Errors, "external plugin endpoint is required for "+manifest.Protocol+" protocol")
	}
	if manifest.Compatibility.PluginAPIVersion == "" {
		result.Warnings = append(result.Warnings, "compatibility.pluginApiVersion is recommended and defaults to "+domainplugin.PluginAPIVersion)
	} else if manifest.Compatibility.PluginAPIVersion != domainplugin.PluginAPIVersion {
		result.Errors = append(result.Errors, "unsupported plugin API version "+manifest.Compatibility.PluginAPIVersion)
	}
	if manifest.Compatibility.NivoraMinVersion != "" && versionAfter(manifest.Compatibility.NivoraMinVersion, NivoraPluginHostVersion) {
		result.Errors = append(result.Errors, "plugin requires Nivora >= "+manifest.Compatibility.NivoraMinVersion)
	}
	if manifest.Compatibility.NivoraMaxVersion != "" && versionAfter(NivoraPluginHostVersion, manifest.Compatibility.NivoraMaxVersion) {
		result.Errors = append(result.Errors, "plugin supports Nivora <= "+manifest.Compatibility.NivoraMaxVersion)
	}
	if len(manifest.Capabilities) == 0 {
		result.Errors = append(result.Errors, "at least one capability is required")
	}
	for i, capability := range manifest.Capabilities {
		if capability.Name == "" {
			result.Errors = append(result.Errors, "capabilities["+itoa(i)+"].name is required")
		}
	}
	if len(result.Errors) > 0 {
		result.Valid = false
	}
	return result
}

func (r *Registry) MatchCapability(ctx context.Context, capabilityName string) ([]CapabilityMatch, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	matches := []CapabilityMatch{}
	for _, name := range r.order {
		manifest := r.plugins[name]
		for _, capability := range manifest.Capabilities {
			if capability.Name == capabilityName {
				matches = append(matches, CapabilityMatch{Plugin: cloneManifest(manifest), Capability: capability})
			}
		}
	}
	return matches, nil
}

func builtin(name string, pluginType domainplugin.Type, capabilities []domainplugin.Capability, now time.Time, metadata ...map[string]string) domainplugin.Manifest {
	mergedMetadata := map[string]string{
		"phase":           "7.4",
		"safe":            "true",
		"defaultMutation": "false",
	}
	for _, item := range metadata {
		for key, value := range item {
			mergedMetadata[key] = value
		}
	}
	return domainplugin.Manifest{
		APIVersion:   domainplugin.ManifestAPIVersion,
		Name:         name,
		Type:         pluginType,
		Version:      NivoraPluginHostVersion,
		Protocol:     string(domainplugin.ProtocolBuiltIn),
		Capabilities: capabilities,
		Compatibility: domainplugin.Compatibility{
			PluginAPIVersion: domainplugin.PluginAPIVersion,
			NivoraMinVersion: "0.1.0-alpha.1",
			Protocols:        []string{string(domainplugin.ProtocolBuiltIn)},
		},
		Lifecycle: domainplugin.Lifecycle{
			Health:         true,
			Capabilities:   true,
			ValidateConfig: true,
			Execute:        false,
		},
		Status:    domainplugin.StatusBuiltIn,
		Metadata:  mergedMetadata,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func boundaryMetadata(maturity string, adapterKind string, boundary string, credentialMode string, networkAccess string) map[string]string {
	return map[string]string{
		"maturity":       maturity,
		"adapterKind":    adapterKind,
		"boundary":       boundary,
		"credentialMode": credentialMode,
		"networkAccess":  networkAccess,
	}
}

func capability(name string, description string) domainplugin.Capability {
	return domainplugin.Capability{Name: name, Description: description}
}

func validType(pluginType domainplugin.Type) bool {
	switch pluginType {
	case domainplugin.TypeSCM, domainplugin.TypeArtifact, domainplugin.TypeCloud, domainplugin.TypeExecutor, domainplugin.TypeSecret, domainplugin.TypeNotification, domainplugin.TypePolicy, domainplugin.TypeScanner, domainplugin.TypeGitOps:
		return true
	default:
		return false
	}
}

func validProtocol(protocol string) bool {
	switch protocol {
	case string(domainplugin.ProtocolBuiltIn), string(domainplugin.ProtocolHTTP), string(domainplugin.ProtocolGRPC):
		return true
	default:
		return false
	}
}

func versionAfter(left string, right string) bool {
	leftParts := versionParts(left)
	rightParts := versionParts(right)
	for i := 0; i < len(leftParts) || i < len(rightParts); i++ {
		var leftValue, rightValue int
		if i < len(leftParts) {
			leftValue = leftParts[i]
		}
		if i < len(rightParts) {
			rightValue = rightParts[i]
		}
		if leftValue != rightValue {
			return leftValue > rightValue
		}
	}
	return false
}

func versionParts(version string) []int {
	version = strings.TrimPrefix(version, "v")
	fields := strings.FieldsFunc(version, func(r rune) bool {
		return r < '0' || r > '9'
	})
	parts := make([]int, 0, len(fields))
	for _, field := range fields {
		if field == "" {
			continue
		}
		value, err := strconv.Atoi(field)
		if err != nil {
			continue
		}
		parts = append(parts, value)
	}
	return parts
}

func itoa(value int) string {
	const digits = "0123456789"
	if value == 0 {
		return "0"
	}
	var out [20]byte
	i := len(out)
	for value > 0 {
		i--
		out[i] = digits[value%10]
		value /= 10
	}
	return string(out[i:])
}

func cloneManifest(manifest domainplugin.Manifest) domainplugin.Manifest {
	manifest.Capabilities = append([]domainplugin.Capability(nil), manifest.Capabilities...)
	for i, capability := range manifest.Capabilities {
		if capability.Inputs != nil {
			manifest.Capabilities[i].Inputs = append([]string(nil), capability.Inputs...)
		}
		if capability.Outputs != nil {
			manifest.Capabilities[i].Outputs = append([]string(nil), capability.Outputs...)
		}
		if capability.Metadata != nil {
			metadata := make(map[string]string, len(capability.Metadata))
			for key, value := range capability.Metadata {
				metadata[key] = value
			}
			manifest.Capabilities[i].Metadata = metadata
		}
	}
	manifest.Compatibility.Protocols = append([]string(nil), manifest.Compatibility.Protocols...)
	if manifest.Metadata != nil {
		metadata := make(map[string]string, len(manifest.Metadata))
		for key, value := range manifest.Metadata {
			metadata[key] = value
		}
		manifest.Metadata = metadata
	}
	return manifest
}
