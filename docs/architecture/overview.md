# Architecture Overview

Nivora is a Go modular monolith with separate binaries for the HTTP control plane, worker, runner, and CLI. The monolith keeps early development simple while preserving package boundaries that can later split into services.

The control plane owns API requests, persistence, audit, policy decisions, and orchestration state. Workers advance background workflow state. Runners execute assigned work and must not be treated as trusted control-plane components.

