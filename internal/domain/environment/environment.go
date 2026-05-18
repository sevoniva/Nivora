package environment

import "time"

type Environment struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"projectId,omitempty"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type ReleaseTarget struct {
	ID            string    `json:"id"`
	EnvironmentID string    `json:"environmentId"`
	Name          string    `json:"name"`
	TargetType    string    `json:"targetType"`
	Context       string    `json:"context,omitempty"`
	Namespace     string    `json:"namespace,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type EnvironmentLock struct {
	ID            string    `json:"id"`
	EnvironmentID string    `json:"environmentId"`
	Reason        string    `json:"reason"`
	LockedBy      string    `json:"lockedBy"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}
