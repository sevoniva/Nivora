GO ?= go
GOPROXY ?= https://proxy.golang.org,direct
DATABASE_URL ?= postgres://nivora:nivora@localhost:5432/nivora?sslmode=disable

.PHONY: build test test-race test-postgres-integration benchmark load-generate-runs load-generate-logs load-simulate-runners coverage vet lint fmt fmt-check tidy tidy-check verify-core verify-contracts verify-architecture verify-no-secrets verify-runtime verify-runtime-recovery verify-api verify-cli verify-examples verify-api-specs verify-deployment verify-release verify-security verify-host verify-web verify-packaging verify-production-profiles verify-alpha validate-mcp-scenarios verify-mcp verify-ai-control-plane verify run-server run-worker run-runner run-web mcp-serve-local smoke-mcp-local docker-build docker-run helm-template helm-lint kind-install pipeline-run-local deployment-plan-local deployment-dry-run-local deployment-run-local deployment-apply-local host-deployment-plan-local host-deployment-run-local host-deployment-apply-local artifact-inspect-local oci-resolve-local release-plan-local release-deploy-local security-scan-local policy-evaluate-local gitops-plan-local gitops-deploy-local gitops-diff-local gitops-write-local argocd-status-local argocd-resources-local smoke-local smoke-api smoke-cli smoke-deployment-dry-run smoke-oci-resolve-local smoke-runtime-recovery-postgres smoke-multiprocess-recovery smoke-live-deploy verify-multiprocess-recovery smoke-production-install verify-production-install smoke-backup-restore smoke-helm-production-profile smoke-compose-production-profile smoke-audit-durability dev-up dev-down migrate-up migrate-down

build:
	GOPROXY=$(GOPROXY) $(GO) build ./cmd/nivora-server ./cmd/nivora-worker ./cmd/nivora-runner ./cmd/nivora ./cmd/nivora-mcp

test:
	GOPROXY=$(GOPROXY) $(GO) test ./...

test-postgres-integration:
	@if [ "$$NIVORA_RUN_POSTGRES_INTEGRATION" != "true" ]; then \
		echo "Skipping PostgreSQL recovery integration tests; set NIVORA_RUN_POSTGRES_INTEGRATION=true and DATABASE_URL to run them."; \
	else \
		GOPROXY=$(GOPROXY) DATABASE_URL="$(DATABASE_URL)" NIVORA_RUN_POSTGRES_INTEGRATION=true $(GO) test -p 1 -run 'TestPostgresIntegration' ./internal/adapters/repository/postgres ./internal/app/runtime; \
	fi

test-race:
	GOPROXY=$(GOPROXY) $(GO) test -race ./internal/usecase/... ./internal/api/http/...

benchmark:
	GOPROXY=$(GOPROXY) $(GO) test -run '^$$' -bench=. ./internal/usecase/pipeline ./internal/usecase/deployment

load-generate-runs:
	./scripts/load-generate-runs.sh

load-generate-logs:
	./scripts/load-generate-logs.sh

load-simulate-runners:
	./scripts/load-simulate-runners.sh

coverage:
	GOPROXY=$(GOPROXY) $(GO) test ./... -coverprofile=coverage.out
	GOPROXY=$(GOPROXY) $(GO) tool cover -func=coverage.out

vet:
	GOPROXY=$(GOPROXY) $(GO) vet ./...

lint: vet

fmt:
	gofmt -w $$(find . -path './.git' -prune -o -name '*.go' -print)

fmt-check:
	@test -z "$$(gofmt -l $$(find . -path './.git' -prune -o -name '*.go' -print))"

tidy:
	GOPROXY=$(GOPROXY) $(GO) mod tidy

tidy-check:
	GOPROXY=$(GOPROXY) $(GO) mod tidy
	git diff --exit-code go.mod go.sum

verify-architecture:
	./scripts/verify-architecture.sh

verify-no-secrets:
	./scripts/verify-no-secrets.sh

verify-runtime:
	./scripts/smoke-pipelinerun-local.sh

verify-runtime-recovery: test-postgres-integration

verify-api:
	./scripts/smoke-api.sh

