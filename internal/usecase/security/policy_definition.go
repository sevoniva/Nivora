package security

import (
	"strings"

	domainpolicy "github.com/sevoniva/nivora/internal/domain/policy"
)

func PolicyConfigFromDefinition(policy domainpolicy.Policy) PolicyConfig {
	return PolicyConfig{
		CriticalDenyThreshold: policy.CriticalDeny,
		HighWarnThreshold:     policy.HighWarn,
		RequireDigest:         policy.RequireDigest,
		ApprovalOnCritical:    policy.ApprovalOnCritical || strings.EqualFold(policy.Mode, "require_approval"),
	}
}

func ApplyPolicyDefinition(policy domainpolicy.Policy, input *ScanInput) {
	input.PolicyID = policy.ID
	input.PolicyMode = policy.Mode
	input.Policy = PolicyConfigFromDefinition(policy)
}

func ApplyPolicyDefinitionToEvaluation(policy domainpolicy.Policy, input *EvaluateInput) {
	input.PolicyID = policy.ID
	input.PolicyMode = policy.Mode
	input.Policy = PolicyConfigFromDefinition(policy)
}
