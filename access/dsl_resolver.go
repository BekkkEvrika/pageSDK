package access

import (
	"context"
	"encoding/json"
)

type DSLPermissionResolver struct {
	Authorizer AccessAuthorizer
}

func (r DSLPermissionResolver) Apply(ctx context.Context, userID string, user map[string]any, dsl any) (any, error) {
	if dsl == nil {
		return dsl, nil
	}
	var node any
	data, err := json.Marshal(dsl)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &node); err != nil {
		return nil, err
	}
	filtered, keep, err := r.applyNode(ctx, userID, user, node, map[string]bool{})
	if err != nil {
		return nil, err
	}
	if !keep {
		return nil, nil
	}
	return filtered, nil
}

func (r DSLPermissionResolver) applyNode(ctx context.Context, userID string, user map[string]any, node any, cache map[string]bool) (any, bool, error) {
	switch value := node.(type) {
	case []any:
		result := make([]any, 0, len(value))
		for _, item := range value {
			filtered, keep, err := r.applyNode(ctx, userID, user, item, cache)
			if err != nil {
				return nil, false, err
			}
			if keep {
				result = append(result, filtered)
			}
		}
		return result, true, nil
	case map[string]any:
		if groupCode, _ := value["accessGroupCode"].(string); groupCode != "" {
			if r.Authorizer != nil {
				allowed, ok := cache[groupCode]
				if !ok {
					var err error
					allowed, err = r.Authorizer.HasAccess(ctx, userID, user, groupCode)
					if err != nil {
						return nil, false, err
					}
					cache[groupCode] = allowed
				}
				if !allowed {
					behavior := NoAccessBehaviorFromAny(value["noAccessBehavior"])
					switch behavior {
					case NoAccessRemove:
						return nil, false, nil
					case NoAccessReadonly:
						value["readOnly"] = true
						value["readonly"] = true
					case NoAccessDisabled:
						value["disabled"] = true
					default:
						value["hidden"] = true
						value["visibility"] = false
					}
				}
			}
		}
		delete(value, "accessGroupCode")
		delete(value, "noAccessBehavior")
		for key, child := range value {
			filtered, keep, err := r.applyNode(ctx, userID, user, child, cache)
			if err != nil {
				return nil, false, err
			}
			if keep {
				value[key] = filtered
			} else {
				delete(value, key)
			}
		}
		return value, true, nil
	default:
		return node, true, nil
	}
}

func NoAccessBehaviorFromAny(value any) NoAccessBehavior {
	if text, ok := value.(string); ok {
		switch NoAccessBehavior(text) {
		case NoAccessReadonly:
			return NoAccessReadonly
		case NoAccessDisabled:
			return NoAccessDisabled
		case NoAccessRemove:
			return NoAccessRemove
		default:
			return NoAccessHidden
		}
	}
	return NoAccessHidden
}
