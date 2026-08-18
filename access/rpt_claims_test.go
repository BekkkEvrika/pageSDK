package access

import (
	"context"
	"testing"
)

func TestRPTClaimSourceMapsPermissionsToAccessGroups(t *testing.T) {
	source := RPTClaimSource{}
	groups, err := source.UserAccessGroups(context.Background(), "user-1", map[string]any{
		"authorization": map[string]any{
			"permissions": []any{
				map[string]any{"resource_set_name": "page.clients.card"},
				map[string]any{"rsname": "client.card.editing"},
				map[string]any{"resource_name": "client.card.passport_data"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, group := range groups {
		got[group] = true
	}
	for _, expected := range []string{"page.clients.card", "client.card.editing", "client.card.passport_data"} {
		if !got[expected] {
			t.Fatalf("missing access group %q in %#v", expected, got)
		}
	}
}

func TestRPTClaimSourceSupportsIntrospectionPermissionsShape(t *testing.T) {
	source := RPTClaimSource{}
	groups, err := source.UserAccessGroups(context.Background(), "user-1", map[string]any{
		"permissions": []any{
			map[string]any{"resource_name": "page.clients"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 || groups[0] != "page.clients" {
		t.Fatalf("unexpected access groups: %#v", groups)
	}
}