verify-cli:
	./scripts/smoke-cli.sh

verify-examples:
	./scripts/validate-examples.sh

verify-api-specs:
	./scripts/verify-api-specs.sh

verify-deployment:
	GOPROXY=$(GOPROXY) $(GO) run ./cmd/nivora deployment plan --local examples/deployments/yaml-dry-run.yaml
	GOPROXY=$(GOPROXY) $(GO) run ./cmd/nivora deployment dry-run --local examples/deployments/yaml-dry-run.yaml

verify-release:
	GOPROXY=$(GOPROXY) $(GO) run ./cmd/nivora release plan --file examples/releases/multi-target-release.yaml --local
	GOPROXY=$(GOPROXY) $(GO) run ./cmd/nivora release deploy --file examples/releases/sequential-release.yaml --local

verify-security:
	GOPROXY=$(GOPROXY) $(GO) run ./cmd/nivora security scan manifest examples/security/manifest-privileged-warning.yaml --local
	GOPROXY=$(GOPROXY) $(GO) run ./cmd/nivora policy evaluate --subject registry.example.com/demo/app:latest

verify-host:
	GOPROXY=$(GOPROXY) $(GO) run ./cmd/nivora deployment host plan --file examples/deployments/host-dry-run.yaml --local
	GOPROXY=$(GOPROXY) $(GO) run ./cmd/nivora deployment host run --file examples/deployments/host-dry-run.yaml --local

verify-web:
	@if command -v npm >/dev/null 2>&1; then \
		cd web && npm ci && npm run typecheck && npm run build && echo "verify-web: passed"; \
	else \
		echo "[verify-web] SKIPPED — npm not found. Install Node.js to include web verification."; \
	fi

verify-packaging:
	@if command -v helm >/dev/null 2>&1; then \
		$(MAKE) helm-template; \
		$(MAKE) helm-lint; \
	else \
		echo "helm not found; skipping Helm template/lint checks"; \
	fi

verify-production-profiles: smoke-helm-production-profile smoke-compose-production-profile smoke-audit-durability

verify-alpha:
	./scripts/verify-alpha-release-docs.sh

validate-mcp-scenarios:
	./scripts/validate-mcp-scenarios.sh

verify-mcp:
	GOPROXY=$(GOPROXY) $(GO) test ./internal/api/mcp ./internal/app/mcp
	GOPROXY=$(GOPROXY) $(GO) build -o /tmp/nivora-mcp ./cmd/nivora-mcp
	GOPROXY=$(GOPROXY) $(GO) run ./cmd/nivora mcp list-tools --local >/tmp/nivora-mcp-tools.json
	GOPROXY=$(GOPROXY) $(GO) run ./cmd/nivora mcp list-resources --local >/tmp/nivora-mcp-resources.json
	$(MAKE) validate-mcp-scenarios
	./scripts/smoke-mcp-local.sh
	@echo "=== verify-mcp passed ==="

verify-ai-control-plane: validate-mcp-scenarios verify-mcp
	@echo "=== verify-ai-control-plane passed ==="

verify-core: fmt-check tidy-check vet test build verify-architecture verify-no-secrets
	@echo "=== verify-core passed (Go-only backend checks) ==="

verify-contracts: verify-api-specs
	@echo "=== verify-contracts passed (OpenAPI/AsyncAPI) ==="

verify: fmt-check tidy-check vet test build verify-architecture verify-no-secrets verify-examples verify-runtime verify-api verify-cli verify-api-specs verify-deployment verify-release verify-security verify-host verify-web verify-packaging verify-helm-safety verify-alpha verify-mcp
	@echo "=== verify passed ==="

verify-postgres:
	@if [ "$$NIVORA_RUN_POSTGRES_INTEGRATION" != "true" ]; then \
		echo "Skipping Postgres integration tests; set NIVORA_RUN_POSTGRES_INTEGRATION=true and DATABASE_URL"; \
	else \
		GOPROXY=$(GOPROXY) DATABASE_URL="$(DATABASE_URL)" NIVORA_RUN_POSTGRES_INTEGRATION=true $(GO) test -p 1 -count=1 -run 'TestPostgresIntegration' ./internal/adapters/repository/postgres ./internal/app/runtime; \
	fi

