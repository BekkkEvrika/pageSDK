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
	if name["readOnly"] != true || name["readonly"] != true {
		t.Fatalf("expected readonly behavior, got %#v", name)
	}
	if name["accessGroupCode"] != nil || name["noAccessBehavior"] != nil {
		t.Fatalf("access metadata leaked to denied field: %#v", name)
	}
	view := fields[1].(map[string]any)
	if view["hidden"] == true || view["visibility"] == false || view["readOnly"] == true || view["readonly"] == true || view["disabled"] == true {
		t.Fatalf("allowed field was modified: %#v", view)
	}
	if view["accessGroupCode"] != nil || view["noAccessBehavior"] != nil {
		t.Fatalf("access metadata leaked to allowed field: %#v", view)
	}
}

func TestDSLPermissionResolverAppliesHiddenVisibility(t *testing.T) {
	resolver := DSLPermissionResolver{
		Authorizer: StaticAuthorizer{Groups: map[string][]string{}},
	}
	dsl := map[string]any{
		"fields": []any{
			map[string]any{
				"id":               "save",
				"accessGroupCode":  "client.card.editing",
				"noAccessBehavior": "hidden",
			},
		},
	}
	result, err := resolver.Apply(context.Background(), "user-1", nil, dsl)
	if err != nil {
		t.Fatal(err)
	}
	fields := result.(map[string]any)["fields"].([]any)
	save := fields[0].(map[string]any)
	if save["hidden"] != true || save["visibility"] != false {
		t.Fatalf("expected hidden behavior to hide by both keys, got %#v", save)
	}
	if save["accessGroupCode"] != nil || save["noAccessBehavior"] != nil {
		t.Fatalf("access metadata leaked to hidden field: %#v", save)
	}
}

func TestDSLPermissionResolverStripsAccessMetadataWithoutAuthorizer(t *testing.T) {
	resolver := DSLPermissionResolver{}
	dsl := map[string]any{
		"fields": []any{
			map[string]any{
				"id":               "save",
				"accessGroupCode":  "client.card.editing",
				"noAccessBehavior": "hidden",
			},
		},
	}
	result, err := resolver.Apply(context.Background(), "", nil, dsl)
	if err != nil {
		t.Fatal(err)
	}
	fields := result.(map[string]any)["fields"].([]any)
	save := fields[0].(map[string]any)
	if save["accessGroupCode"] != nil || save["noAccessBehavior"] != nil {
		t.Fatalf("access metadata leaked without authorizer: %#v", save)
	}
	if save["hidden"] == true || save["visibility"] == false {
		t.Fatalf("field was modified without authorizer: %#v", save)
	}
}
