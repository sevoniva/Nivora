GO ?= go
GOPROXY ?= https://proxy.golang.org,direct
DATABASE_URL ?= postgres://nivora:nivora@localhost:5432/nivora?sslmode=disable

.PHONY: build test vet lint fmt fmt-check tidy tidy-check verify-architecture verify-no-secrets verify-runtime verify-deployment verify run-server run-worker run-runner pipeline-run-local deployment-plan-local deployment-dry-run-local deployment-run-local deployment-apply-local gitops-plan-local gitops-diff-local gitops-write-local argocd-status-local smoke-local smoke-api smoke-deployment-dry-run dev-up dev-down migrate-up migrate-down

build:
	GOPROXY=$(GOPROXY) $(GO) build ./cmd/nivora-server ./cmd/nivora-worker ./cmd/nivora-runner ./cmd/nivora

test:
	GOPROXY=$(GOPROXY) $(GO) test ./...

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

verify-deployment:
	GOPROXY=$(GOPROXY) $(GO) run ./cmd/nivora deployment plan --local examples/deployments/yaml-dry-run.yaml
	GOPROXY=$(GOPROXY) $(GO) run ./cmd/nivora deployment dry-run --local examples/deployments/yaml-dry-run.yaml

verify: fmt-check tidy-check vet test build verify-architecture verify-no-secrets verify-runtime verify-deployment

run-server:
	GOPROXY=$(GOPROXY) $(GO) run ./cmd/nivora server --config configs/server.yaml

run-worker:
	GOPROXY=$(GOPROXY) $(GO) run ./cmd/nivora worker --config configs/worker.yaml

run-runner:
	GOPROXY=$(GOPROXY) $(GO) run ./cmd/nivora runner --config configs/runner.yaml

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

gitops-plan-local:
	GOPROXY=$(GOPROXY) $(GO) run ./cmd/nivora gitops plan --local examples/deployments/argocd-plan.yaml

gitops-diff-local:
	GOPROXY=$(GOPROXY) $(GO) run ./cmd/nivora gitops diff --local examples/deployments/argocd-plan.yaml

gitops-write-local:
	@test "$$NIVORA_ALLOW_GITOPS_WRITE" = "true" || (echo "set NIVORA_ALLOW_GITOPS_WRITE=true to update ./tmp/gitops" >&2; exit 1)
	mkdir -p tmp/gitops
	cp -R examples/gitops/apps tmp/gitops/
	GOPROXY=$(GOPROXY) $(GO) run ./cmd/nivora gitops write --local examples/deployments/argocd-local-workingtree.yaml --working-tree ./tmp/gitops --confirm

argocd-status-local:
	GOPROXY=$(GOPROXY) $(GO) run ./cmd/nivora argocd status --app demo-springboot

smoke-local:
	./scripts/smoke-pipelinerun-local.sh

smoke-api:
	./scripts/smoke-api.sh

smoke-deployment-dry-run:
	./scripts/smoke-deployment-dry-run.sh

dev-up:
	./scripts/dev-up.sh

dev-down:
	./scripts/dev-down.sh

migrate-up:
	GOPROXY=$(GOPROXY) $(GO) run github.com/pressly/goose/v3/cmd/goose@latest -dir internal/infra/migration postgres "$(DATABASE_URL)" up

migrate-down:
	GOPROXY=$(GOPROXY) $(GO) run github.com/pressly/goose/v3/cmd/goose@latest -dir internal/infra/migration postgres "$(DATABASE_URL)" down
