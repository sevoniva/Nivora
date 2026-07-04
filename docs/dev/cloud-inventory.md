# Cloud Inventory

Phase 8.0 supports local cloud inventory APIs and CLI commands backed by deterministic provider foundations. Real credentials are optional only and are not required for CI.

## API

```sh
curl -s http://localhost:8080/api/v1/cloud/providers \
  -H "Authorization: Bearer ${NIVORA_AUTH_TOKEN}"
curl -s http://localhost:8080/api/v1/cloud/accounts \
  -H "Authorization: Bearer ${NIVORA_AUTH_TOKEN}" \
  -H 'content-type: application/json' \
  -d '{"name":"dev-aws","provider":"aws","credentialRef":"credential-ref-placeholder"}'
curl -s http://localhost:8080/api/v1/cloud/accounts/<id>/inventory \
  -H "Authorization: Bearer ${NIVORA_AUTH_TOKEN}"
```

## CLI

Local provider metadata can be inspected without contacting the server:

```sh
nivora cloud providers --local
```

Server-backed cloud account and inventory commands are RBAC-protected. Use `--token-env` so the token stays out of shell history:

```sh
nivora cloud account create \
  --name dev-aws \
  --provider aws \
  --credential-ref credential-ref-placeholder \
  --default-region us-test-1 \
  --metadata owner=platform \
  --token-env NIVORA_AUTH_TOKEN
nivora cloud account list --token-env NIVORA_AUTH_TOKEN
nivora cloud account get <account-id> --token-env NIVORA_AUTH_TOKEN
nivora cloud account validate <account-id> --token-env NIVORA_AUTH_TOKEN
nivora cloud inventory <account-id> --token-env NIVORA_AUTH_TOKEN
nivora cloud clusters <account-id> --token-env NIVORA_AUTH_TOKEN
nivora cloud hosts <account-id> --token-env NIVORA_AUTH_TOKEN
nivora cloud registries <account-id> --token-env NIVORA_AUTH_TOKEN
```

## Credentials

Cloud credentials should be represented by `CredentialRef` or `SecretRef`. The CLI account create command accepts `--credential-ref` metadata only; it does not accept provider access keys, passwords, bearer tokens, private keys, or kubeconfigs. `--token-env` is for the Nivora API bearer token only. Do not put access keys, secret keys, tokens, or realistic fake credentials in example files, config files, logs, or audit records.

## Limits

Cloud account metadata can be created, listed, fetched, validated, and used for deterministic fake/provider-skeleton inventory through the API and CLI. AWS, Aliyun, and Tencent adapters expose provider capability metadata, config validation, inventory snapshots, and target binding metadata. Real provider SDK integration, pagination, filtering, tagging, permission discovery, and cloud deployments are future work.
