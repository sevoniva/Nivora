package mcp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	postgresrepo "github.com/sevoniva/nivora/internal/adapters/repository/postgres"
	domainauth "github.com/sevoniva/nivora/internal/domain/auth"
	"github.com/sevoniva/nivora/internal/infra/config"
	authusecase "github.com/sevoniva/nivora/internal/usecase/auth"
	complianceusecase "github.com/sevoniva/nivora/internal/usecase/compliance"
)

func TestPostgresIntegrationMCPAuditHashChain(t *testing.T) {
	db := newMCPPostgresIntegration(t)
	defer db.cleanup()

	ctx := context.Background()
	store := postgresrepo.NewComplianceStore(db.pool)
	compliance := complianceusecase.NewServiceWithStore(store, nil, nil, nil, nil, nil, nil)
	server := NewServer(Services{
		Config: config.Default(),
		Subject: domainauth.Subject{
			ID:        "mcp-postgres-auditor",
			Username:  "mcp-postgres-auditor",
			Roles:     []string{domainauth.RoleAuditor},
			AuthMode:  "service_account",
			ScopeType: "project",
			ScopeID:   "project-a",
		},
		Auth:       authusecase.NewService(authusecase.NewMemoryStore(), nil),
		Compliance: compliance,
		Audit:      NewComplianceAuditRecorder(compliance),
	}, nil)

	if _, err := server.ReadResource(ctx, "nivora://audit/search"); err != nil {
		t.Fatalf("read audit resource through MCP: %v", err)
	}
	time.Sleep(time.Millisecond)
	result, err := server.CallTool(ctx, "nivora_apply_deployment", map[string]any{
		"id":            "drun-denied",
		"authorization": "Bearer should-not-leak",
	})
	if err != nil {
		t.Fatalf("blocked MCP tool transport: %v", err)
	}
	if !result.IsError {
		t.Fatalf("blocked MCP tool unexpectedly succeeded: %#v", result)
	}

	search, err := compliance.SearchAudit(ctx, complianceusecase.AuditSearchInput{ActorID: "mcp-postgres-auditor"})
	if err != nil {
		t.Fatalf("search MCP audit through compliance service: %v", err)
	}
	if search.Count != 2 {
		t.Fatalf("expected 2 persisted MCP audit entries, got %#v", search)
	}
	for _, entry := range search.Items {
		if entry.ActorID != "mcp-postgres-auditor" || entry.SubjectType != "mcp" {
			t.Fatalf("unexpected MCP audit entry: %#v", entry)
		}
	}

	records := loadMCPAuditRecords(t, db.pool)
	if len(records) != 2 {
		t.Fatalf("expected 2 hash-chain records, got %#v", records)
	}
	if records[0].recordHash == "" || records[1].recordHash == "" || records[1].previousHash != records[0].recordHash {
		t.Fatalf("MCP audit chain records are not linked: %#v", records)
	}
	for _, record := range records {
		if got, want := record.recordHash, computeMCPAuditRecordHash(record); got != want {
			t.Fatalf("MCP audit record hash mismatch for %s: got %s want %s", record.id, got, want)
		}
		for _, forbidden := range []string{"should-not-leak", "tokenHash", "BEGIN PRIVATE KEY", "password", "private_key", "kubeconfig"} {
			if strings.Contains(record.payload, forbidden) {
				t.Fatalf("MCP audit payload leaked sensitive marker %q: %s", forbidden, record.payload)
			}
		}
	}
}

type mcpPostgresIntegration struct {
	admin       *pgxpool.Pool
	pool        *pgxpool.Pool
	databaseURL string
	schema      string
}

func newMCPPostgresIntegration(t *testing.T) *mcpPostgresIntegration {
	t.Helper()
	if os.Getenv("NIVORA_RUN_POSTGRES_INTEGRATION") != "true" {
		t.Skip("set NIVORA_RUN_POSTGRES_INTEGRATION=true and DATABASE_URL to run PostgreSQL MCP integration tests")
	}
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("DATABASE_URL is required when NIVORA_RUN_POSTGRES_INTEGRATION=true")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect admin postgres: %v", err)
	}
	schema := "nivora_mcp_it_" + time.Now().UTC().Format("20060102150405_000000000")
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		admin.Close()
		t.Fatalf("create schema: %v", err)
	}
	pool, err := pgxpool.New(ctx, mcpPostgresURLWithSearchPath(t, databaseURL, schema))
	if err != nil {
		_, _ = admin.Exec(ctx, "DROP SCHEMA IF EXISTS "+schema+" CASCADE")
		admin.Close()
		t.Fatalf("connect schema postgres: %v", err)
	}
	db := &mcpPostgresIntegration{admin: admin, pool: pool, databaseURL: databaseURL, schema: schema}
	for _, path := range mcpMigrationFiles(t) {
		raw, err := os.ReadFile(path)
		if err != nil {
			db.cleanup()
			t.Fatalf("read migration %s: %v", path, err)
		}
		if _, err := pool.Exec(ctx, string(raw)); err != nil {
			db.cleanup()
			t.Fatalf("execute migration %s: %v", path, err)
		}
	}
	return db
}

func (db *mcpPostgresIntegration) cleanup() {
	if db.pool != nil {
		db.pool.Close()
	}
	if db.admin != nil {
		_, _ = db.admin.Exec(context.Background(), "DROP SCHEMA IF EXISTS "+db.schema+" CASCADE")
		db.admin.Close()
	}
}

func mcpPostgresURLWithSearchPath(t *testing.T, raw string, schema string) string {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse DATABASE_URL: %v", err)
	}
	q := u.Query()
	q.Set("options", "-c search_path="+schema)
	u.RawQuery = q.Encode()
	return u.String()
}

func mcpMigrationFiles(t *testing.T) []string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("..", "..", "infra", "migration", "*.up.sql"))
	if err != nil {
		t.Fatalf("glob migrations: %v", err)
	}
	sort.Strings(files)
	if len(files) == 0 {
		t.Fatal("no migration files found")
	}
	return files
}

type mcpAuditRecordRow struct {
	id           string
	actorID      string
	action       string
	subjectType  string
	subjectID    string
	scopeType    string
	scopeID      string
	previousHash string
	recordHash   string
	payload      string
	createdAt    time.Time
}

func loadMCPAuditRecords(t *testing.T, pool *pgxpool.Pool) []mcpAuditRecordRow {
	t.Helper()
	rows, err := pool.Query(context.Background(), `SELECT id, actor_id, action, subject_type, subject_id, scope_type, scope_id, previous_hash, record_hash, payload::text, created_at
		FROM compliance_audit_records
		WHERE scope_type = 'mcp'
		ORDER BY created_at, id`)
	if err != nil {
		t.Fatalf("query MCP audit records: %v", err)
	}
	defer rows.Close()
	records := []mcpAuditRecordRow{}
	for rows.Next() {
		var record mcpAuditRecordRow
		if err := rows.Scan(&record.id, &record.actorID, &record.action, &record.subjectType, &record.subjectID, &record.scopeType, &record.scopeID, &record.previousHash, &record.recordHash, &record.payload, &record.createdAt); err != nil {
			t.Fatalf("scan MCP audit record: %v", err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate MCP audit records: %v", err)
	}
	return records
}

func computeMCPAuditRecordHash(record mcpAuditRecordRow) string {
	canonical := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s",
		record.previousHash,
		record.actorID,
		record.action,
		record.subjectType,
		record.subjectID,
		record.scopeType,
		record.scopeID,
		record.createdAt.UTC().Format(time.RFC3339Nano))
	hash := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(hash[:])
}
