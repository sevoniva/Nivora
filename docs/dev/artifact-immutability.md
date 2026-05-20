# Artifact Immutability Enforcement

Nivora enforces artifact immutability through the `ImmutabilityPolicy` in the artifact domain. The policy gates artifact references used in releases and deployments.

## Quick Start

```bash
# Inspect an artifact reference
nivora artifact inspect registry.example.com/team/app@sha256:abc123...

# Check immutability status
nivora artifact inspect registry.example.com/team/app:latest
```

## Policy Rules

| Rule | Production Default | Dev Default | Behavior |
|---|---|---|---|
| `denyLatestTag` | true | false | Rejects `:latest` and tagless references |
| `requireDigest` | true | false | Requires `@sha256:...` digest pinning |
| `warnOnLatest` | true | true | Warns on mutable tag usage |
| `warnOnMissing` | true | true | Warns when digest is not pinned |

## Evaluation

The `ImmutabilityPolicy.Evaluate()` method takes one or more artifact references and returns an `ImmutabilityResult`:

```go
type ImmutabilityResult struct {
    Allowed        bool     // Whether the reference passes the policy
    Immutable      bool     // Whether the reference is inherently immutable
    IsDigestPinned bool     // Whether the reference uses a canonical digest
    Digest         string   // The digest if present
    Tag            string   // The tag if present
    Warnings       []string // Non-blocking warnings
    Denials        []string // Blocking denials
    OverrideReason string   // Reason for override (audited)
}
```

## Override

When an override reason is provided (e.g., "emergency hotfix approved by security"), `denyLatestTag` and `requireDigest` denials are bypassed. The override reason is recorded in the result and should be audited.

```go
result := policy.Evaluate(refs, "emergency hotfix approved by security")
// result.Allowed == true even for :latest tag
// result.OverrideReason == "emergency hotfix approved by security"
```

## Integration

The immutability policy is used by:

- **Artifact Service** (`CreateRelease`) — validates artifact references when creating releases
- **Deployment Safety Policy** (`K8sSafetyPolicy`) — validates container images in Kubernetes manifests
- **GitOps Plan** — validates artifact references in GitOps deployment plans

## Examples

### Accepted: Digest-pinned reference
```
registry.example.com/app@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
```
Result: Allowed, Immutable=true, IsDigestPinned=true

### Denied: Latest tag (production)
```
registry.example.com/app:latest
```
Result: Denied — "latest/mutable tag is denied by immutability policy"

### Denied: Tag without digest (production)
```
registry.example.com/app:1.0.0
```
Result: Denied — "digest is required by immutability policy"

### Allowed: Override
```
registry.example.com/app:latest
```
With override reason: "emergency hotfix"
Result: Allowed, OverrideReason recorded

## Current Limitations

- OCI digest resolution is a foundation/skeleton adapter — no real registry API calls.
- Digest is parsed from the reference string, not verified against a registry.
- No signature/cosign verification (future DevSecOps hardening).
- No Harbor/Nexus/JFrog management API integration.
- Immutability is enforced at the policy level, not at the registry level.
