#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

required_docs=(
  "docs/status/ENTERPRISE_PRODUCTION_BASELINE.md"
  "docs/status/ENTERPRISE_READINESS_MATRIX.md"
  "docs/status/ENTERPRISE_PRODUCTION_READINESS_REVIEW.md"
  "docs/status/ENTERPRISE_RISK_REGISTER.md"
  "docs/status/ENTERPRISE_NEXT_GOALS.md"
  "docs/status/MCP_ENTERPRISE_OPENING_DECISION.md"
  "docs/status/MCP_ENTERPRISE_RISK_REGISTER.md"
  "docs/status/DELIVERY_ENGINE_PRODUCTION_BOUNDARY.md"
  "docs/security/ENTERPRISE_SECURITY_GAP_REVIEW.md"
)

for doc in "${required_docs[@]}"; do
  if [[ ! -s "$doc" ]]; then
    echo "enterprise readiness verification failed: missing or empty $doc" >&2
    exit 1
  fi
done

if ! grep -qi "not production-ready" docs/status/ENTERPRISE_PRODUCTION_READINESS_REVIEW.md; then
  echo "enterprise readiness verification failed: production readiness review must state not production-ready" >&2
  exit 1
fi

if ! grep -qi "remote.*no-go" docs/status/MCP_ENTERPRISE_OPENING_DECISION.md; then
  echo "enterprise readiness verification failed: MCP opening decision must keep remote MCP no-go" >&2
  exit 1
fi

if ! grep -qi "action MCP.*no-go" docs/status/MCP_ENTERPRISE_OPENING_DECISION.md; then
  echo "enterprise readiness verification failed: MCP opening decision must keep action MCP no-go" >&2
  exit 1
fi

risk_count="$(grep -Ec '^\| P[0-3] \|' docs/status/ENTERPRISE_RISK_REGISTER.md)"
if [[ "$risk_count" -lt 50 ]]; then
  echo "enterprise readiness verification failed: expected at least 50 enterprise risks, found $risk_count" >&2
  exit 1
fi

mcp_risk_count="$(grep -Ec '^\| P[0-3] \|' docs/status/MCP_ENTERPRISE_RISK_REGISTER.md)"
if [[ "$mcp_risk_count" -lt 20 ]]; then
  echo "enterprise readiness verification failed: expected at least 20 MCP risks, found $mcp_risk_count" >&2
  exit 1
fi

goal_count="$(grep -Ec '^## [0-9]+\. ' docs/status/ENTERPRISE_NEXT_GOALS.md)"
if [[ "$goal_count" -lt 10 ]]; then
  echo "enterprise readiness verification failed: expected at least 10 enterprise next goals, found $goal_count" >&2
  exit 1
fi

for topic in \
  "Runtime Recovery Closure" \
  "Enterprise Security Closure" \
  "Runner Sandbox and Fleet Hardening" \
  "MCP Remote Read-only Readiness" \
  "Tenant Isolation and Quota Hardening" \
  "Audit Evidence and Compliance Closure" \
  "Production Install and DR Drill" \
  "API Contract Stabilization" \
  "Observability and SLO Closure" \
  "Performance and Load Readiness"; do
  if ! grep -q "$topic" docs/status/ENTERPRISE_NEXT_GOALS.md; then
    echo "enterprise readiness verification failed: missing next goal $topic" >&2
    exit 1
  fi
done

scenario_count="$(find examples/mcp/scenarios -name '*.yaml' -type f | wc -l | tr -d ' ')"
golden_count="$(find examples/mcp/golden-answers -name '*.md' -type f | wc -l | tr -d ' ')"
if [[ "$scenario_count" -lt 25 || "$golden_count" -lt "$scenario_count" ]]; then
  echo "enterprise readiness verification failed: expected at least 25 MCP scenarios and matching golden answers, found scenarios=$scenario_count golden=$golden_count" >&2
  exit 1
fi

for required in \
  "tenant-idor-attempt" \
  "cross-project-audit-read-attempt" \
  "massive-log-response-truncation" \
  "missing-resource-lookup" \
  "evidence-bundle-request"; do
  if [[ ! -f "examples/mcp/scenarios/$required.yaml" ]]; then
    echo "enterprise readiness verification failed: missing MCP scenario $required" >&2
    exit 1
  fi
  if [[ ! -f "examples/mcp/golden-answers/$required.md" ]]; then
    echo "enterprise readiness verification failed: missing MCP golden answer $required" >&2
    exit 1
  fi
done

overclaims="$(grep -R -Ein 'is production-ready|has reached GA|GA has been reached|production-grade platform' README.md docs/status docs/security docs/operations docs/dev docs/releases \
  | grep -Eiv 'not|do not|does not|future|no-go|without maintainer|not evidence|no production-ready' || true)"
if [[ -n "$overclaims" ]]; then
  echo "enterprise readiness verification failed: found overclaiming production maturity language" >&2
  echo "$overclaims" >&2
  exit 1
fi

echo "Enterprise readiness verification passed ($risk_count enterprise risks, $mcp_risk_count MCP risks, $goal_count goals, $scenario_count MCP scenarios)."
