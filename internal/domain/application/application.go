package application

import "time"

type Application struct {
	ID        string
	ProjectID string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Service struct {
	ID            string
	ApplicationID string
	Name          string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Repository struct {
	ID        string
	ProjectID string
	Name      string
	URL       string
	Provider  string
	CreatedAt time.Time
	UpdatedAt time.Time
}
