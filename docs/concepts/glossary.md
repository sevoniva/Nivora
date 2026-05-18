# Glossary

## Control Plane

The part of Nivora responsible for API, state, policy, audit, integration configuration, and orchestration decisions.

## Execution Plane

The part of Nivora responsible for executing assigned work through Runners and Executors.

## Port

An interface that defines an external capability needed by use cases.

## Adapter

An implementation of a Port for a specific system or local mechanism.

## Pipeline

A reusable delivery definition.

## PipelineRun

One execution of a Pipeline.

## Release

A versioned delivery intent, typically pointing to immutable Artifacts.

## DeploymentRun

One execution of a Release or deployment plan against an Environment or Release Target.

## Artifact

An immutable build output or package reference. Digests are preferred over mutable tags.

## Artifact Registry

A system that stores Artifacts, such as an OCI registry or package registry.

## Environment

A delivery context such as dev, staging, production, regional production, or a tenant-specific context.

## Release Target

A concrete target inside an Environment. It may be a host group, Kubernetes cluster, Argo CD application, cloud target, or webhook target.

## Runner

An Execution Plane process that receives work, sends heartbeats, runs jobs through Executors, streams logs, and reports status.

## Executor

A mechanism for running work, such as shell, SSH, Kubernetes Job, YAML apply, Helm, Argo CD, or webhook.

## Policy

An enforceable gate that evaluates whether an action may proceed.

## Audit

Durable accountability records for important delivery actions.

## GitOps

A deployment mode where desired state is represented in Git and reconciled by a GitOps system. It is one mode, not the whole product.

