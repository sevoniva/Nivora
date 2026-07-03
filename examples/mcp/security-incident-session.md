# MCP Example: Security Incident Summary

Safe local workflow:

1. Use `nivora_search_audit` with a narrow `subject`, `actorId`, or `action` filter.
2. Read `nivora://security/summary`.
3. Use the `audit_incident_summary` prompt.
4. If policy context is needed, use `policy_gate_review`.

Expected AI behavior:

- Group evidence by actor, action, subject, time, and decision.
- Redact secret-like content.
- Separate audit facts from incident hypotheses.
- List evidence gaps and safe read-only follow-up.

This example does not retrieve secret values or token hashes.
