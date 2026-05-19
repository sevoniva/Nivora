# Plugin RFC Template

Use this template for plugin system changes, external adapter protocols, plugin execution, or marketplace-like behavior.

## Title

## Status

Proposed

## Author

## Created

YYYY-MM-DD

## Summary

## Problem

## Goals

## Non-Goals

## Plugin Type

One of:

- scm
- artifact
- cloud
- executor
- secret
- notification
- policy
- scanner
- gitops

## Capabilities

List capability names, inputs, outputs, and failure behavior.

## Protocol

Describe HTTP, gRPC, or another explicit process boundary. Do not propose unsafe dynamic loading.

## Authentication and Authorization

Explain how the plugin authenticates to Nivora and how Nivora authorizes plugin actions.

## Secret Access

Explain SecretRef or CredentialRef usage. Secret values must not appear in manifests, logs, examples, or audit records.

## Runtime Behavior

Cover timeouts, retries, cancellation, logging, events, and audit.

## Compatibility

Describe version negotiation and how old plugin versions are handled.

## Security Impact

Cover sandboxing, endpoint trust, credential exposure, and supply chain risks.

## Operational Impact

Cover health checks, diagnostics, metrics, and failure modes.

## Alternatives Considered

## Rollout Plan

## Open Questions
