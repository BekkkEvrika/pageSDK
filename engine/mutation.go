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
	Type  MutationType `json:"type"`
	Path  string       `json:"path"`
	Value any          `json:"value,omitempty"`
}

// MutationWriter collects explicit runtime mutations.
type MutationWriter struct {
	items []Mutation
}

// Update records a control/node update.
func (w *MutationWriter) Update(path string, value any) {
	w.items = append(w.items, Mutation{Type: MutationUpdate, Path: path, Value: value})
}

// Add records a node add mutation.
func (w *MutationWriter) Add(path string, value any) {
	w.items = append(w.items, Mutation{Type: MutationAdd, Path: path, Value: value})
}

// Remove records a node remove mutation.
func (w *MutationWriter) Remove(path string) {
	w.items = append(w.items, Mutation{Type: MutationRemove, Path: path})
}

// Items returns collected mutations.
func (w *MutationWriter) Items() []Mutation {
	return append([]Mutation(nil), w.items...)
}
