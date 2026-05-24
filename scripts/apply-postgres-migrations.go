//go:build ignore

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_URL is required")
		os.Exit(1)
	}
	migrationDir := os.Getenv("NIVORA_MIGRATION_DIR")
	if migrationDir == "" {
		migrationDir = "internal/infra/migration"
	}

	files, err := filepath.Glob(filepath.Join(migrationDir, "*.up.sql"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "find migrations: %v\n", err)
		os.Exit(1)
	}
	sort.Strings(files)
	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "no up migrations found in %s\n", migrationDir)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect postgres: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	if alreadyApplied(ctx, pool) {
		fmt.Println("migrations already applied")
		return
	}

	for _, file := range files {
		body, err := os.ReadFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read %s: %v\n", file, err)
			os.Exit(1)
		}
		if _, err := pool.Exec(ctx, string(body)); err != nil {
			fmt.Fprintf(os.Stderr, "apply %s: %v\n", filepath.Base(file), err)
			os.Exit(1)
		}
		fmt.Printf("applied %s\n", filepath.Base(file))
	}
}

func alreadyApplied(ctx context.Context, pool *pgxpool.Pool) bool {
	var exists bool
	err := pool.QueryRow(ctx, `SELECT to_regclass('runtime_pipeline_runs') IS NOT NULL`).Scan(&exists)
	return err == nil && exists
}