verify-helm-safety:
	./scripts/verify-helm-safety.sh

run-server:
	GOPROXY=$(GOPROXY) $(GO) run ./cmd/nivora server --config configs/server.yaml

run-worker:
	GOPROXY=$(GOPROXY) $(GO) run ./cmd/nivora worker --config configs/worker.yaml

run-runner:
	GOPROXY=$(GOPROXY) $(GO) run ./cmd/nivora runner --config configs/runner.yaml

run-web:
	cd web && npm install && npm run dev

mcp-serve-local:
	GOPROXY=$(GOPROXY) $(GO) run ./cmd/nivora-mcp --config configs/server.yaml --stdio

smoke-mcp-local:
	./scripts/smoke-mcp-local.sh

docker-build:
	docker build -t nivora:local .

docker-run:
	docker run --rm -p 8080:8080 -v "$$(pwd)/configs/server.yaml:/etc/nivora/server.yaml:ro" nivora:local server --config /etc/nivora/server.yaml

helm-template:
	helm template nivora deployments/helm

helm-lint:
	helm lint deployments/helm

smoke-helm-production-profile:
	./scripts/smoke-helm-production-profile.sh

smoke-compose-production-profile:
	./scripts/smoke-compose-production-profile.sh

smoke-compose-live:
	./scripts/smoke-compose-live.sh

smoke-production-install:
	./scripts/smoke-production-install.sh

smoke-audit-durability:
	./scripts/smoke-audit-durability.sh

kind-install:
	@test "$$NIVORA_ALLOW_KIND_INSTALL" = "true" || (echo "set NIVORA_ALLOW_KIND_INSTALL=true to install into the current Kubernetes context" >&2; exit 1)
	helm upgrade --install nivora deployments/helm

pipeline-run-local:
	GOPROXY=$(GOPROXY) $(GO) run ./cmd/nivora pipeline run --local examples/pipelines/simple-shell.yaml

deployment-plan-local:
	GOPROXY=$(GOPROXY) $(GO) run ./cmd/nivora deployment plan --local examples/deployments/yaml-dry-run.yaml

deployment-dry-run-local:
	GOPROXY=$(GOPROXY) $(GO) run ./cmd/nivora deployment dry-run --local examples/deployments/yaml-dry-run.yaml

deployment-run-local:
	GOPROXY=$(GOPROXY) $(GO) run ./cmd/nivora deployment run --local examples/deployments/yaml-dry-run.yaml

deployment-apply-local:
	@test "$$NIVORA_ALLOW_LOCAL_APPLY" = "true" || (echo "set NIVORA_ALLOW_LOCAL_APPLY=true to run local apply" >&2; exit 1)
	GOPROXY=$(GOPROXY) $(GO) run ./cmd/nivora deployment apply --local examples/deployments/yaml-apply-local.yaml --confirm

host-deployment-plan-local:
	GOPROXY=$(GOPROXY) $(GO) run ./cmd/nivora deployment host plan --file examples/deployments/host-dry-run.yaml --local

host-deployment-run-local:
	GOPROXY=$(GOPROXY) $(GO) run ./cmd/nivora deployment host run --file examples/deployments/host-dry-run.yaml --local

host-deployment-apply-local:
	@test "$$NIVORA_ALLOW_REMOTE_HOST_DEPLOY" = "true" || (echo "set NIVORA_ALLOW_REMOTE_HOST_DEPLOY=true to test guarded host apply" >&2; exit 1)
	GOPROXY=$(GOPROXY) $(GO) run ./cmd/nivora deployment host run --file examples/deployments/host-dry-run.yaml --local --confirm --allow-remote-host-deploy

artifact-inspect-local:
	GOPROXY=$(GOPROXY) $(GO) run ./cmd/nivora artifact inspect registry.example.com/team/demo:1.0.0

oci-resolve-local:
	GOPROXY=$(GOPROXY) $(GO) run ./cmd/nivora artifact resolve registry.example.com/team/demo@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa

release-plan-local:
	GOPROXY=$(GOPROXY) $(GO) run ./cmd/nivora release plan --file examples/releases/multi-target-release.yaml --local

