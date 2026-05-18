package event

import "time"

type Event struct {
	SpecVersion     string         `json:"specversion"`
	ID              string         `json:"id"`
	Type            string         `json:"type"`
	Source          string         `json:"source"`
	Subject         string         `json:"subject,omitempty"`
	Time            time.Time      `json:"time"`
	DataContentType string         `json:"datacontenttype"`
	Data            map[string]any `json:"data,omitempty"`
}

type LogChunk struct {
	ID        string
	StreamID  string
	Sequence  int64
	Content   string
	CreatedAt time.Time
}
