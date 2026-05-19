DROP INDEX IF EXISTS idx_compliance_audit_scope;
DROP INDEX IF EXISTS idx_compliance_audit_subject;
DROP INDEX IF EXISTS idx_compliance_retention_scope;
DROP INDEX IF EXISTS idx_compliance_evidence_scope;
DROP INDEX IF EXISTS idx_compliance_evidence_subject;

DROP TABLE IF EXISTS compliance_audit_records;
DROP TABLE IF EXISTS compliance_retention_policies;
DROP TABLE IF EXISTS compliance_evidence_bundles;