release-deploy-local:
	GOPROXY=$(GOPROXY) $(GO) run ./cmd/nivora release deploy --file examples/releases/sequential-release.yaml --local

security-scan-local:
	GOPROXY=$(GOPROXY) $(GO) run ./cmd/nivora security scan manifest examples/security/manifest-privileged-warning.yaml --local

policy-evaluate-local:
	GOPROXY=$(GOPROXY) $(GO) run ./cmd/nivora policy evaluate --subject registry.example.com/demo/app:latest

gitops-plan-local:
	GOPROXY=$(GOPROXY) $(GO) run ./cmd/nivora gitops plan --local examples/deployments/argocd-plan.yaml

gitops-deploy-local:
	GOPROXY=$(GOPROXY) $(GO) run ./cmd/nivora gitops deploy --local examples/deployments/argocd-status-read.yaml

gitops-diff-local:
	GOPROXY=$(GOPROXY) $(GO) run ./cmd/nivora gitops diff --local examples/deployments/argocd-plan.yaml

gitops-write-local:
	@test "$$NIVORA_ALLOW_GITOPS_WRITE" = "true" || (echo "set NIVORA_ALLOW_GITOPS_WRITE=true to update ./tmp/gitops" >&2; exit 1)
	mkdir -p tmp/gitops
	cp -R examples/gitops/apps tmp/gitops/
	GOPROXY=$(GOPROXY) $(GO) run ./cmd/nivora gitops write --local examples/deployments/argocd-local-workingtree.yaml --working-tree ./tmp/gitops --confirm

argocd-status-local:
	GOPROXY=$(GOPROXY) $(GO) run ./cmd/nivora argocd status --app demo-springboot

argocd-resources-local:
	GOPROXY=$(GOPROXY) $(GO) run ./cmd/nivora argocd resources --app demo-springboot

smoke-local:
	./scripts/smoke-pipelinerun-local.sh

smoke-api:
	./scripts/smoke-api.sh

smoke-cli:
	./scripts/smoke-cli.sh

smoke-deployment-dry-run:
	./scripts/smoke-deployment-dry-run.sh

smoke-oci-resolve-local:
	./scripts/smoke-oci-resolve-local.sh

smoke-runtime-recovery-postgres:
	./scripts/smoke-runtime-recovery-postgres.sh

soak-runtime-postgres:
	./scripts/soak-runtime-postgres.sh

smoke-soak-runtime:
	NIVORA_SOAK_DURATION_SECONDS=10 NIVORA_SOAK_RUNS=2 ./scripts/soak-runtime-postgres.sh

drill-backup-restore:
	./scripts/drill-backup-restore-postgres.sh

drill-migrations:
	NIVORA_RUN_POSTGRES_INTEGRATION=true DATABASE_URL="$(DATABASE_URL)" go test -p 1 -count=1 -run "TestPostgresIntegrationMigrationUpDown" ./internal/adapters/repository/postgres

runbook-check-runtime:
	./scripts/runbook-check-runtime.sh

runbook-check-runner:
	./scripts/runbook-check-runner.sh

runbook-check-database:
	./scripts/runbook-check-database.sh

runbook-check-audit:
	./scripts/runbook-check-audit.sh

smoke-live-deploy:
	./scripts/smoke-live-deploy-postgres.sh

smoke-multiprocess-recovery:
	./scripts/smoke-multiprocess-recovery-postgres.sh

verify-multiprocess-recovery: smoke-multiprocess-recovery

verify-production-install: smoke-production-install

smoke-backup-restore:
	./scripts/smoke-backup-restore-postgres.sh

dev-up:
	./scripts/dev-up.sh

dev-down:
	./scripts/dev-down.sh

migrate-up:
	GOPROXY=$(GOPROXY) $(GO) run github.com/pressly/goose/v3/cmd/goose@latest -dir internal/infra/migration postgres "$(DATABASE_URL)" up

migrate-down:
	GOPROXY=$(GOPROXY) $(GO) run github.com/pressly/goose/v3/cmd/goose@latest -dir internal/infra/migration postgres "$(DATABASE_URL)" down
