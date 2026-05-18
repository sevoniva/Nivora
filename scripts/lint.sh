#!/usr/bin/env sh
set -eu

GOPROXY="${GOPROXY:-https://goproxy.cn,direct}" go vet ./...

