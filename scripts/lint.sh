#!/usr/bin/env sh
set -eu

GOPROXY="${GOPROXY:-https://proxy.golang.org,direct}" go vet ./...
