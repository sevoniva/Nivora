package integration

import (
	"context"
	"time"

	domainplugin "github.com/sevoniva/nivora/internal/domain/plugin"
)

type PluginCatalog interface {
	List(ctx context.Context) ([]domainplugin.Manifest, error)
}

type Integration struct {
	Name                   string                  `json:"name"`
	Type                   string                  `json:"type"`
	Status                 string                  `json:"status"`
	Protocol               string                  `json:"protocol"`
	Maturity               string                  `json:"maturity"`
	Capabilities           []IntegrationCapability `json:"capabilities"`
	SafeByDefault          bool                    `json:"safeByDefault"`
	MutatesExternalSystems bool                    `json:"mutatesExternalSystems"`
	Notes                  []string                `json:"notes,omitempty"`
	UpdatedAt              time.Time               `json:"updatedAt,omitempty"`
}

type IntegrationCapability struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type ListResult struct {
	Integrations []Integration `json:"integrations"`
	Count        int           `json:"count"`
	Warnings     []string      `json:"warnings,omitempty"`
}
