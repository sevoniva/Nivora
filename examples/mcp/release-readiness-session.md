# MCP Example: Release Readiness Review

Safe local workflow:

1. Read `nivora://releases/executions/<release-execution-id>`.
2. Read `nivora://releases/executions/<release-execution-id>/timeline`.
3. Optionally call `nivora_get_runner_summary`.
4. Optionally call `nivora_search_audit` when the subject has `audit.read`.
5. Use the `release_readiness_review` prompt.

Expected AI behavior:

- List target status facts.
- Call out policy, approval, artifact identity, and rollback-readiness gaps.
- Recommend only read-only or plan-only next checks.
- Avoid approval, rollback, sync, or deploy requests through MCP.

This example is backend-only and credential-free.
