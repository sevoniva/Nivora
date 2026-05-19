# KMS And External Secret Providers

Phase 7.1 adds external secret provider foundations for Kubernetes Secrets and cloud KMS-backed providers. These are adapter skeletons and placeholders, not complete production integrations.

## Provider Shapes

- `builtin`: in-memory development provider with create, retrieve, list, delete, rotate, usage audit, and validation support.
- `kubernetes`: Kubernetes Secret adapter foundation. It records namespace/context configuration shape but does not require a cluster.
- `aws-kms`: AWS KMS placeholder.
- `aliyun-kms`: Aliyun KMS placeholder.
- `tencent-kms`: Tencent Cloud KMS placeholder.

Cloud provider placeholders do not import cloud SDKs and do not require credentials. Future real adapters must resolve credentials through `SecretRef` or `CredentialRef` and must keep SDK usage inside adapter packages.

## Rotation

Rotation is modeled through the `SecretProvider` port. A successful rotation updates provider metadata such as version and emits audit/events, but the rotated value must not be logged, audited, or returned in API responses.

## Usage Policy

`SecretRef` metadata can carry simple policy hints:

- allowed use names
- allowed environments

These checks are intentionally small. They help prevent accidental use of a secret outside its intended purpose, but they are not a replacement for enterprise IAM or KMS policy.

## Security Rules

- Do not commit secret values, kubeconfigs, tokens, private keys, or realistic fake credentials.
- Do not log secret values.
- Do not return secret values from normal APIs.
- Use environment variable names and placeholders in examples.

Nivora remains early-stage and not production-ready.
