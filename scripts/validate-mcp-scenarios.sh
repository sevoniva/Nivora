#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

scenario_dir="examples/mcp/scenarios"
if [[ ! -d "$scenario_dir" ]]; then
  echo "MCP scenario validation failed: $scenario_dir is missing" >&2
  exit 1
fi

count="$(find "$scenario_dir" -name '*.yaml' -type f | wc -l | tr -d ' ')"
if [[ "$count" -lt 8 ]]; then
  echo "MCP scenario validation failed: expected at least 8 scenarios, found $count" >&2
  exit 1
fi

required_fields=(
  "operator_question:"
  "fixture_state:"
  "resources:"
  "tools:"
  "prompts:"
  "safe_answer:"
  "forbidden_claims:"
  "next_safe_checks:"
  "blocked_actions:"
)

while IFS= read -r file; do
  for field in "${required_fields[@]}"; do
    if ! grep -q "$field" "$file"; then
      echo "MCP scenario validation failed: $file missing $field" >&2
      exit 1
    fi
  done
  if grep -Eiq 'GA-ready|guaranteed safe|secret value:' "$file"; then
    echo "MCP scenario validation failed: unsafe maturity or secret wording in $file" >&2
    exit 1
  fi
done < <(find "$scenario_dir" -name '*.yaml' -type f | sort)

for action in nivora_apply_deployment nivora_sync_argocd nivora_execute_rollback nivora_get_secret nivora_rotate_token; do
  if ! grep -R -q "$action" "$scenario_dir"; then
    echo "MCP scenario validation failed: blocked action $action is not represented" >&2
    exit 1
  fi
done

echo "MCP scenario validation passed ($count scenarios)"
