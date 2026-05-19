# Secret

A Secret is sensitive material such as a token, password, private key, kubeconfig, webhook secret, registry credential, or future signer/scanner secret.

Nivora does not expose secret values through normal APIs. Instead, it stores or resolves values through a `SecretProvider` and passes around `SecretRef` metadata.

In Phase 3.1 the builtin provider is in-memory and intended for development. Phase 7.1 adds rotation, provider validation, and adapter foundations for Vault, Kubernetes Secrets, and cloud KMS placeholders.

Production-grade encrypted persistence, real Vault/Kubernetes/KMS value storage, enterprise key lifecycle management, and provider-specific credential workflows remain future work.
