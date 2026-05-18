GO ?= go
GOPROXY ?= https://proxy.golang.org,direct
DATABASE_URL ?= postgres://nivora:nivora@localhost:5432/nivora?sslmode=disable

.PHONY: build test vet lint fmt fmt-check tidy tidy-check verify-architecture verify-no-secrets verify run-server run-worker run-runner dev-up dev-down migrate-up migrate-down

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

verify: fmt-check tidy-check vet test build verify-architecture verify-no-secrets

run-server:
	GOPROXY=$(GOPROXY) $(GO) run ./cmd/nivora server --config configs/server.yaml

run-worker:
	GOPROXY=$(GOPROXY) $(GO) run ./cmd/nivora worker --config configs/worker.yaml

run-runner:
	GOPROXY=$(GOPROXY) $(GO) run ./cmd/nivora runner --config configs/runner.yaml

dev-up:
	./scripts/dev-up.sh

dev-down:
	./scripts/dev-down.sh

migrate-up:
	GOPROXY=$(GOPROXY) $(GO) run github.com/pressly/goose/v3/cmd/goose@latest -dir internal/infra/migration postgres "$(DATABASE_URL)" up

migrate-down:
	GOPROXY=$(GOPROXY) $(GO) run github.com/pressly/goose/v3/cmd/goose@latest -dir internal/infra/migration postgres "$(DATABASE_URL)" down
