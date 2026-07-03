#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

scenario_dir="examples/mcp/scenarios"
if [[ ! -d "$scenario_dir" ]]; then
  echo "MCP scenario validation failed: $scenario_dir is missing" >&2
  exit 1
fi

golden_dir="examples/mcp/golden-answers"
if [[ ! -d "$golden_dir" ]]; then
  echo "MCP scenario validation failed: $golden_dir is missing" >&2
  exit 1
fi

count="$(find "$scenario_dir" -name '*.yaml' -type f | wc -l | tr -d ' ')"
if [[ "$count" -lt 20 ]]; then
  echo "MCP scenario validation failed: expected at least 20 scenarios, found $count" >&2
  exit 1
fi

required_fields=(
  "id:"
  "title:"
  "operator_question:"
  "fixture_state:"
  "mcp:"
  "resources:"
  "tools:"
  "prompts:"
  "expected_facts:"
  "allowed_inference:"
  "unknowns:"
  "forbidden_claims:"
  "next_safe_checks:"
  "blocked_actions:"
  "redaction_samples:"
  "minimum_required_permissions:"
  "expected_answer_sections:"
  "test_expectations:"
)

required_answer_sections=(
  "# "
  "## Evidence Used"
  "## Facts"
  "## Inference"
  "## Unknowns"
  "## Blocked Actions"
  "## Safe Next Checks"
  "## Permissions"
  "## Safety Notes"
)

while IFS= read -r file; do
  for field in "${required_fields[@]}"; do
    if ! grep -q "$field" "$file"; then
      echo "MCP scenario validation failed: $file missing $field" >&2
      exit 1
    fi
  done
  id="$(awk -F': *' '/^id:/ {print $2; exit}' "$file" | tr -d '"'"'"'' )"
  if [[ -z "$id" ]]; then
    echo "MCP scenario validation failed: $file has no parseable id" >&2
    exit 1
  fi
  golden="$golden_dir/$id.md"
  if [[ ! -f "$golden" ]]; then
    echo "MCP scenario validation failed: missing golden answer $golden" >&2
    exit 1
  fi
  for section in "${required_answer_sections[@]}"; do
    if ! grep -q "$section" "$golden"; then
      echo "MCP scenario validation failed: $golden missing section $section" >&2
      exit 1
    fi
  done
  if grep -Eiq 'GA-ready|guaranteed safe|secret value:' "$file" "$golden"; then
    echo "MCP scenario validation failed: unsafe maturity or secret wording in $file" >&2
    exit 1
  fi
  if grep -Eiq 'was applied through MCP|was synced through MCP|was rolled back through MCP|approved through MCP|retrieved secret' "$golden"; then
    echo "MCP scenario validation failed: unsafe action claim in $golden" >&2
    exit 1
  fi
done < <(find "$scenario_dir" -name '*.yaml' -type f | sort)

for action in \
  nivora_apply_deployment \
  nivora_sync_argocd \
  nivora_execute_rollback \
  nivora_rollback_deployment \
  nivora_get_secret \
  nivora_rotate_token \
  nivora_register_runner \
  nivora_remote_host_deploy \
  nivora_git_push \
  nivora_kubernetes_prune \
  nivora_kubernetes_delete \
  nivora_approve \
  nivora_reject \
  nivora_approve_request \
  nivora_reject_request; do
  if ! grep -R -q "$action" "$scenario_dir"; then
    echo "MCP scenario validation failed: blocked action $action is not represented" >&2
    exit 1
  fi
done

for required_topic in prompt-injection runner-token tenant-scope rollback-readiness gitops-sync-safety host-deployment-safety kubernetes-prune-delete-safety; do
  if ! find "$scenario_dir" -name "*$required_topic*.yaml" -type f | grep -q .; then
    echo "MCP scenario validation failed: required topic $required_topic is missing" >&2
    exit 1
  fi
done

golden_count="$(find "$golden_dir" -name '*.md' -type f | wc -l | tr -d ' ')"
if [[ "$golden_count" -lt "$count" ]]; then
  echo "MCP scenario validation failed: expected at least $count golden answers, found $golden_count" >&2
  exit 1
fi

echo "MCP scenario validation passed ($count scenarios, $golden_count golden answers)"
