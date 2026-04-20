package runtime

import (
	"fmt"
	"reflect"
	"strings"

	inputs "github.com/behzod/pageSDK/form"
)

// EvalRule вычисляет результат одного Rule относительно текущего state values.
func EvalRule(rule inputs.Rule, values map[string]interface{}) bool {
	val, exists := values[rule.Field]
	if !exists {
		val = nil
	}

	// Разрешаем valueRef
	expected := rule.Value
	if rule.ValueRef != "" {
		expected = values[rule.ValueRef]
	}

	switch rule.Operator {
	case inputs.OpEq:
		return fmt.Sprintf("%v", val) == fmt.Sprintf("%v", expected)
	case inputs.OpNeq:
		return fmt.Sprintf("%v", val) != fmt.Sprintf("%v", expected)
	case inputs.OpGt:
		return toFloat(val) > toFloat(expected)
	case inputs.OpGte:
		return toFloat(val) >= toFloat(expected)
	case inputs.OpLt:
		return toFloat(val) < toFloat(expected)
	case inputs.OpLte:
		return toFloat(val) <= toFloat(expected)
	case inputs.OpEmpty:
		return isEmpty(val)
	case inputs.OpNotEmpty:
		return !isEmpty(val)
	case inputs.OpIn:
		return containsValue(expected, val)
	case inputs.OpNotIn:
		return !containsValue(expected, val)
	case inputs.OpContains:
		return strings.Contains(fmt.Sprintf("%v", val), fmt.Sprintf("%v", expected))
	}
	return true
}

// EvalRules вычисляет список правил (все AND по умолчанию, если combine не задан).
func EvalRules(rules []inputs.Rule, values map[string]interface{}) bool {
	if len(rules) == 0 {
		return true
	}
	result := EvalRule(rules[0], values)
	for i := 1; i < len(rules); i++ {
		r := rules[i]
		combine := strings.ToLower(r.Combine)
		if combine == "or" {
			result = result || EvalRule(r, values)
		} else {
			result = result && EvalRule(r, values)
		}
	}
	return result
}

func isEmpty(v interface{}) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.String:
		return rv.String() == ""
	case reflect.Slice, reflect.Map, reflect.Array:
		return rv.Len() == 0
	}
	return false
}

func toFloat(v interface{}) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case float32:
		return float64(x)
	case int:
		return float64(x)
	case int64:
		return float64(x)
	}
	return 0
}

func containsValue(collection, item interface{}) bool {
	if collection == nil {
		return false
	}
	rv := reflect.ValueOf(collection)
	if rv.Kind() == reflect.Slice {
		for i := 0; i < rv.Len(); i++ {
			if fmt.Sprintf("%v", rv.Index(i).Interface()) == fmt.Sprintf("%v", item) {
				return true
			}
		}
	}
	return false
}
