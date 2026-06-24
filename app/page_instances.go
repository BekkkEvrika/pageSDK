package app

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/BekkkEvrika/pageSDK/engine"
)

var (
	ErrPageInstanceNotFound = errors.New("page instance not found")
	ErrPageInstanceExpired  = errors.New("page instance expired")
	ErrPageInstanceLimit    = errors.New("page instance limit reached")
)

type pageInstance struct {
	ID         string
	PageKey    string
	OwnerID    string
	Page       engine.Page
	CreatedAt  time.Time
	LastAccess time.Time
	mu         sync.Mutex
}

type pageInstanceManager struct {
	mu        sync.Mutex
	instances map[string]*pageInstance
	ttl       time.Duration
	max       int
	now       func() time.Time
}

func newPageInstanceManager(ttl time.Duration, max int) *pageInstanceManager {
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}
	if max <= 0 {
		max = 10_000
	}
	return &pageInstanceManager{
		instances: make(map[string]*pageInstance),
		ttl:       ttl,
		max:       max,
		now:       time.Now,
	}
}

func (m *pageInstanceManager) NewID() (string, error) {
	value := make([]byte, 24)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("create page instance id: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func (m *pageInstanceManager) Add(id, pageKey, ownerID string, page engine.Page) error {
	now := m.now()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.removeExpiredLocked(now)
	if len(m.instances) >= m.max {
		return ErrPageInstanceLimit
	}
	m.instances[id] = &pageInstance{
		ID:         id,
		PageKey:    pageKey,
		OwnerID:    ownerID,
		Page:       page,
		CreatedAt:  now,
		LastAccess: now,
	}
	return nil
}

func (m *pageInstanceManager) Acquire(id, pageKey, ownerID string) (*pageInstance, error) {
	now := m.now()
	m.mu.Lock()
	instance, ok := m.instances[id]
	if !ok {
		m.mu.Unlock()
		return nil, ErrPageInstanceNotFound
	}
	if now.Sub(instance.LastAccess) > m.ttl {
		delete(m.instances, id)
		m.mu.Unlock()
		return nil, ErrPageInstanceExpired
	}
	if instance.PageKey != pageKey || instance.OwnerID != ownerID {
		m.mu.Unlock()
		return nil, ErrPageInstanceNotFound
	}
	instance.LastAccess = now
	m.mu.Unlock()

	instance.mu.Lock()
	return instance, nil
}

func (m *pageInstanceManager) Release(instance *pageInstance) {
	if instance != nil {
		instance.mu.Unlock()
	}
}

func (m *pageInstanceManager) Delete(id, pageKey, ownerID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	instance, ok := m.instances[id]
	if !ok || instance.PageKey != pageKey || instance.OwnerID != ownerID {
		return false
	}
	delete(m.instances, id)
	return true
}

func (m *pageInstanceManager) removeExpiredLocked(now time.Time) {
	for id, instance := range m.instances {
		if now.Sub(instance.LastAccess) > m.ttl {
			delete(m.instances, id)
		}
	}
}
