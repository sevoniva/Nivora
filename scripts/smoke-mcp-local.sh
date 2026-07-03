#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

SENSITIVE_SAMPLE="mcp-sensitive-token-should-not-leak"

tools_json="$(go run ./cmd/nivora mcp list-tools --local)"
resources_json="$(go run ./cmd/nivora mcp list-resources --local)"

if [[ "$tools_json" != *"nivora_status"* ]]; then
  echo "MCP smoke failed: nivora_status missing from local tool list" >&2
  exit 1
fi

if [[ "$resources_json" != *"nivora://capabilities/current"* ]]; then
  echo "MCP smoke failed: capability resource missing from local resource list" >&2
  exit 1
fi

rpc_input="$(mktemp)"
rpc_output="$(mktemp)"
trap 'rm -f "$rpc_input" "$rpc_output"' EXIT

cat >"$rpc_input" <<JSON
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}
{"jsonrpc":"2.0","id":2,"method":"resources/read","params":{"uri":"nivora://capabilities/current"}}
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"nivora_status","arguments":{}}}
{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"nivora_apply_deployment","arguments":{"authorization":"Bearer ${SENSITIVE_SAMPLE}","password":"${SENSITIVE_SAMPLE}"}}}
JSON

go run ./cmd/nivora-mcp --config configs/server.yaml --stdio <"$rpc_input" >"$rpc_output"

if ! grep -q '"name":"nivora-mcp"' "$rpc_output"; then
  echo "MCP smoke failed: initialize did not return nivora-mcp server info" >&2
  cat "$rpc_output" >&2
  exit 1
fi

if ! grep -q 'hardened beta-candidate' "$rpc_output"; then
  echo "MCP smoke failed: capability status was not returned" >&2
  cat "$rpc_output" >&2
  exit 1
fi

if ! grep -q 'mcp_action_not_allowed' "$rpc_output"; then
  echo "MCP smoke failed: denied action did not return mcp_action_not_allowed" >&2
  cat "$rpc_output" >&2
  exit 1
fi

if grep -q "$SENSITIVE_SAMPLE" "$rpc_output"; then
  echo "MCP smoke failed: sensitive sample leaked in MCP output" >&2
  cat "$rpc_output" >&2
  exit 1
fi

echo "MCP local smoke passed"
