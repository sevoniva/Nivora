package artifact

// ImmutabilityPolicy defines rules for artifact reference immutability.
type ImmutabilityPolicy struct {
	DenyLatestTag bool `json:"denyLatestTag" yaml:"denyLatestTag"`
	RequireDigest bool `json:"requireDigest" yaml:"requireDigest"`
	WarnOnLatest  bool `json:"warnOnLatest" yaml:"warnOnLatest"`
	WarnOnMissing bool `json:"warnOnMissing" yaml:"warnOnMissing"`
}

// DefaultImmutabilityPolicy returns a safe-by-default policy for production.
func DefaultImmutabilityPolicy() ImmutabilityPolicy {
	return ImmutabilityPolicy{
		DenyLatestTag: true,
		RequireDigest: true,
		WarnOnLatest:  true,
		WarnOnMissing: true,
	}
}

// DevImmutabilityPolicy returns a relaxed policy for development.
func DevImmutabilityPolicy() ImmutabilityPolicy {
	return ImmutabilityPolicy{
		DenyLatestTag: false,
		RequireDigest: false,
		WarnOnLatest:  true,
		WarnOnMissing: true,
	}
}

// ImmutabilityResult is the result of evaluating artifact immutability.
type ImmutabilityResult struct {
	Allowed        bool     `json:"allowed"`
	Immutable      bool     `json:"immutable"`
	IsDigestPinned bool     `json:"isDigestPinned"`
	Digest         string   `json:"digest,omitempty"`
	Tag            string   `json:"tag,omitempty"`
	Warnings       []string `json:"warnings,omitempty"`
	Denials        []string `json:"denials,omitempty"`
	OverrideReason string   `json:"overrideReason,omitempty"`
}

// Evaluate evaluates artifact references against this policy.
func (p ImmutabilityPolicy) Evaluate(refs []Reference, overrideReason string) ImmutabilityResult {
	result := ImmutabilityResult{Allowed: true}
	if overrideReason != "" {
		result.OverrideReason = overrideReason
	}

	if len(refs) == 0 {
		result.Warnings = append(result.Warnings, "no artifact references provided")
		return result
	}

	ref := refs[0] // Primary reference.
	result.Immutable = ref.Immutable
	result.IsDigestPinned = ref.IsDigestPinned
	result.Digest = ref.Digest
	result.Tag = ref.Tag

	// Check canonical SHA256 digest.
	if IsCanonicalSHA256Digest(ref.Digest) {
		result.IsDigestPinned = true
		result.Immutable = true
	}

	// Latest tag check.
	if ref.Tag == "latest" || (ref.Tag == "" && ref.Digest == "") {
		if p.DenyLatestTag && overrideReason == "" {
			result.Allowed = false
			result.Denials = append(result.Denials, "latest/mutable tag is denied by immutability policy")
		} else if p.WarnOnLatest {
			result.Warnings = append(result.Warnings, "artifact uses latest/mutable tag; prefer digest reference")
		}
	}

	// Digest required check.
	if p.RequireDigest && !result.IsDigestPinned && overrideReason == "" {
		result.Allowed = false
		result.Denials = append(result.Denials, "digest is required by immutability policy")
	} else if !result.IsDigestPinned && p.WarnOnMissing {
		result.Warnings = append(result.Warnings, "artifact reference does not use a pinned digest")
	}

	return result
}
