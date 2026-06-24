package access

import (
	"context"
	"testing"
)

func TestDSLPermissionResolverAppliesNoAccessBehavior(t *testing.T) {
	resolver := DSLPermissionResolver{
		Authorizer: StaticAuthorizer{Groups: map[string][]string{
			"user-1": {"client.card.viewing"},
		}},
	}
	dsl := map[string]any{
		"fields": []any{
			map[string]any{
				"id":               "name",
				"accessGroupCode":  "client.card.editing",
				"noAccessBehavior": "readonly",
			},
			map[string]any{
				"id":               "delete",
				"accessGroupCode":  "client.card.delete_actions",
				"noAccessBehavior": "remove",
			},
			map[string]any{
				"id":              "view",
				"accessGroupCode": "client.card.viewing",
			},
		},
	}
	result, err := resolver.Apply(context.Background(), "user-1", nil, dsl)
	if err != nil {
		t.Fatal(err)
	}
	fields := result.(map[string]any)["fields"].([]any)
	if len(fields) != 2 {
		t.Fatalf("expected removed field to disappear, got %#v", fields)
	}
	name := fields[0].(map[string]any)
	if name["readonly"] != true {
		t.Fatalf("expected readonly behavior, got %#v", name)
	}
	view := fields[1].(map[string]any)
	if view["hidden"] == true || view["readonly"] == true || view["disabled"] == true {
		t.Fatalf("allowed field was modified: %#v", view)
	}
}
