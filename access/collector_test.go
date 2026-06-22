package access_test

import (
	"testing"

	"github.com/BekkkEvrika/pageSDK/access"
	"github.com/BekkkEvrika/pageSDK/engine"
	"github.com/BekkkEvrika/pageSDK/engine/tableengine"
	"github.com/BekkkEvrika/pageSDK/manifest"
)

type collectorPage struct {
	*tableengine.TableEngine
}

func (p *collectorPage) Init(_ *engine.BuildContext) error {
	p.Table("users").
		Columns(p.Column("email")).
		OnReload(func(*tableengine.TableRuntimeContext) {}).
		ToolbarActions(p.Action("refresh", func(*tableengine.TableRuntimeContext) {}))
	return nil
}

func TestCollectTableAccessPoints(t *testing.T) {
	registry := manifest.New()
	registry.Register("users.list", func() engine.Page {
		return &collectorPage{TableEngine: &tableengine.TableEngine{}}
	})
	resources, err := access.Collect(registry, "clients")
	if err != nil {
		t.Fatal(err)
	}
	keys := map[string]bool{}
	for _, item := range resources {
		keys[item.Key] = true
	}
	expected := []string{
		"clients.page.users.list",
		"clients.event.users.list.table.users.reload",
		"clients.event.users.list.table.users.toolbar.refresh",
		"clients.ui.users.list.table.users.column.email.view",
	}
	for _, key := range expected {
		if !keys[key] {
			t.Fatalf("missing key %q in %#v", key, keys)
		}
	}
}
