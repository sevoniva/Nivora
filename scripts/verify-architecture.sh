#!/usr/bin/env bash
set -euo pipefail

echo "Checking Nivora architecture boundaries..."

if [ ! -d internal ]; then
  echo "internal/ does not exist yet; skipping architecture import checks."
  exit 0
fi

check_forbidden() {
  local path="$1"
  local pattern="$2"
  local message="$3"

  if [ -d "$path" ] && grep -R --include='*.go' -n "$pattern" "$path" >/tmp/nivora_arch_check.txt 2>/dev/null; then
    echo "$message"
    cat /tmp/nivora_arch_check.txt
    rm -f /tmp/nivora_arch_check.txt
    exit 1
  fi

  rm -f /tmp/nivora_arch_check.txt
}

check_forbidden "internal/domain" "internal/api" "domain must not import internal/api"
check_forbidden "internal/domain" "internal/app" "domain must not import internal/app"
check_forbidden "internal/domain" "internal/adapters" "domain must not import internal/adapters"
check_forbidden "internal/domain" "internal/infra" "domain must not import internal/infra"
check_forbidden "internal/domain" "github.com/go-chi/chi" "domain must not import HTTP router packages"
check_forbidden "internal/domain" "github.com/jackc/pgx" "domain must not import database drivers"
check_forbidden "internal/domain" "k8s.io/client-go" "domain must not import Kubernetes clients"
check_forbidden "internal/domain" "github.com/aws/aws-sdk-go" "domain must not import AWS SDK"
check_forbidden "internal/domain" "github.com/aliyun" "domain must not import Aliyun SDK"
check_forbidden "internal/domain" "github.com/tencentcloud" "domain must not import Tencent Cloud SDK"

check_forbidden "internal/usecase" "internal/adapters" "usecase must not import concrete adapters"
check_forbidden "internal/usecase" "github.com/aws/aws-sdk-go" "usecase must not import AWS SDK directly"
check_forbidden "internal/usecase" "k8s.io/client-go" "usecase must not import Kubernetes clients directly"

if find . -path './.git' -prune -o -type d -name utils -print | grep -q .; then
  echo "Directory named utils found. Avoid vague utility packages unless explicitly justified."
  find . -path './.git' -prune -o -type d -name utils -print
  exit 1
fi

echo "Architecture check passed."
