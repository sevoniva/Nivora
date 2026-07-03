# MCP Permission Matrix

Nivora MCP is a local stdio, read-only and plan-only control-plane surface. It is disabled by default and does not expose destructive action tools. Runner tokens are never allowed to use MCP.

| Name | Type | Required Permission | Allowed Roles | Runner Token Allowed | Secret Exposure Risk | Mutates State | Audit Event | Status |
|---|---|---|---|---|---|---|---|---|
| nivora://capabilities/current | resource | project.read | viewer, developer, maintainer, admin, owner | no | low | no | mcp.resource.read | implemented |
| nivora://system/runtime | resource | project.read | viewer, developer, maintainer, admin, owner | no | low | no | mcp.resource.read | implemented |
| nivora://api/inventory | resource | project.read | viewer, developer, maintainer, admin, owner | no | low | no | mcp.resource.read | implemented |
| nivora://pipelines/runs/{id} | resource | project.read | viewer, developer, maintainer, admin, owner | no | medium; logs and metadata are redacted/truncated | no | mcp.resource.read | implemented |
| nivora://pipelines/runs/{id}/timeline | resource | project.read | viewer, developer, maintainer, admin, owner | no | medium; event metadata is redacted | no | mcp.resource.read | implemented |
| nivora://pipelines/runs/{id}/logs | resource | project.read | viewer, developer, maintainer, admin, owner | no | medium; logs are truncated and redacted | no | mcp.resource.read | implemented |
| nivora://deployments/{id} | resource | project.read | viewer, developer, maintainer, admin, owner | no | medium; plans and warnings are redacted | no | mcp.resource.read | implemented |
| nivora://deployments/{id}/timeline | resource | project.read | viewer, developer, maintainer, admin, owner | no | medium; event metadata is redacted | no | mcp.resource.read | implemented |
| nivora://deployments/{id}/resources | resource | project.read | viewer, developer, maintainer, admin, owner | no | low; Secret resource data is not stored | no | mcp.resource.read | implemented |
| nivora://deployments/{id}/health | resource | project.read | viewer, developer, maintainer, admin, owner | no | low | no | mcp.resource.read | implemented |
| nivora://deployments/{id}/diff | resource | project.read | viewer, developer, maintainer, admin, owner | no | low | no | mcp.resource.read | implemented |
| nivora://releases/{id} | resource | project.read | viewer, developer, maintainer, admin, owner | no | medium; artifact refs are metadata-only | no | mcp.resource.read | implemented |
| nivora://releases/executions/{id} | resource | project.read | viewer, developer, maintainer, admin, owner | no | medium; target metadata only | no | mcp.resource.read | implemented |
| nivora://releases/executions/{id}/timeline | resource | project.read | viewer, developer, maintainer, admin, owner | no | medium; event metadata is redacted | no | mcp.resource.read | implemented |
| nivora://runners/summary | resource | project.read | viewer, developer, maintainer, admin, owner | no | medium; token hashes are never returned | no | mcp.resource.read | implemented |
| nivora://security/summary | resource | project.read | viewer, developer, maintainer, admin, owner | no | medium; findings are metadata and redacted | no | mcp.resource.read | implemented |
| nivora://audit/search | resource | audit.read | auditor, admin, owner | no | high; audit payload is redacted | no | mcp.resource.read | implemented |
| nivora://plugins/capabilities | resource | project.read | viewer, developer, maintainer, admin, owner | no | low | no | mcp.resource.read | implemented |
| nivora_status | tool | project.read | viewer, developer, maintainer, admin, owner | no | low | no | mcp.tool.called | implemented |
| nivora_get_pipeline_run | tool | project.read | viewer, developer, maintainer, admin, owner | no | medium; run metadata is redacted | no | mcp.tool.called | implemented |
| nivora_get_pipeline_timeline | tool | project.read | viewer, developer, maintainer, admin, owner | no | medium; event metadata is redacted | no | mcp.tool.called | implemented |
| nivora_get_deployment | tool | project.read | viewer, developer, maintainer, admin, owner | no | medium; plan metadata is redacted | no | mcp.tool.called | implemented |
| nivora_get_deployment_health | tool | project.read | viewer, developer, maintainer, admin, owner | no | low | no | mcp.tool.called | implemented |
| nivora_get_deployment_diff | tool | project.read | viewer, developer, maintainer, admin, owner | no | low | no | mcp.tool.called | implemented |
| nivora_get_release_execution | tool | project.read | viewer, developer, maintainer, admin, owner | no | medium; target metadata only | no | mcp.tool.called | implemented |
| nivora_get_runner_summary | tool | project.read | viewer, developer, maintainer, admin, owner | no | medium; token hashes are never returned | no | mcp.tool.called | implemented |
| nivora_search_audit | tool | audit.read | auditor, admin, owner | no | high; audit payload is redacted | no | mcp.tool.called | implemented |
| nivora_get_capability_status | tool | project.read | viewer, developer, maintainer, admin, owner | no | low | no | mcp.tool.called | implemented |
| nivora_explain_pipeline_failure | tool | deployment.create | developer, maintainer, admin, owner | no | medium; log preview is truncated and redacted | no | mcp.tool.called | plan-only |
| nivora_explain_deployment | tool | deployment.create | developer, maintainer, admin, owner | no | medium; plan metadata is redacted | no | mcp.tool.called | plan-only |
| nivora_explain_deployment_risk | tool | deployment.create | developer, maintainer, admin, owner | no | medium; compatibility alias for nivora_explain_deployment | no | mcp.tool.called | plan-only |
| nivora_explain_release | tool | deployment.create | developer, maintainer, admin, owner | no | medium; target metadata only | no | mcp.tool.called | plan-only |
| nivora_generate_release_readiness_summary | tool | deployment.create | developer, maintainer, admin, owner | no | medium; compatibility alias for nivora_explain_release | no | mcp.tool.called | plan-only |
| nivora_evaluate_policy_local | tool | deployment.create | developer, maintainer, admin, owner | no | medium; local input is redacted | no | mcp.tool.called | plan-only |
| nivora_inspect_artifact | tool | deployment.create | developer, maintainer, admin, owner | no | low; reference metadata only | no | mcp.tool.called | plan-only |
| nivora_inspect_artifact_reference | tool | deployment.create | developer, maintainer, admin, owner | no | low; compatibility alias for nivora_inspect_artifact | no | mcp.tool.called | plan-only |
| nivora_plan_deployment_local | tool | deployment.create | developer, maintainer, admin, owner | no | medium; local definition is parsed but not applied | no | mcp.tool.called | plan-only |
| nivora_apply_deployment | tool | not exposed | none | no | high | yes | mcp.tool.denied | denied |
| nivora_sync_argocd | tool | not exposed | none | no | high | yes | mcp.tool.denied | denied |
| nivora_execute_rollback | tool | not exposed | none | no | high | yes | mcp.tool.denied | denied |
| nivora_rollback_deployment | tool | not exposed | none | no | high | yes | mcp.tool.denied | denied |
| nivora_approve | tool | not exposed | none | no | medium | yes | mcp.tool.denied | denied |
| nivora_reject | tool | not exposed | none | no | medium | yes | mcp.tool.denied | denied |
| nivora_approve_request | tool | not exposed | none | no | medium | yes | mcp.tool.denied | denied |
| nivora_reject_request | tool | not exposed | none | no | medium | yes | mcp.tool.denied | denied |
| nivora_get_secret | tool | not exposed | none | no | critical | yes | mcp.tool.denied | denied |
| nivora_rotate_token | tool | not exposed | none | no | critical | yes | mcp.tool.denied | denied |
| nivora_register_runner | tool | not exposed | none | no | high | yes | mcp.tool.denied | denied |
| nivora_remote_host_deploy | tool | not exposed | none | no | high | yes | mcp.tool.denied | denied |
| nivora_git_push | tool | not exposed | none | no | high | yes | mcp.tool.denied | denied |
| nivora_kubernetes_prune | tool | not exposed | none | no | high | yes | mcp.tool.denied | denied |
| nivora_kubernetes_delete | tool | not exposed | none | no | high | yes | mcp.tool.denied | denied |
| diagnose_pipeline_run | prompt | project.read | viewer, developer, maintainer, admin, owner | no | medium; prompts instruct redaction | no | mcp.prompt.rendered | implemented |
| diagnose_deployment_run | prompt | project.read | viewer, developer, maintainer, admin, owner | no | medium; prompts instruct redaction | no | mcp.prompt.rendered | implemented |
| release_readiness_review | prompt | project.read | viewer, developer, maintainer, admin, owner | no | medium; prompts instruct redaction | no | mcp.prompt.rendered | implemented |
| audit_incident_summary | prompt | project.read | viewer, developer, maintainer, admin, owner | no | high; prompts instruct audit redaction | no | mcp.prompt.rendered | implemented |
| policy_gate_review | prompt | project.read | viewer, developer, maintainer, admin, owner | no | medium; prompts instruct redaction | no | mcp.prompt.rendered | implemented |
| runner_fleet_health_review | prompt | project.read | viewer, developer, maintainer, admin, owner | no | medium; prompts mention runner token sensitivity | no | mcp.prompt.rendered | implemented |
| mcp_safe_operation_check | prompt | project.read | viewer, developer, maintainer, admin, owner | no | low | no | mcp.prompt.rendered | implemented |
