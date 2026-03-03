package commands

type Registry struct {
	defs []Definition
}

// NewRegistry stores the canonical command set used by both dispatch and
// optional platform registration adapters.
func NewRegistry(defs []Definition) *Registry {
	return &Registry{defs: defs}
}

// Definitions returns all registered command definitions.
// Command availability is global and no longer channel-scoped.
func (r *Registry) Definitions() []Definition {
	out := make([]Definition, len(r.defs))
	copy(out, r.defs)
	return out
}
