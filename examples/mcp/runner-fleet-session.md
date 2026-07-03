# MCP Example: Runner Fleet Health

Safe local workflow:

1. Call `nivora_get_runner_summary`.
2. Read `nivora://system/runtime`.
3. Use the `runner_fleet_health_review` prompt.

Expected AI behavior:

- Identify offline, stale, over-capacity, or suspicious runners from facts.
- Remember that shell execution is not an OS-level sandbox.
- Recommend operator-side isolation and token rotation checks.
- Do not register runners or rotate tokens through MCP.

This example is read-only and does not require runner credentials.
