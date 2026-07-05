package postgres

import (
	"context"
	"strings"
	"testing"

	repositoryusecase "github.com/sevoniva/nivora/internal/usecase/repository"
)

func TestRepositoryStoreImplementsInterface(t *testing.T) {
	var _ repositoryusecase.Store = (*RepositoryStore)(nil)
}

func TestRepositoryWorkflowMigrationIsReversibleAndIndexed(t *testing.T) {
	up := readMigration(t, "000017_repository_workflow_persistence.up.sql")
	down := readMigration(t, "000017_repository_workflow_persistence.down.sql")

	for _, table := range []string{
		"repository_records",
		"repository_snapshots",
		"repository_intelligence",
	} {
		if !strings.Contains(up, "CREATE TABLE IF NOT EXISTS "+table) {
			t.Fatalf("up migration missing table %s", table)
		}
		if !strings.Contains(down, "DROP TABLE IF EXISTS "+table) {
			t.Fatalf("down migration missing table %s", table)
		}
	}
	for _, index := range []string{
		"idx_repository_records_project_id",
		"idx_repository_records_status",
		"idx_repository_snapshots_repository_created",
		"idx_repository_intelligence_snapshot_id",
	} {
		if !strings.Contains(up, index) {
			t.Fatalf("up migration missing index %s", index)
		}
	}
}

func TestPostgresIntegrationRepositorySnapshotIntelligenceRecovery(t *testing.T) {
	db := newPostgresIntegration(t, true)
	defer db.cleanup()
	ctx := context.Background()
	now := fixedIntegrationTime()
	store := NewRepositoryStore(db.pool)

	repository := repositoryusecase.Repository{
		ID:            "repo-durable",
		Name:          "durable-repository",
		Provider:      repositoryusecase.ProviderLocal,
		URL:           "file:///tmp/nivora-durable-repository",
		DefaultBranch: "main",
		ProjectID:     "project-a",
		Labels:        map[string]string{"team": "platform"},
		Metadata:      map[string]string{"source": "integration-test"},
		Status:        repositoryusecase.RepositoryStatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := store.SaveRepository(ctx, repository); err != nil {
		t.Fatalf("save repository: %v", err)
	}

	snapshot := repositoryusecase.RepositorySnapshot{
		ID:           "repo-snapshot-durable",
		RepositoryID: repository.ID,
		Ref:          "main",
		TreeHash:     "sha256:tree",
		Files: []repositoryusecase.RepositoryFile{
			{Path: "go.mod", Size: 128, Hash: "sha256:gomod"},
			{Path: ".env", Size: 64},
			{Path: ".nivora/workflows/build.yaml", Size: 256, Hash: "sha256:workflow"},
		},
		DetectedLanguages:       []string{"go"},
		DetectedBuildTools:      []string{"go"},
		DetectedPackageManagers: []string{"go-modules"},
		DetectedWorkflowFiles:   []string{".nivora/workflows/build.yaml"},
		Warnings:                []string{"secret-like file \".env\" recorded as metadata only; content was not read"},
		Metadata:                map[string]string{"provider": "local"},
		CreatedAt:               now,
	}
	if err := store.SaveSnapshot(ctx, snapshot); err != nil {
		t.Fatalf("save snapshot: %v", err)
	}
	intelligence := repositoryusecase.RepositoryIntelligence{
		RepositoryID:           repository.ID,
		SnapshotID:             snapshot.ID,
		LanguageSummary:        []string{"go"},
		FrameworkSummary:       []string{"go"},
		BuildCommandCandidates: []repositoryusecase.CommandCandidate{{Name: "go-build", Command: "go build ./...", Source: "go.mod"}},
		TestCommandCandidates:  []repositoryusecase.CommandCandidate{{Name: "go-test", Command: "go test ./...", Source: "go.mod"}},
		SecurityScanCandidates: []string{"gosec", "govulncheck"},
		Warnings:               []string{"plan-only: commands are not executed"},
		CreatedAt:              now,
	}
	if err := store.SaveIntelligence(ctx, intelligence); err != nil {
		t.Fatalf("save intelligence: %v", err)
	}

	restartedPool := db.restart(t)
	store = NewRepositoryStore(restartedPool)

	loadedRepository, err := store.GetRepository(ctx, repository.ID)
	if err != nil {
		t.Fatalf("reload repository: %v", err)
	}
	if loadedRepository.ProjectID != "project-a" || loadedRepository.Labels["team"] != "platform" {
		t.Fatalf("loaded repository = %#v", loadedRepository)
	}
	repositories, err := store.ListRepositories(ctx, "project-a")
	if err != nil || len(repositories) != 1 || repositories[0].ID != repository.ID {
		t.Fatalf("list repositories = %#v err=%v", repositories, err)
	}

	loadedSnapshot, err := store.GetLatestSnapshot(ctx, repository.ID)
	if err != nil {
		t.Fatalf("reload latest snapshot: %v", err)
	}
	if loadedSnapshot.ID != snapshot.ID || loadedSnapshot.TreeHash != snapshot.TreeHash || len(loadedSnapshot.Files) != 3 {
		t.Fatalf("loaded snapshot = %#v", loadedSnapshot)
	}
	var envFile repositoryusecase.RepositoryFile
	for _, file := range loadedSnapshot.Files {
		if file.Path == ".env" {
			envFile = file
			break
		}
	}
	if envFile.Path == "" || envFile.Hash != "" {
		t.Fatalf("secret-like file should persist as metadata-only, got %#v", envFile)
	}
	snapshots, err := store.ListSnapshots(ctx, repository.ID)
	if err != nil || len(snapshots) != 1 || snapshots[0].ID != snapshot.ID {
		t.Fatalf("list snapshots = %#v err=%v", snapshots, err)
	}

	loadedIntelligence, err := store.GetIntelligence(ctx, repository.ID, snapshot.ID)
	if err != nil {
		t.Fatalf("reload intelligence: %v", err)
	}
	if len(loadedIntelligence.BuildCommandCandidates) != 1 || loadedIntelligence.BuildCommandCandidates[0].Command != "go build ./..." {
		t.Fatalf("loaded intelligence = %#v", loadedIntelligence)
	}
}
