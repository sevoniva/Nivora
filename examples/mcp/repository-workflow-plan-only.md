# MCP Repository Workflow Plan-Only Session

This example is for a local MCP client. It does not execute repository code or mutate delivery state.

## Inspect A Local Repository

Tool:

```json
{
  "name": "nivora_repository_inspect",
  "arguments": {
    "path": "/replace/with/local/repository/path",
    "name": "local-service"
  }
}
```

Expected behavior:

- returns `mutated: false`
- returns snapshot metadata
- returns static repository intelligence
- records secret-like files as metadata only

## Plan A Workflow

Tool:

```json
{
  "name": "nivora_workflow_plan",
  "arguments": {
    "content": "apiVersion: nivora.io/v1alpha1\nkind: Workflow\nmetadata:\n  name: go-ci\non: [manual]\njobs:\n  test:\n    steps:\n      - name: test\n        run: go test ./...\n"
  }
}
```

Expected behavior:

- returns `mutated: false`
- returns a plan-only DAG
- does not create a PipelineRun
- does not execute shell commands

Blocked actions remain blocked through MCP, including Git push, deployment apply, Argo CD sync, rollback execution, secret retrieval, and token rotation.
