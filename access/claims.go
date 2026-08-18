package access

import (
	"context"
	"sort"
)

type RPTClaimSource struct{}

func (RPTClaimSource) UserAccessGroups(_ context.Context, _ string, user map[string]any) ([]string, error) {
	groups := map[string]struct{}{}
	authorization, _ := user["authorization"].(map[string]any)
	addPermissionAccessGroups(groups, authorization["permissions"])
	addPermissionAccessGroups(groups, user["permissions"])
	result := make([]string, 0, len(groups))
	for group := range groups {
		result = append(result, group)
	}
	sort.Strings(result)
	return result, nil
}

type JWTAuthorizationClaimSource = RPTClaimSource

func addPermissionAccessGroups(groups map[string]struct{}, raw any) {
	rawPermissions, _ := raw.([]any)
	for _, item := range rawPermissions {
		permission, ok := item.(map[string]any)
		if !ok {
			continue
		}
		for _, key := range []string{"rsname", "resource_set_name", "resource_name", "resource"} {
			if name, ok := permission[key].(string); ok && name != "" {
				groups[name] = struct{}{}
			}
		}
	}
}
