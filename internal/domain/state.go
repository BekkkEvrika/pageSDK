package domain

import "time"

// UIState — runtime-состояние страницы для конкретной сессии.
type UIState struct {
	SessionID  string                 `json:"sessionId"`
	PageID     string                 `json:"pageId"`
	Values     map[string]interface{} `json:"values"`     // componentID → value
	Visibility map[string]bool        `json:"visibility"` // componentID → visible
	Disabled   map[string]bool        `json:"disabled"`   // componentID → disabled
	Errors     map[string][]string    `json:"errors"`     // componentID → errors
	Version    int64                  `json:"version"`
	UpdatedAt  time.Time              `json:"updatedAt"`
	Meta       map[string]interface{} `json:"meta,omitempty"`
}

// NewUIState создаёт пустое состояние.
func NewUIState(sessionID, pageID string) *UIState {
	return &UIState{
		SessionID:  sessionID,
		PageID:     pageID,
		Values:     make(map[string]interface{}),
		Visibility: make(map[string]bool),
		Disabled:   make(map[string]bool),
		Errors:     make(map[string][]string),
		Version:    1,
		UpdatedAt:  time.Now(),
	}
}

// StateDiff — изменения состояния, которые возвращаются фронтенду.
type StateDiff struct {
	ComponentID string                 `json:"componentId"`
	Changes     map[string]interface{} `json:"changes"`
}
