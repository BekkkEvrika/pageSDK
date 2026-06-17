package tableengine

import (
	"testing"

	"github.com/BekkkEvrika/pageSDK/table"
)

func TestColumnBuildsPackageTableSchema(t *testing.T) {
	engine := &TableEngine{}

	engine.Columns(
		engine.Column("email").
			Header("Email").
			AccessorKey("email").
			Sortable(true).
			Searchable(true),
	)

	dsl, ok := engine.DSL().(table.TableSchema)
	if !ok {
		t.Fatalf("DSL type = %T, want table.TableSchema", engine.DSL())
	}
	if len(dsl.Columns) != 1 {
		t.Fatalf("columns len = %d, want 1", len(dsl.Columns))
	}

	got := dsl.Columns[0]
	if got.ID != "email" {
		t.Fatalf("column ID = %q, want email", got.ID)
	}
	if got.Header != "Email" {
		t.Fatalf("column Header = %q, want Email", got.Header)
	}
	if got.AccessorKey != "email" {
		t.Fatalf("column AccessorKey = %q, want email", got.AccessorKey)
	}
	if got.Kind != table.TableColumnKindAccessor {
		t.Fatalf("column Kind = %q, want accessor", got.Kind)
	}
	if !got.Sortable || !got.Searchable {
		t.Fatalf("column flags not preserved: sortable=%v searchable=%v", got.Sortable, got.Searchable)
	}
}

func TestTableEngineSupportsRowsAndActions(t *testing.T) {
	engine := &TableEngine{}

	engine.Table("users").Title("Users")
	engine.Columns(engine.Column("id").Header("ID").DataType(table.TableColumnDataTypeNumber))
	engine.SetRows([]map[string]any{
		{"id": 1},
		{"id": 2},
	})
	engine.AddToolbarAction(table.ActionSchema{
		ID:      "refresh",
		Label:   "Refresh",
		Method:  table.HTTPMethodPOST,
		Variant: table.ActionVariantSecondary,
	})

	dsl := engine.DSL().(table.TableSchema)
	if dsl.Title != "Users" {
		t.Fatalf("title = %q, want Users", dsl.Title)
	}
	if dsl.Data == nil || dsl.Data.Total != 2 {
		t.Fatalf("data = %#v, want total 2", dsl.Data)
	}
	if dsl.Actions == nil || len(dsl.Actions.Toolbar) != 1 {
		t.Fatalf("toolbar actions = %#v, want one action", dsl.Actions)
	}
}

func TestColumnWithHeaderAppendsForCompatibility(t *testing.T) {
	engine := &TableEngine{}

	engine.Column("id", "ID")
	engine.Column("name", "Name").Searchable(true)
	engine.Column("email", "Email")

	dsl := engine.DSL().(table.TableSchema)
	if len(dsl.Columns) != 3 {
		t.Fatalf("columns len = %d, want 3", len(dsl.Columns))
	}
	if dsl.Columns[0].ID != "id" || dsl.Columns[1].ID != "name" || dsl.Columns[2].ID != "email" {
		t.Fatalf("unexpected columns: %#v", dsl.Columns)
	}
	if !dsl.Columns[1].Searchable {
		t.Fatalf("returned compatibility builder did not mutate stored column: %#v", dsl.Columns[1])
	}
}

func TestTableBuilderFluentAPI(t *testing.T) {
	engine := &TableEngine{}

	engine.Table("users").
		Title("Users").
		RequestURL("/api/users").
		RowIDKey("id").
		Columns(
			engine.Column("id").Header("ID").AccessorKey("id"),
			engine.Column("name").Header("Name").AccessorKey("name").Searchable(true),
			engine.Column("status").Header("Status").AccessorKey("status").CellType(table.TableColumnCellTypeBadge),
		).
		Features(table.TableFeatureConfig{
			Sorting:      true,
			Filtering:    true,
			Pagination:   true,
			RowSelection: true,
		}).
		Selection(table.TableSelectionSchema{
			Mode:     table.TableSelectionModeMultiple,
			Checkbox: true,
		})

	dsl := engine.DSL().(table.TableSchema)
	if dsl.Title != "Users" || dsl.RequestURL != "/api/users" || dsl.RowIDKey != "id" {
		t.Fatalf("unexpected table metadata: %#v", dsl)
	}
	if len(dsl.Columns) != 3 {
		t.Fatalf("columns len = %d, want 3", len(dsl.Columns))
	}
	if !dsl.Columns[1].Searchable {
		t.Fatalf("name column must be searchable: %#v", dsl.Columns[1])
	}
	if dsl.Columns[2].CellType != table.TableColumnCellTypeBadge {
		t.Fatalf("status cell type = %q, want badge", dsl.Columns[2].CellType)
	}
	if dsl.Features == nil || !dsl.Features.Sorting || !dsl.Features.Pagination {
		t.Fatalf("features not preserved: %#v", dsl.Features)
	}
	if dsl.Selection == nil || dsl.Selection.Mode != table.TableSelectionModeMultiple || !dsl.Selection.Checkbox {
		t.Fatalf("selection not preserved: %#v", dsl.Selection)
	}
}

func TestTableBuilderAppliesDefaultsAndSimpleData(t *testing.T) {
	engine := &TableEngine{}

	engine.Table("users").
		Columns(
			engine.Column("id"),
			engine.Column("name").Searchable(true),
			engine.Column("status").CellType(table.TableColumnCellTypeBadge),
		).
		Data([]map[string]any{
			{"id": 1, "name": "Behzod", "status": "active"},
			{"id": 2, "name": "Ali", "status": "inactive"},
		})

	dsl := engine.DSL().(table.TableSchema)
	if dsl.Title != "Users" {
		t.Fatalf("title = %q, want Users", dsl.Title)
	}
	if dsl.RequestURL != "/api/users" {
		t.Fatalf("request URL = %q, want /api/users", dsl.RequestURL)
	}
	if dsl.RowIDKey != "id" {
		t.Fatalf("row ID key = %q, want id", dsl.RowIDKey)
	}
	if dsl.Features == nil || !dsl.Features.Sorting || !dsl.Features.Filtering || !dsl.Features.GlobalSearch || !dsl.Features.Pagination {
		t.Fatalf("default features not applied: %#v", dsl.Features)
	}
	if dsl.State == nil || dsl.State.PageSize != 20 {
		t.Fatalf("default state not applied: %#v", dsl.State)
	}
	if dsl.Data == nil || dsl.Data.Total != 2 {
		t.Fatalf("data = %#v, want total 2", dsl.Data)
	}
	if dsl.Columns[0].Header != "id" || dsl.Columns[0].AccessorKey != "id" {
		t.Fatalf("column defaults not applied: %#v", dsl.Columns[0])
	}
}
