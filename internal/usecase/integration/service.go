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
			Maturity:               maturity(manifest.Name, capabilities),
			Capabilities:           capabilities,
			SafeByDefault:          manifest.Metadata["safe"] == "true",
			MutatesExternalSystems: manifest.Lifecycle.Execute,
			Notes:                  notes(manifest.Name, manifest.Protocol, capabilities),
			UpdatedAt:              manifest.UpdatedAt,
		})
	}
	result.Count = len(result.Integrations)
	return result, nil
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

func notes(name string, protocol string, capabilities []IntegrationCapability) []string {
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
