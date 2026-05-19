package dto

import "time"

type GraphNode struct {
	ID       string         `json:"id"`
	Type     string         `json:"type"`
	Label    string         `json:"label"`
	Status   StatusBadge    `json:"status,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type GraphEdge struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
	Label  string `json:"label,omitempty"`
}

type TimelineItem struct {
	ID      string         `json:"id"`
	Type    string         `json:"type"`
	Time    time.Time      `json:"time"`
	Subject string         `json:"subject,omitempty"`
	Status  StatusBadge    `json:"status,omitempty"`
	Message string         `json:"message,omitempty"`
	Data    map[string]any `json:"data,omitempty"`
}

type StatusBadge struct {
	Value string `json:"value,omitempty"`
	Tone  string `json:"tone,omitempty"`
}

type ResourceNode struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	Name      string         `json:"name"`
	Namespace string         `json:"namespace,omitempty"`
	Status    StatusBadge    `json:"status,omitempty"`
	Health    StatusBadge    `json:"health,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

type EnvironmentTopology struct {
	EnvironmentID     string           `json:"environmentId"`
	Applications      []ResourceNode   `json:"applications,omitempty"`
	Targets           []ResourceNode   `json:"targets,omitempty"`
	LatestDeployments []ResourceNode   `json:"latestDeployments,omitempty"`
	Resources         []ResourceNode   `json:"resources,omitempty"`
	HealthSummary     DashboardSummary `json:"healthSummary"`
}

type DashboardSummary struct {
	ID        string            `json:"id,omitempty"`
	Title     string            `json:"title"`
	Status    StatusBadge       `json:"status,omitempty"`
	Counts    map[string]int    `json:"counts,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	UpdatedAt time.Time         `json:"updatedAt,omitempty"`
}

type SecuritySummary struct {
	DashboardSummary
	Findings map[string]int `json:"findings,omitempty"`
}

type RunnerSummary struct {
	DashboardSummary
	Runners []ResourceNode `json:"runners,omitempty"`
}
