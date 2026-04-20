package adapters

import (
	"context"
	"fmt"
	"sync"

	"github.com/behzod/pageSDK/internal/domain"
)

// InMemoryStateStore — in-memory реализация ports.StateStore.
type InMemoryStateStore struct {
	mu     sync.RWMutex
	states map[string]*domain.UIState
}

func NewInMemoryStateStore() *InMemoryStateStore {
	return &InMemoryStateStore{states: make(map[string]*domain.UIState)}
}

func (s *InMemoryStateStore) Get(_ context.Context, sessionID string) (*domain.UIState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	st, ok := s.states[sessionID]
	if !ok {
		return nil, nil // nil означает: состояния ещё нет
	}
	cp := *st
	return &cp, nil
}

func (s *InMemoryStateStore) Save(_ context.Context, sessionID string, state *domain.UIState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *state
	s.states[sessionID] = &cp
	return nil
}

func (s *InMemoryStateStore) Delete(_ context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.states[sessionID]; !ok {
		return fmt.Errorf("state not found: %s", sessionID)
	}
	delete(s.states, sessionID)
	return nil
}
