package inputs

// FormActionTrigger — события формы
type FormActionTrigger string

const (
	Init   FormActionTrigger = "init"
	Change FormActionTrigger = "change"
	Click  FormActionTrigger = "click"
)

// FormActionType — типы действий
type FormActionType string

const (
	APICall       FormActionType = "apiCall"
	ChangeAPICall FormActionType = "changeApiCall"
	Calculate     FormActionType = "calculate"
)

// FormActionConfig — конфигурация действия
type FormActionConfig struct {
	Type           FormActionType `json:"type"`
	URL            string         `json:"url,omitempty"`
	Method         string         `json:"method,omitempty"` // GET, POST, PUT, DELETE
	Formula        string         `json:"formula,omitempty"`
	SuccessMessage string         `json:"successMessage,omitempty"`
}

// FormAction — действие формы
type FormAction struct {
	ID      string            `json:"id"`
	Trigger FormActionTrigger `json:"trigger"`
	Config  *FormActionConfig `json:"config,omitempty"`
}
