#!/usr/bin/env bash
set -euo pipefail

NIVORA_URL="${NIVORA_URL:-http://127.0.0.1:8080}"
NIVORA_LOAD_RUNS="${NIVORA_LOAD_RUNS:-20}"
NIVORA_LOAD_LOG_BYTES="${NIVORA_LOAD_LOG_BYTES:-4096}"

python3 - "$NIVORA_URL" "$NIVORA_LOAD_RUNS" "$NIVORA_LOAD_LOG_BYTES" <<'PY'
import json
import sys
import time
import urllib.error
import urllib.request

base = sys.argv[1].rstrip("/")
count = int(sys.argv[2])
log_bytes = int(sys.argv[3])
if log_bytes > 65536:
    print("NIVORA_LOAD_LOG_BYTES must be <= 65536 to match the API log chunk limit", file=sys.stderr)
    sys.exit(1)

payload = "x" * log_bytes
definition = {
    "apiVersion": "nivora.io/v1alpha1",
    "kind": "Pipeline",
    "metadata": {"name": "load-logs"},
    "spec": {"stages": [{"name": "build", "jobs": [{"name": "log", "executor": "shell", "steps": [{"name": "emit", "run": f"printf '{payload}'"}]}]}]},
}

latencies = []
for i in range(count):
    definition["metadata"]["name"] = f"load-logs-{i}"
    body = json.dumps(definition).encode()
    req = urllib.request.Request(base + "/api/v1/pipeline-runs", data=body, headers={"Content-Type": "application/json"}, method="POST")
    started = time.perf_counter()
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            data = json.loads(resp.read().decode())
    except urllib.error.HTTPError as exc:
        print(f"log run {i} failed: HTTP {exc.code} {exc.read().decode(errors='ignore')}", file=sys.stderr)
        sys.exit(1)
    run_id = data["run"]["id"]
    with urllib.request.urlopen(base + f"/api/v1/pipeline-runs/{run_id}/logs?limit=1", timeout=30) as resp:
        resp.read()
    latencies.append((time.perf_counter() - started) * 1000)

latencies.sort()
avg = sum(latencies) / len(latencies) if latencies else 0
p95 = latencies[int(len(latencies) * 0.95) - 1] if latencies else 0
print(f"generated_log_runs={count} bytes_per_run={log_bytes} avg_ms={avg:.2f} p95_ms={p95:.2f}")
PY
