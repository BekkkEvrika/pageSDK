package adapters

import (
	"fmt"
	"sync"

	inputs "github.com/behzod/pageSDK/form"
)

// InMemoryPageRegistry — in-memory реализация ports.PageRegistry.
type InMemoryPageRegistry struct {
	mu    sync.RWMutex
	pages map[string]*inputs.Form
}

func NewInMemoryPageRegistry() *InMemoryPageRegistry {
	return &InMemoryPageRegistry{pages: make(map[string]*inputs.Form)}
}

func (r *InMemoryPageRegistry) Get(pageID string) (*inputs.Form, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	f, ok := r.pages[pageID]
	if !ok {
		return nil, fmt.Errorf("page not found: %s", pageID)
	}
	return f, nil
}

func (r *InMemoryPageRegistry) Register(pageID string, form *inputs.Form) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pages[pageID] = form
	return nil
}
