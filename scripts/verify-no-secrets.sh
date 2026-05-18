#!/usr/bin/env bash
set -euo pipefail

echo "Checking for high-confidence secret patterns..."

patterns=(
  '-----BEGIN (RSA |DSA |EC |OPENSSH |PGP )?PRIVATE KEY-----'
  'AKIA[0-9A-Z]{16}'
  'ASIA[0-9A-Z]{16}'
  'ghp_[A-Za-z0-9_]{30,}'
  'github_pat_[A-Za-z0-9_]{30,}'
  'glpat-[A-Za-z0-9_-]{20,}'
  'xox[baprs]-[A-Za-z0-9-]{10,}'
  'sk-[A-Za-z0-9]{32,}'
  '-----BEGIN [A-Z ]*PRIVATE KEY-----'
  'Authorization:[[:space:]]*Bearer[[:space:]]+[A-Za-z0-9._-]{20,}'
  'password[[:space:]]*=[[:space:]]*["'\''][^"'\'']{12,}["'\'']'
)

exclude=(
  --exclude-dir=.git
  --exclude-dir=vendor
  --exclude-dir=node_modules
  --exclude-dir=.nivora
  --exclude='verify-no-secrets.sh'
)

for pattern in "${patterns[@]}"; do
  if grep -R -E -n "${exclude[@]}" "$pattern" . >/tmp/nivora_secret_check.txt 2>/dev/null; then
    echo "Possible secret detected for pattern: $pattern"
    cat /tmp/nivora_secret_check.txt
    rm -f /tmp/nivora_secret_check.txt
    exit 1
  fi
  rm -f /tmp/nivora_secret_check.txt
done

echo "No high-confidence secret patterns found."
