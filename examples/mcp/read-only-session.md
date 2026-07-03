# MCP Read-Only Session

This example shows a local stdio MCP session. It is safe for local development and does not require external services.

Start the server:

```bash
go run ./cmd/nivora-mcp --config configs/server.yaml --stdio
```

Send line-delimited JSON-RPC:

```json
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}
{"jsonrpc":"2.0","id":2,"method":"resources/list","params":{}}
{"jsonrpc":"2.0","id":3,"method":"tools/list","params":{}}
{"jsonrpc":"2.0","id":4,"method":"resources/read","params":{"uri":"nivora://capabilities/current"}}
```

Blocked action example:

```json
{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"nivora_apply_deployment","arguments":{"id":"dep-example"}}}
```

The response should contain `mcp_action_not_allowed`.

Do not use MCP to request secrets, raw tokens, token hashes, kubeconfigs, private keys, or Authorization headers.
