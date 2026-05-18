GO ?= go
GOPROXY ?= https://goproxy.cn,direct
DATABASE_URL ?= postgres://nivora:nivora@localhost:5432/nivora?sslmode=disable

.PHONY: build test lint run-server run-worker run-runner dev-up dev-down migrate-up migrate-down tidy

build:
	GOPROXY=$(GOPROXY) $(GO) build ./cmd/nivora-server ./cmd/nivora-worker ./cmd/nivora-runner ./cmd/nivora

test:
	GOPROXY=$(GOPROXY) $(GO) test ./...

lint:
	GOPROXY=$(GOPROXY) $(GO) vet ./...

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

tidy:
	GOPROXY=$(GOPROXY) $(GO) mod tidy

