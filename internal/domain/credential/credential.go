package credential

import "time"

type Credential struct {
	ID        string
	ProjectID string
	Name      string
	Type      string
	SecretRef SecretRef
	CreatedAt time.Time
	UpdatedAt time.Time
}

type SecretRef struct {
	Provider string
	Key      string
}
