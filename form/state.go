package inputs

import "encoding/json"

// FormState is the concrete runtime state a form page may receive from frontend.
// Field keys must match Input.Id from the Form DSL.
type FormState struct {
	Elements     []ElementState        `json:"elements,omitempty"`
	Sender       *ElementState         `json:"sender,omitempty"`
	Fields       map[string]FieldState `json:"fields,omitempty"`
	Form         *Form                 `json:"form,omitempty"`
	ActionID     string                `json:"actionId,omitempty"`
	Trigger      FormActionTrigger     `json:"trigger,omitempty"`
	ChangedField string                `json:"changedField,omitempty"`
}

// FieldState is kept for compatibility with the older fields map payload.
type FieldState = ElementState

// ElementState contains runtime value and metadata for one sent UI element.
type ElementState struct {
	Input
	Value any            `json:"value,omitempty"`
	Props map[string]any `json:"props,omitempty"`
}

// UnmarshalJSON accepts both known Input fields and arbitrary extra properties.
func (e *ElementState) UnmarshalJSON(data []byte) error {
	type elementAlias ElementState
	var decoded elementAlias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	for _, key := range knownElementJSONKeys() {
		delete(raw, key)
	}
	if len(raw) > 0 {
		if decoded.Props == nil {
			decoded.Props = map[string]any{}
		}
		for key, value := range raw {
			decoded.Props[key] = value
		}
	}

	*e = ElementState(decoded)
	return nil
}

func knownElementJSONKeys() []string {
	return []string{
		"id",
		"type",
		"name",
		"label",
		"actionId",
		"variant",
		"fromName",
		"readOnly",
		"placeholder",
		"validation",
		"metaData",
		"metaKey",
		"format",
		"options",
		"visibilityRules",
		"fieldActions",
		"fileConfig",
		"colSpan",
		"hint",
		"searchObject",
		"defaultValue",
		"searchSource",
		"dataType",
		"value",
		"props",
	}
}
