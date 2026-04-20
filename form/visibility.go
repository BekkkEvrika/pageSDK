package inputs

type RuleOperator string

const (
	OpEq       RuleOperator = "eq"
	OpNeq      RuleOperator = "neq"
	OpGt       RuleOperator = "gt"
	OpGte      RuleOperator = "gte"
	OpLt       RuleOperator = "lt"
	OpLte      RuleOperator = "lte"
	OpIn       RuleOperator = "in"
	OpNotIn    RuleOperator = "notIn"
	OpEmpty    RuleOperator = "empty"
	OpNotEmpty RuleOperator = "notEmpty"
	OpContains RuleOperator = "contains"
)

type Rule struct {
	Field    string       `json:"field"`
	Operator RuleOperator `json:"operator"`
	Value    interface{}  `json:"value,omitempty"`
	ValueRef string       `json:"valueRef,omitempty"`
	Combine  string       `json:"combine,omitempty"` // "and" | "or"
}
