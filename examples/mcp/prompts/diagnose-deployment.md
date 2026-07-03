# Diagnose Deployment Prompt

Use the `diagnose_deployment_run` MCP prompt with:

```json
{
  "name": "diagnose_deployment_run",
  "arguments": {
    "id": "dep-example"
  }
}
```

The model should:

- read the DeploymentRun, timeline, resource inventory, health, and diff resources
- cite the resources or tools used
- separate facts from inference
- identify risks and missing evidence
- recommend guarded next checks
- avoid apply, sync, rollback execution, prune, or delete actions through MCP
- avoid production-ready claims
