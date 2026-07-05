package integration

import (
	"context"
	"strings"
)

type Service struct {
	plugins PluginCatalog
}

func NewService(plugins PluginCatalog) *Service {
	return &Service{plugins: plugins}
}

func (s *Service) List(ctx context.Context) (ListResult, error) {
	if err := ctx.Err(); err != nil {
		return ListResult{}, err
	}
	manifests, err := s.plugins.List(ctx)
	if err != nil {
		return ListResult{}, err
	}
	result := ListResult{
		Integrations: make([]Integration, 0, len(manifests)),
		Warnings: []string{
			"integration entries describe configured adapter capabilities only; skeleton, noop, and foundation entries are not production integrations",
		},
	}
	for _, manifest := range manifests {
		capabilities := make([]IntegrationCapability, 0, len(manifest.Capabilities))
		for _, capability := range manifest.Capabilities {
			capabilities = append(capabilities, IntegrationCapability{
				Name:        capability.Name,
				Description: capability.Description,
			})
		}
		result.Integrations = append(result.Integrations, Integration{
			Name:                   manifest.Name,
			Type:                   string(manifest.Type),
			Status:                 string(manifest.Status),
			Protocol:               manifest.Protocol,
			Maturity:               metadataOr(manifest.Metadata, "maturity", maturity(manifest.Name, capabilities)),
			AdapterKind:            metadataOr(manifest.Metadata, "adapterKind", adapterKind(manifest.Name, capabilities)),
			Boundary:               metadataOr(manifest.Metadata, "boundary", boundary(manifest.Name, manifest.Protocol, capabilities)),
			CredentialMode:         metadataOr(manifest.Metadata, "credentialMode", "none"),
			NetworkAccess:          metadataOr(manifest.Metadata, "networkAccess", "none"),
			Capabilities:           capabilities,
			SafeByDefault:          manifest.Metadata["safe"] == "true",
			DefaultMutation:        manifest.Metadata["defaultMutation"] == "true",
			MutatesExternalSystems: manifest.Lifecycle.Execute,
			Notes:                  notes(manifest.Name, manifest.Protocol, manifest.Metadata, capabilities),
			UpdatedAt:              manifest.UpdatedAt,
		})
	}
	result.Count = len(result.Integrations)
	return result, nil
}

func metadataOr(metadata map[string]string, key string, fallback string) string {
	if metadata == nil {
		return fallback
	}
	if value := strings.TrimSpace(metadata[key]); value != "" {
		return value
	}
	return fallback
}

func maturity(name string, capabilities []IntegrationCapability) string {
	text := strings.ToLower(name + " " + capabilityText(capabilities))
	switch {
	case strings.Contains(text, "placeholder"):
		return "placeholder"
	case strings.Contains(text, "argocd") || strings.Contains(text, "kubernetes_yaml"):
		return "experimental"
	case strings.Contains(text, "skeleton"):
		return "foundation"
	case strings.Contains(text, "noop") || strings.Contains(text, "no-op"):
		return "foundation"
	default:
		return "foundation"
	}
}

func adapterKind(name string, capabilities []IntegrationCapability) string {
	text := strings.ToLower(name + " " + capabilityText(capabilities))
	switch {
	case strings.Contains(text, "noop") || strings.Contains(text, "no-op"):
		return "noop"
	case strings.Contains(text, "fake") || strings.Contains(text, "deterministic local"):
		return "fake"
	case strings.Contains(text, "placeholder"):
		return "placeholder"
	case strings.Contains(text, "skeleton"):
		return "skeleton"
	default:
		return "foundation"
	}
}

func boundary(name string, protocol string, capabilities []IntegrationCapability) string {
	text := strings.ToLower(name + " " + protocol + " " + capabilityText(capabilities))
	switch {
	case strings.Contains(text, "guarded_sync") || strings.Contains(text, "noop_apply"):
		return "guarded-action"
	case strings.Contains(text, "plan"):
		return "plan-only"
	case strings.Contains(text, "noop") || strings.Contains(text, "no-op"):
		return "noop"
	case strings.Contains(text, "skeleton") || strings.Contains(text, "placeholder"):
		return "metadata-only"
	default:
		return "read-only"
	}
}

func notes(name string, protocol string, metadata map[string]string, capabilities []IntegrationCapability) []string {
	items := []string{"metadata-only integration entry; no credentials or secret values are returned"}
	text := strings.ToLower(name + " " + protocol + " " + capabilityText(capabilities))
	if strings.Contains(text, "skeleton") || strings.Contains(text, "placeholder") {
		items = append(items, "adapter is a skeleton or placeholder and should not be treated as complete")
	}
	if strings.Contains(text, "noop") || strings.Contains(text, "no-op") {
		items = append(items, "adapter uses noop behavior for local tests or foundation flows")
	}
	if strings.Contains(text, "argocd") {
		items = append(items, "Argo CD sync remains guarded and disabled unless explicitly allowed")
	}
	if metadataOr(metadata, "networkAccess", "none") == "optional" {
		items = append(items, "network access is optional and must be explicitly configured")
	}
	if metadataOr(metadata, "credentialMode", "none") != "none" {
		items = append(items, "credentials are represented by references only; secret values are not returned")
	}
	return items
}

func capabilityText(capabilities []IntegrationCapability) string {
	var builder strings.Builder
	for _, capability := range capabilities {
		builder.WriteString(" ")
		builder.WriteString(capability.Name)
		builder.WriteString(" ")
		builder.WriteString(capability.Description)
	}
	return builder.String()
}
