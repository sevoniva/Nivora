# Credential

A Credential is public metadata that tells Nivora how a future adapter should authenticate without embedding the sensitive value.

Examples include registry credentials, Argo CD credentials, kubeconfigs, SSH keys, webhook tokens, and cloud provider credentials. A Credential points to a `SecretRef`; it should never include the secret value itself.

Credential validation in Phase 3.1 only verifies that the referenced secret can be resolved by the configured provider. Vendor-specific validation remains adapter work for later phases.
