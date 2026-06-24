package access

import "context"

type PermissionGroupClaimSource struct {
	Manifest Manifest
	ClientID string
}

func (s PermissionGroupClaimSource) UserAccessGroups(_ context.Context, _ string, user map[string]any) ([]string, error) {
	userRoles := rolesFromClaims(user, s.ClientID)
	groups := map[string]struct{}{}
	for _, permissionGroup := range s.Manifest.PermissionGroups {
		code := permissionGroupCode(permissionGroup)
		if _, ok := userRoles[code]; !ok {
			continue
		}
		for _, accessGroup := range permissionGroup.AccessGroups {
			groups[accessGroup] = struct{}{}
		}
	}
	result := make([]string, 0, len(groups))
	for group := range groups {
		result = append(result, group)
	}
	return result, nil
}

func rolesFromClaims(user map[string]any, clientID string) map[string]struct{} {
	roles := map[string]struct{}{}
	addRoles(roles, user["roles"])
	if realmAccess, ok := user["realm_access"].(map[string]any); ok {
		addRoles(roles, realmAccess["roles"])
	}
	if resourceAccess, ok := user["resource_access"].(map[string]any); ok && clientID != "" {
		if clientAccess, ok := resourceAccess[clientID].(map[string]any); ok {
			addRoles(roles, clientAccess["roles"])
		}
	}
	return roles
}

func addRoles(result map[string]struct{}, raw any) {
	switch values := raw.(type) {
	case []any:
		for _, item := range values {
			if role, ok := item.(string); ok && role != "" {
				result[role] = struct{}{}
			}
		}
	case []string:
		for _, role := range values {
			if role != "" {
				result[role] = struct{}{}
			}
		}
	}
}
