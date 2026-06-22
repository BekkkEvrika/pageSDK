package access

import (
	"strings"
	"testing"
)

func TestKey(t *testing.T) {
	if got := Key("clients", "event", "users.edit", "button", "save"); got != "clients.event.users.edit.button.save" {
		t.Fatalf("unexpected key %q", got)
	}
	if got := Key("", "page", "users"); got != "page.users" {
		t.Fatalf("unexpected key without module %q", got)
	}
}

func TestMergePreservesGroupsDescriptionsAndMarksStale(t *testing.T) {
	current := Manifest{
		Module:  "clients",
		Version: 1,
		Resources: []Resource{
			{Key: "clients.page.users", Description: "Custom users description"},
			{Key: "clients.page.old", Description: "Old"},
		},
		PermissionGroups: []PermissionGroup{
			{Key: "clients.viewer", Name: "Viewer", Permissions: []string{"clients.page.users"}},
		},
	}
	merged := Merge(current, "clients", []Resource{
		{Key: "clients.page.users", Type: "page", Page: "users", Description: "Generated"},
		{Key: "clients.page.new", Type: "page", Page: "new", Description: "Generated new"},
	})

	if len(merged.PermissionGroups) != 1 || merged.PermissionGroups[0].Key != "clients.viewer" {
		t.Fatalf("permission groups were not preserved: %#v", merged.PermissionGroups)
	}
	if len(merged.Resources) != 2 {
		t.Fatalf("expected two resources, got %#v", merged.Resources)
	}
	var users Resource
	for _, item := range merged.Resources {
		if item.Key == "clients.page.users" {
			users = item
		}
	}
	if users.Description != "Custom users description" {
		t.Fatalf("description was overwritten: %#v", users)
	}
	if len(merged.Stale) != 1 || merged.Stale[0].Key != "clients.page.old" {
		t.Fatalf("expected old resource to become stale, got %#v", merged.Stale)
	}
}

func TestValidate(t *testing.T) {
	valid := Manifest{
		Module:  "clients",
		Version: 1,
		Resources: []Resource{
			{Key: "clients.page.users"},
			{Key: "clients.event.users.button.save"},
		},
		PermissionGroups: []PermissionGroup{
			{Key: "clients.operator", Permissions: []string{"clients.event.users.button.*"}},
		},
	}
	if err := Validate(valid, "clients"); err != nil {
		t.Fatalf("expected valid manifest: %v", err)
	}

	valid.PermissionGroups[0].Permissions = []string{"clients.missing"}
	err := Validate(valid, "clients")
	if err == nil || !strings.Contains(err.Error(), "unknown permission") {
		t.Fatalf("expected unknown permission error, got %v", err)
	}
}

func TestCompare(t *testing.T) {
	diff := Compare(
		[]Resource{{Key: "clients.page.users"}, {Key: "clients.page.new"}},
		Manifest{
			Resources: []Resource{{Key: "clients.page.users"}, {Key: "clients.page.old"}},
			PermissionGroups: []PermissionGroup{
				{Key: "clients.viewer", Permissions: []string{"clients.page.old", "clients.page.missing"}},
			},
		},
	)
	if len(diff.NewInDSL) != 1 || diff.NewInDSL[0] != "clients.page.new" {
		t.Fatalf("unexpected new resources: %#v", diff.NewInDSL)
	}
	if len(diff.MissingInDSL) != 1 || diff.MissingInDSL[0] != "clients.page.old" {
		t.Fatalf("unexpected missing resources: %#v", diff.MissingInDSL)
	}
	if len(diff.BrokenGroupPermissions) != 2 ||
		diff.BrokenGroupPermissions[0] != "clients.viewer: clients.page.missing" ||
		diff.BrokenGroupPermissions[1] != "clients.viewer: clients.page.old" {
		t.Fatalf("unexpected broken permissions: %#v", diff.BrokenGroupPermissions)
	}
}
