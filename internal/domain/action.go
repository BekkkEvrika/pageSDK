package domain

// ActionType — тип action-а в pipeline.
type ActionType string

const (
	ActionUpdateState ActionType = "updateState"
	ActionCallUseCase ActionType = "callUseCase"
	ActionNavigate    ActionType = "navigate"
	ActionValidate    ActionType = "validate"
	ActionSetValue    ActionType = "setValue"
	ActionSetVisible  ActionType = "setVisible"
	ActionSetDisabled ActionType = "setDisabled"
	ActionClearField  ActionType = "clearField"
)

// Action — декларативное действие, привязанное к событию.
type Action struct {
	Type        ActionType             `json:"type"`
	TargetID    string                 `json:"targetId,omitempty"`
	Params      map[string]interface{} `json:"params,omitempty"`
	UseCaseName string                 `json:"useCaseName,omitempty"`
	NavigateTo  string                 `json:"navigateTo,omitempty"`
}

// ActionResult — результат выполнения action-а.
type ActionResult struct {
	Diffs      []StateDiff            `json:"diffs,omitempty"`
	NavigateTo string                 `json:"navigateTo,omitempty"`
	Errors     []AppError             `json:"errors,omitempty"`
	Meta       map[string]interface{} `json:"meta,omitempty"`
}

// AppError — стандартная ошибка приложения.
type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// EventBinding — связь события компонента со списком action-ов (runtime-регистрация).
type EventBinding struct {
	ComponentID string
	EventType   EventType
	Actions     []Action
}
