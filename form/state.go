package inputs

// FormState is the concrete runtime state a form page may receive from frontend.
// Field keys must match Input.Id from the Form DSL.
type FormState struct {
	Fields       map[string]FieldState `json:"fields,omitempty"`
	ActionID     string                `json:"actionId,omitempty"`
	Trigger      FormActionTrigger     `json:"trigger,omitempty"`
	ChangedField string                `json:"changedField,omitempty"`
}

// FieldState contains runtime value for one Input.
type FieldState struct {
	Value any `json:"value,omitempty"`
}
