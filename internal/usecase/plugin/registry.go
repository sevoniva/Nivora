package plugin

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	domainplugin "github.com/sevoniva/nivora/internal/domain/plugin"
)

var ErrPluginNotFound = errors.New("plugin not found")

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
		}, now),
		builtin("artifact-oci", domainplugin.TypeArtifact, []domainplugin.Capability{
			capability("artifact.inspect", "Parse and inspect OCI artifact references"),
			capability("artifact.resolve_digest", "Resolve OCI image digests when configured"),
		}, now),
		builtin("cloud-aws", domainplugin.TypeCloud, []domainplugin.Capability{
			capability("cloud.inventory", "AWS inventory provider skeleton backed by deterministic local behavior"),
		}, now),
		builtin("cloud-aliyun", domainplugin.TypeCloud, []domainplugin.Capability{
			capability("cloud.inventory", "Aliyun inventory provider skeleton backed by deterministic local behavior"),
		}, now),
		builtin("cloud-tencent", domainplugin.TypeCloud, []domainplugin.Capability{
			capability("cloud.inventory", "Tencent Cloud inventory provider skeleton backed by deterministic local behavior"),
		}, now),
		builtin("executor-shell", domainplugin.TypeExecutor, []domainplugin.Capability{
			capability("executor.shell", "Execute safe local shell steps for Phase 1 PipelineRuns"),
		}, now),
		builtin("executor-yaml-apply", domainplugin.TypeExecutor, []domainplugin.Capability{
			capability("executor.kubernetes_yaml.plan", "Plan and validate static Kubernetes YAML manifests"),
			capability("executor.kubernetes_yaml.noop_apply", "Run guarded noop apply flow for tests and local development"),
		}, now),
		builtin("executor-argocd", domainplugin.TypeGitOps, []domainplugin.Capability{
			capability("gitops.plan", "Build GitOps change plans"),
			capability("argocd.status", "Read deterministic Argo CD application status through noop provider"),
			capability("argocd.guarded_sync", "Model guarded sync requests without production automation"),
		}, now),
		builtin("secret-builtin", domainplugin.TypeSecret, []domainplugin.Capability{
			capability("secret.store_development", "Store local development secrets behind SecretRef metadata"),
		}, now),
		builtin("notification-noop", domainplugin.TypeNotification, []domainplugin.Capability{
			capability("notification.noop", "Record notification requests without external delivery"),
		}, now),
		builtin("policy-builtin", domainplugin.TypePolicy, []domainplugin.Capability{
			capability("policy.evaluate", "Evaluate minimal built-in delivery gate rules"),
		}, now),
		builtin("scanner-noop", domainplugin.TypeScanner, []domainplugin.Capability{
			capability("security.scan_noop", "Return deterministic no-op security scan results for tests"),
		}, now),
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

func builtin(name string, pluginType domainplugin.Type, capabilities []domainplugin.Capability, now time.Time) domainplugin.Manifest {
	return domainplugin.Manifest{
		Name:         name,
		Type:         pluginType,
		Version:      "0.1.0-dev",
		Protocol:     "builtin",
		Capabilities: capabilities,
		Status:       domainplugin.StatusBuiltIn,
		Metadata: map[string]string{
			"phase": "4.3",
			"safe":  "true",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func capability(name string, description string) domainplugin.Capability {
	return domainplugin.Capability{Name: name, Description: description}
}

func cloneManifest(manifest domainplugin.Manifest) domainplugin.Manifest {
	manifest.Capabilities = append([]domainplugin.Capability(nil), manifest.Capabilities...)
	if manifest.Metadata != nil {
		metadata := make(map[string]string, len(manifest.Metadata))
		for key, value := range manifest.Metadata {
			metadata[key] = value
		}
		manifest.Metadata = metadata
	}
	return manifest
}
