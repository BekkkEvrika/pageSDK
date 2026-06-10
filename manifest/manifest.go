package manifest

import (
	"sort"

	"github.com/BekkkEvrika/pageSDK/engine"
)

// Entry — запись в манифесте для одной page.
type Entry struct {
	// Key — стабильный runtime identifier.
	// Примеры: "users.list", "users.edit", "admin.roles"
	// Используется для: routing, lookup, UMA/API Gateway.
	Key string

	// Factory создаёт новый stateless экземпляр Page на каждый request.
	Factory engine.PageFactory
}

// Manifest — реестр всех pages приложения.
// Заполняется один раз в project.Initial(app), не изменяется после Bootstrap.
type Manifest struct {
	entries map[string]Entry
}

// New создаёт пустой Manifest.
func New() *Manifest {
	return &Manifest{
		entries: make(map[string]Entry),
	}
}

// Register добавляет page в манифест по ключу.
// Паникует при дублировании ключа — ошибка конфигурации должна быть обнаружена при старте.
func (m *Manifest) Register(key string, factory engine.PageFactory) {
	if _, exists := m.entries[key]; exists {
		panic("manifest: duplicate page key: " + key)
	}
	m.entries[key] = Entry{Key: key, Factory: factory}
}

// Get возвращает Entry по ключу.
func (m *Manifest) Get(key string) (Entry, bool) {
	e, ok := m.entries[key]
	return e, ok
}

// All возвращает все записи манифеста.
// Используется Bootstrap для генерации routes.
func (m *Manifest) All() []Entry {
	keys := make([]string, 0, len(m.entries))
	for key := range m.entries {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	result := make([]Entry, 0, len(m.entries))
	for _, key := range keys {
		result = append(result, m.entries[key])
	}
	return result
}
