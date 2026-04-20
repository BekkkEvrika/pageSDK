package domain

// EventType — тип события UI.
type EventType string

const (
	EventClick  EventType = "click"
	EventChange EventType = "change"
	EventInit   EventType = "init"
	EventBlur   EventType = "blur"
	EventSubmit EventType = "submit"
)

// UIEvent — входящее событие от фронтенда.
type UIEvent struct {
	SessionID   string                 `json:"sessionId"   validate:"required"`
	PageID      string                 `json:"pageId"      validate:"required"`
	ComponentID string                 `json:"componentId" validate:"required"`
	EventType   EventType              `json:"event"       validate:"required"`
	Payload     map[string]interface{} `json:"payload,omitempty"`
}
