package plugin

import domainplugin "github.com/sevoniva/nivora/internal/domain/plugin"

type Registry struct {
	plugins map[string]domainplugin.Manifest
	order   []string
}

type CapabilityMatch struct {
	Plugin     domainplugin.Manifest   `json:"plugin"`
	Capability domainplugin.Capability `json:"capability"`
}
