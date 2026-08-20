package table

import (
	"testing"

	"github.com/BekkkEvrika/pageSDK/engine"
)

func TestOpenDialogKeepsEngineParamsCompatibility(t *testing.T) {
	ctx := &TableRuntimeContext{}
	params := engine.Params{"id": "7"}

	ctx.OpenDialog("users.edit", params)

	if err := ctx.Error(); err != nil {
		t.Fatal(err)
	}
	if len(ctx.Navigation) != 1 {
		t.Fatalf("navigation actions = %d, want 1", len(ctx.Navigation))
	}
	action := ctx.Navigation[0]
	if action.Page != "users.edit" || action.Mode != engine.NavigationModeDialog || action.Extra["id"] != "7" {
		t.Fatalf("unexpected navigation action: %#v", action)
	}
	params["id"] = "8"
	if action.Extra["id"] != "7" {
		t.Fatalf("navigation extra was not copied: %#v", action.Extra)
	}
}
