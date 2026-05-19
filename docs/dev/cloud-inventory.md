# Cloud Inventory

Phase 3.4 adds local cloud inventory APIs and CLI commands backed by fake provider adapters.

## API

```sh
curl -s http://localhost:8080/api/v1/cloud/providers
curl -s http://localhost:8080/api/v1/cloud/accounts \
  -H 'content-type: application/json' \
  -d '{"name":"dev-aws","provider":"aws","credentialRef":"credential-ref-placeholder"}'
curl -s http://localhost:8080/api/v1/cloud/accounts/<id>/inventory
```

## CLI

```sh
nivora cloud providers
nivora cloud account validate <account-id>
nivora cloud inventory <account-id>
nivora cloud clusters <account-id>
nivora cloud hosts <account-id>
nivora cloud registries <account-id>
```

## Credentials

Cloud credentials should be represented by `CredentialRef` or `SecretRef`. Do not put access keys, secret keys, tokens, or realistic fake credentials in example files, config files, logs, or audit records.

## Limits

AWS, Aliyun, and Tencent adapters are skeletons backed by deterministic inventory behavior. Real provider SDK integration, pagination, filtering, tagging, permission discovery, and cloud deployments are future work.
