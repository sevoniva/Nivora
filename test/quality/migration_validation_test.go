package quality_test

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"testing"
)

func TestMigrationsHaveReversiblePairs(t *testing.T) {
	files, err := filepath.Glob(filepath.Join(repoRoot(t), "internal/infra/migration/*.sql"))
	if err != nil {
		t.Fatalf("glob migrations: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no SQL migrations found")
	}

	up := map[string]string{}
	down := map[string]string{}
	re := regexp.MustCompile(`^([0-9]{6})_.+\.(up|down)\.sql$`)
	for _, path := range files {
		name := filepath.Base(path)
		matches := re.FindStringSubmatch(name)
		if matches == nil {
			t.Fatalf("migration %s does not follow NNNNNN_name.up/down.sql", name)
		}
		switch matches[2] {
		case "up":
			up[matches[1]] = path
		case "down":
			down[matches[1]] = path
		}
	}

	var versions []string
	for version, path := range up {
		versions = append(versions, version)
		if down[version] == "" {
			t.Fatalf("migration %s has up file %s but no down file", version, path)
		}
	}
	for version, path := range down {
		if up[version] == "" {
			t.Fatalf("migration %s has down file %s but no up file", version, path)
		}
	}
	sort.Strings(versions)
	for i, version := range versions {
		expected := i + 1
		if version != formatMigrationVersion(expected) {
			t.Fatalf("migration sequence gap at index %d: got %s want %s", i, version, formatMigrationVersion(expected))
		}
	}
}

func TestMigrationFilesAreNonEmpty(t *testing.T) {
	files, err := filepath.Glob(filepath.Join(repoRoot(t), "internal/infra/migration/*.sql"))
	if err != nil {
		t.Fatalf("glob migrations: %v", err)
	}
	for _, path := range files {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat %s: %v", path, err)
		}
		if info.Size() == 0 {
			t.Fatalf("%s is empty", path)
		}
	}
}

func formatMigrationVersion(version int) string {
	return fmt.Sprintf("%06d", version)
}
