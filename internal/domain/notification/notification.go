package notification

import "time"

type Notification struct {
	ID         string            `json:"id" yaml:"id"`
	Type       string            `json:"type" yaml:"type"`
	Channel    string            `json:"channel" yaml:"channel"`
	Subject    string            `json:"subject" yaml:"subject"`
	Body       string            `json:"body,omitempty" yaml:"body,omitempty"`
	Recipients []string          `json:"recipients,omitempty" yaml:"recipients,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	CreatedAt  time.Time         `json:"createdAt" yaml:"createdAt"`
}
