# MCP Golden Scenarios

These YAML files describe safe AI/operator workflows for the local MCP control-plane foundation.

Each scenario records:

- the operator question
- fixture assumptions
- MCP resources, tools, and prompts that provide evidence
- facts AI may cite
- inference AI must label clearly
- unknowns AI must not hide
- forbidden claims
- safe next checks
- blocked action-shaped tools
- redaction samples
- minimum required permissions
- expected answer sections
- test expectations

Every scenario must have a matching markdown answer in `examples/mcp/golden-answers/`.

The scenarios and golden answers are validated by `internal/api/mcp/scenario_test.go` and `scripts/validate-mcp-scenarios.sh`.

They are not live production evidence. They are deterministic local fixtures for keeping MCP behavior, documentation, and safety prompts aligned.
