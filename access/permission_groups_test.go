package access

import (
	"context"
	"testing"
)

func TestPermissionGroupClaimSourceMapsRolesToAccessGroups(t *testing.T) {
	source := PermissionGroupClaimSource{
		ClientID: "gateway",
		Manifest: Manifest{
			PermissionGroups: []PermissionGroup{
				{
					Code:         "client_operator",
					AccessGroups: []string{"page.clients", "client.card.editing"},
				},
				{
					Code:         "client_controller",
					AccessGroups: []string{"client.card.approve_actions"},
				},
			},
		},
	}
	groups, err := source.UserAccessGroups(context.Background(), "user-1", map[string]any{
		"resource_access": map[string]any{
			"gateway": map[string]any{
				"roles": []any{"client_operator"},
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
	for _, expected := range []string{"page.clients", "client.card.editing"} {
		if !got[expected] {
			t.Fatalf("missing access group %q in %#v", expected, got)
		}
	}
	if got["client.card.approve_actions"] {
		t.Fatalf("unexpected controller access group: %#v", got)
	}
}
