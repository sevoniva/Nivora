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

type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Plugin   string   `json:"plugin,omitempty"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}
