# Secret

A Secret is sensitive material such as a token, password, private key, kubeconfig, webhook secret, registry credential, or future signer/scanner secret.

Nivora does not expose secret values through normal APIs. Instead, it stores or resolves values through a `SecretProvider` and passes around `SecretRef` metadata.

In Phase 3.1 the builtin provider is in-memory and intended for development. Production-grade encrypted persistence, Vault, Kubernetes Secret, and cloud KMS integrations are future work.
