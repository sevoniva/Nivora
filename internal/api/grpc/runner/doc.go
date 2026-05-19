// Package runner contains runner gRPC API scaffolding.
//
// Phase 3.6 uses HTTP runner protocol endpoints first. A future gRPC protocol
// can mirror register, heartbeat, claim, log append, status update, and cancel
// request semantics after the HTTP shape stabilizes.
package runner
