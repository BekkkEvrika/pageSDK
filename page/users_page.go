package page

import (
	"fmt"

	"github.com/BekkkEvrika/pageSDK/engine"
	"github.com/BekkkEvrika/pageSDK/engine/tableengine"
)

// UsersPage is a sample table page for testing TableEngine DSL generation.
type UsersPage struct {
	*tableengine.TableEngine
}

// NewUsersPage creates a fresh page instance for the manifest.
func NewUsersPage() engine.Page {
	return &UsersPage{
		TableEngine: &tableengine.TableEngine{},
	}
}

// Init builds only the table DSL schema.
func (p *UsersPage) Init(ctx *engine.BuildContext) error {
	p.Table("users").
		Columns(
			p.Column("id").
				DataType(tableengine.TableColumnDataTypeNumber).
				Width(80),
			p.Column("name").
				Searchable(true),
			p.Column("email").
				Searchable(true),
			p.Column("status").
				CellType(tableengine.TableColumnCellTypeBadge).
				Filterable(true),
		).
		Data([]map[string]any{
			{"id": 1, "name": "Behzod", "email": "behzod@example.com", "status": "active"},
			{"id": 2, "name": "Ali", "email": "ali@example.com", "status": "inactive"},
			{"id": 3, "name": "Madina", "email": "madina@example.com", "status": "active"},
		}).
		OnReload(onUsersReload).
		OnFilter(onUsersFilter).
		OnPagination(onUsersPagination).
		RowAction(tableengine.ActionSchema{
			ID:      "edit",
			Label:   "Edit",
			Icon:    "pencil",
			Variant: tableengine.ActionVariantSecondary,
		}, onUserEdit)

	return nil
}

func onUsersReload(ctx *tableengine.TableRuntimeContext) {
	ctx.Table("users").SetData(usersData(ctx.EventTable.PageIndex, ctx.EventTable.PageSize))
}

func onUsersFilter(ctx *tableengine.TableRuntimeContext) {
	ctx.Table("users").SetData(usersData(ctx.EventTable.PageIndex, ctx.EventTable.PageSize))
}

func onUsersPagination(ctx *tableengine.TableRuntimeContext) {
	ctx.Table("users").SetData(usersData(ctx.EventTable.PageIndex, ctx.EventTable.PageSize))
}

func onUserEdit(ctx *tableengine.TableRuntimeContext) {
	row := ctx.EventTable.Row
	ctx.OpenDialog("users.edit", engine.Params{
		"id":     fmt.Sprint(row["id"]),
		"name":   fmt.Sprint(row["name"]),
		"email":  fmt.Sprint(row["email"]),
		"status": fmt.Sprint(row["status"]),
	})
}

func usersData(pageIndex, pageSize int) tableengine.TableData {
	rows := []map[string]any{
		{"id": 1, "name": "Behzod", "email": "behzod@example.com", "status": "active"},
		{"id": 2, "name": "Ali", "email": "ali@example.com", "status": "inactive"},
		{"id": 3, "name": "Madina", "email": "madina@example.com", "status": "active"},
	}
	return tableengine.TableData{
		Rows:      rows,
		Total:     len(rows),
		PageIndex: pageIndex,
		PageSize:  pageSize,
	}
}
