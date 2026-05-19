#!/usr/bin/env bash
set -euo pipefail

NIVORA_URL="${NIVORA_URL:-http://127.0.0.1:8080}"
NIVORA_LOAD_RUNS="${NIVORA_LOAD_RUNS:-50}"

python3 - "$NIVORA_URL" "$NIVORA_LOAD_RUNS" <<'PY'
import json
import sys
import time
import urllib.error
import urllib.request

base = sys.argv[1].rstrip("/")
count = int(sys.argv[2])
definition = {
    "apiVersion": "nivora.io/v1alpha1",
    "kind": "Pipeline",
    "metadata": {"name": "load-run"},
    "spec": {"stages": [{"name": "build", "jobs": [{"name": "echo", "executor": "shell", "steps": [{"name": "say", "run": "printf load"}]}]}]},
}

latencies = []
for i in range(count):
    definition["metadata"]["name"] = f"load-run-{i}"
    body = json.dumps(definition).encode()
    req = urllib.request.Request(base + "/api/v1/pipeline-runs", data=body, headers={"Content-Type": "application/json"}, method="POST")
    started = time.perf_counter()
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            resp.read()
            status = resp.status
    except urllib.error.HTTPError as exc:
        print(f"run {i} failed: HTTP {exc.code} {exc.read().decode(errors='ignore')}", file=sys.stderr)
        sys.exit(1)
    latencies.append((time.perf_counter() - started) * 1000)
    if status != 201:
        print(f"run {i} failed: HTTP {status}", file=sys.stderr)
        sys.exit(1)

latencies.sort()
avg = sum(latencies) / len(latencies) if latencies else 0
p95 = latencies[int(len(latencies) * 0.95) - 1] if latencies else 0
print(f"created_runs={count} avg_ms={avg:.2f} p95_ms={p95:.2f}")
PY
