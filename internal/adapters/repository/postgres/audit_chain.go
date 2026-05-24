package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sevoniva/nivora/internal/domain/audit"
)

// AppendHashChainedAudit writes an audit entry to both the governance_audit_logs
// table (for querying) and the compliance_audit_records table (for tamper-evident
// hash chain verification).
func AppendHashChainedAudit(ctx context.Context, pool *pgxpool.Pool, source string, entry audit.AuditLog) error {
	// 1. Write plain audit log for queryability
	payload, _ := json.Marshal(entry)
	_, err := pool.Exec(ctx, `INSERT INTO governance_audit_logs (id, source, actor_id, action, subject, payload, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		entry.ID, source, entry.ActorID, entry.Action, entry.Subject, payload, entry.CreatedAt)
	if err != nil {
		return err
	}

	// 2. Write hash-chained record for tamper evidence
	prevHash, err := latestComplianceAuditHash(ctx, pool, source, "")
	if err != nil {
		return err
	}
	recordHash := computeAuditHash(prevHash, entry.ActorID, entry.Action, source, entry.Subject, source, "", entry.CreatedAt)

	_, err = pool.Exec(ctx, `INSERT INTO compliance_audit_records
		(id, actor_id, action, subject_type, subject_id, subject, scope_type, scope_id, correlation_id, request_id, previous_hash, record_hash, payload, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		entry.ID, entry.ActorID, entry.Action, source, entry.Subject, entry.Subject, source, "",
		entry.ID, "", prevHash, recordHash, payload, entry.CreatedAt)
	return err
}

func latestComplianceAuditHash(ctx context.Context, pool *pgxpool.Pool, scopeType, scopeID string) (string, error) {
	var hash string
	err := pool.QueryRow(ctx, `SELECT record_hash FROM compliance_audit_records WHERE scope_type=$1 AND ($2='' OR scope_id=$2) ORDER BY created_at DESC, id DESC LIMIT 1`, scopeType, scopeID).Scan(&hash)
	if err != nil {
		return "", nil
	}
	return hash, nil
}

func computeAuditHash(prevHash, actorID, action, subjectType, subjectID, scopeType, scopeID string, createdAt time.Time) string {
	canonical := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s",
		prevHash, actorID, action, subjectType, subjectID, scopeType, scopeID, createdAt.UTC().Format(time.RFC3339Nano))
	hash := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(hash[:])
}
