// Package ssh contains a guarded SSH executor foundation for host deployments.
//
// Remote SSH is disabled unless a caller provides an explicit runner transport
// and the deployment request includes apply confirmation, allowRemote, and a
// CredentialRef. Tests use fake runner transports; the default constructor does
// not open network connections.
package ssh
