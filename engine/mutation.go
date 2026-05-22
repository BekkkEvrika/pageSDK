package engine

// MutationType enumerates explicit runtime mutations.
type MutationType string

const (
	MutationUpdate MutationType = "update"
	MutationAdd    MutationType = "add"
	MutationRemove MutationType = "remove"
)

// Mutation is an explicit frontend runtime change.
type Mutation struct {
	Type   MutationType `json:"type"`
	Target string       `json:"target"`
	Value  any          `json:"value,omitempty"`
}

// MutationWriter collects explicit runtime mutations.
type MutationWriter struct {
	items []Mutation
}

// Update records a control/node update.
func (w *MutationWriter) Update(target string, value any) {
	w.items = append(w.items, Mutation{Type: MutationUpdate, Target: target, Value: value})
}

// Add records a node add mutation.
func (w *MutationWriter) Add(target string, value any) {
	w.items = append(w.items, Mutation{Type: MutationAdd, Target: target, Value: value})
}

// Remove records a node remove mutation.
func (w *MutationWriter) Remove(target string) {
	w.items = append(w.items, Mutation{Type: MutationRemove, Target: target})
}

// Items returns collected mutations.
func (w *MutationWriter) Items() []Mutation {
	return append([]Mutation(nil), w.items...)
}
