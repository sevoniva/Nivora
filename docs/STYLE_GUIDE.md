# Documentation Style Guide

Nivora documentation should help platform engineers, DevOps engineers, SREs, infrastructure architects, security reviewers, and contributors make accurate decisions.

## Language

English is the primary documentation language for open-source readability. Keep sentences direct and technical.

## Tone

Use precise infrastructure-engineering wording. Avoid marketing claims, exaggerated maturity, and vague phrases such as "enterprise-grade" unless backed by implemented capabilities.

## Current vs Target Wording

Separate current implementation from target architecture.

Use:

- "Phase 0 currently includes..."
- "Future phases should..."
- "The target architecture is..."

Avoid:

- "Nivora supports..." for unimplemented features.
- "Production-ready" in early phases.
- "Seamless" or "automatic" when behavior is not implemented.

## Describing Unimplemented Features

When a feature is planned but not implemented, say so explicitly. Mention the intended design and the phase where it may arrive.

Example:

> Kubernetes Job execution is a future Executor Adapter. Phase 0 only reserves the package boundary.

## Terminology

Use these terms consistently:

- Control Plane
- Execution Plane
- Runner
- Executor
- Pipeline
- PipelineRun
- Release
- DeploymentRun
- Artifact
- Artifact Registry
- Policy
- Audit
- Environment
- Release Target
- GitOps
- Adapter
- Port

## Diagrams

Use Mermaid for simple architecture diagrams. Keep diagrams readable in plain Markdown and avoid diagrams that imply unimplemented features are already complete.

## Linking

Prefer relative links within `docs/`. Link to the most authoritative source rather than repeating long explanations. Root-level documents should stay concise and point to detailed docs.
