# Backup, Restore, and Migration Drill

Comprehensive backup/restore and migration drill for PostgreSQL-backed Nivora runtime state. The drill script (`scripts/drill-backup-restore-postgres.sh`) exercises migration, representative record creation, backup, and restart verification. The narrower CI smoke script (`scripts/smoke-backup-restore-postgres.sh`) performs the actual pg_dump restore into a temporary database when PostgreSQL permissions allow it.

## Quick Start

```bash
# Full drill (requires PostgreSQL)
DATABASE_URL="postgres://nivora:nivora@localhost:5432/nivora?sslmode=disable" make drill-backup-restore

# Migration drill only
DATABASE_URL="..." NIVORA_RUN_POSTGRES_INTEGRATION=true make drill-migrations

# Skip
SKIP_DRILL=1 make drill-backup-restore
```

## Safety Guards

The drill refuses to run if `DATABASE_URL` looks like a production database:
- URLs containing `prod`, `production`, `live`
- URLs containing AWS RDS patterns (`rds.amazonaws`)
- URLs containing Aliyun RDS patterns (`rds.aliyuncs`)
- URLs containing Tencent Cloud DB patterns (`tencentcdb`)

Override with `NIVORA_ALLOW_PRODUCTION_DRILL=true` only in controlled environments.

The drill does NOT:
- Print secret values
- Drop production databases
- Require elevated privileges beyond CREATE/DROP SCHEMA

## Phases

### Phase 1: Migration Drill
- Verifies migration pairs are reversible (up/down count match)
- Runs up/down/up cycle via integration tests
- Validates schema integrity by checking all expected tables exist

### Phase 2: Representative Record Insertion
Creates records across all governance and runtime stores:
- **Runtime**: PipelineRun, DeploymentRun, Release, ReleaseExecution
- **Auth**: ServiceAccount + API token metadata (hashes only, no raw tokens)
- **Credential**: Credential metadata with SecretRef (metadata only, no secret values)
- **Governance**: Approval, SecurityScan, CloudAccount
- **Tenancy**: Quota

### Phase 3: Backup
- Stops the server
- Runs `pg_dump --no-owner --no-privileges`
- Verifies dump size > 500 bytes
- Verifies key tables appear in the dump

### Phase 4: Restart Verification
- Restarts the server
- Verifies PipelineRun, DeploymentRun, and Credential survived stop/restart
- Checks audit chain verification availability

For actual restore proof into a fresh temporary database, run:

```bash
DATABASE_URL="postgres://..." NIVORA_REQUIRE_ACTUAL_RESTORE=1 make smoke-backup-restore
```

## Tested vs Manual

**Tested in this drill:**
- Migration up/down/up cycle
- Schema table presence
- Server startup with Postgres
- Record creation via API across 9 store types
- pg_dump backup
- Record survival across stop/restart
- Audit chain verification

**Not tested in this drill:**
- Actual restore into a fresh temporary database by this comprehensive drill script. Use `make smoke-backup-restore` for the CI-covered temporary database restore path.
- Large-scale backup (>1GB)
- Point-in-time recovery (PITR)
- Object store backup (filesystem/S3)
- Encrypted backups
- Production-scale verification

## Interpreting Failures

| Failure | Likely Cause |
|---|---|
| Migration count mismatch | New migration without down pair |
| Tables missing | Migration not run or schema corruption |
| Records not created | Server not started or API error |
| pg_dump too small | Empty database or dump failure |
| Records lost after restart | Persistence issue in Postgres store |

## CI Integration

```yaml
backup-restore-drill:
  runs-on: ubuntu-latest
  services:
    postgres:
      image: postgres:16-alpine
  env:
    DATABASE_URL: "postgres://nivora:nivora@localhost:5432/nivora?sslmode=disable"
    NIVORA_RUN_POSTGRES_INTEGRATION: "true"
  steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
      with: { go-version: "1.22" }
    - run: make drill-backup-restore
    - run: make drill-migrations
```
