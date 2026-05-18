package audit

import "time"

type AuditLog struct {
	ID        string
	OrgID     string
	ActorID   string
	Action    string
	Subject   string
	CreatedAt time.Time
}
