package quality_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBackupRestoreScriptsDoNotEchoRawDatabaseURL(t *testing.T) {
	scripts := []string{
		"scripts/smoke-backup-restore-postgres.sh",
		"scripts/drill-backup-restore-postgres.sh",
	}
	for _, script := range scripts {
		script := script
		t.Run(script, func(t *testing.T) {
			body, err := os.ReadFile(filepath.Join(repoRoot(t), script))
			if err != nil {
				t.Fatalf("read script: %v", err)
			}
			for lineNumber, line := range strings.Split(string(body), "\n") {
				trimmed := strings.TrimSpace(line)
				if !strings.HasPrefix(trimmed, "echo ") {
					continue
				}
				printsRawDatabaseURL := strings.Contains(trimmed, "$DATABASE_URL") || strings.Contains(trimmed, "${DATABASE_URL}")
				if printsRawDatabaseURL && !strings.Contains(trimmed, "redact_database_url") {
					t.Fatalf("%s:%d echoes DATABASE_URL without redaction: %s", script, lineNumber+1, strings.TrimSpace(line))
				}
			}
		})
	}
}
