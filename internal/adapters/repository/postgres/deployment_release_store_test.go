package postgres

import (
	"strings"
	"testing"

	artifactusecase "github.com/sevoniva/nivora/internal/usecase/artifact"
	deploymentusecase "github.com/sevoniva/nivora/internal/usecase/deployment"
	releaseorchestration "github.com/sevoniva/nivora/internal/usecase/releaseorchestration"
)

func TestDeploymentReleaseStoresImplementRuntimeInterfaces(t *testing.T) {
	var _ deploymentusecase.Store = (*DeploymentStore)(nil)
	var _ artifactusecase.Store = (*ReleaseStore)(nil)
	var _ releaseorchestration.Store = (*ReleaseOrchestrationStore)(nil)
}

func TestDeploymentReleaseRuntimeMigrationIsReversibleAndIndexed(t *testing.T) {
	up := readMigration(t, "000007_deployment_release_runtime.up.sql")
	down := readMigration(t, "000007_deployment_release_runtime.down.sql")

	requiredTables := []string{
		"runtime_deployment_runs",
		"runtime_deployment_logs",
		"runtime_deployment_events",
		"runtime_deployment_audit_logs",
		"runtime_deployment_resources",
		"runtime_manifest_snapshots",
		"runtime_rollback_plans",
		"runtime_releases",
		"runtime_release_artifacts",
		"runtime_release_plans",
		"runtime_release_executions",
		"runtime_release_execution_targets",
		"runtime_release_execution_events",
		"runtime_release_execution_audit_logs",
	}
	for _, table := range requiredTables {
		if !strings.Contains(up, "CREATE TABLE IF NOT EXISTS "+table) {
			t.Fatalf("up migration missing table %s", table)
		}
		if !strings.Contains(down, "DROP TABLE IF EXISTS "+table) {
			t.Fatalf("down migration missing table %s", table)
		}
	}

	for _, index := range []string{
		"idx_runtime_deployment_runs_status_created_at",
		"idx_runtime_deployment_runs_lease",
		"idx_runtime_deployment_logs_run_sequence",
		"idx_runtime_deployment_resources_run_type",
		"idx_runtime_release_artifacts_release_id",
		"idx_runtime_release_executions_status_created_at",
		"idx_runtime_release_executions_lease",
	} {
		if !strings.Contains(up, index) {
			t.Fatalf("up migration missing index %s", index)
		}
		if !strings.Contains(down, index) {
			t.Fatalf("down migration missing index %s", index)
		}
	}
}
