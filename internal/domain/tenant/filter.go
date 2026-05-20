package tenant

// ListFilter provides optional scope filtering for list queries.
// Nil or zero-value means "no filter" (return all records).
type ListFilter struct {
	ScopeType string
	ScopeID   string
}

func (f ListFilter) IsZero() bool { return f.ScopeType == "" && f.ScopeID == "" }

// NewListFilter creates a ListFilter from scope parameters.
func NewListFilter(scopeType, scopeID string) ListFilter {
	return ListFilter{ScopeType: scopeType, ScopeID: scopeID}
}
