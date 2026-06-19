package page

import (
	"fmt"
	"slices"
	"strings"

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
				Hidden(true).
				Hideable(false).
				Width(80),
			p.Column("name").
				Searchable(true).
				AddAction(onNormalizeUserNames, "normalize_names"),
			p.Column("email").
				Searchable(true).
				AddAction(onNormalizeUserEmails, "normalize_emails"),
			p.Column("status").
				CellType(tableengine.TableColumnCellTypeBadge).
				Filterable(true),
		).
		Data(usersData(0, 20)).
		Features(tableengine.TableFeatureConfig{
			Sorting:      true,
			Filtering:    true,
			Pagination:   true,
			RowSelection: true,
		}).
		Selection(tableengine.TableSelectionSchema{
			Mode:     tableengine.TableSelectionModeMultiple,
			Checkbox: true,
		}).
		OnReload(onUsersReload).
		OnFilter(onUsersFilter).
		OnPagination(onUsersPagination).
		ToolbarAction(tableengine.ActionSchema{
			ID:      "refresh",
			Label:   "Refresh",
			Icon:    "refresh",
			Variant: tableengine.ActionVariantSecondary,
			Hotkey:  "F5",
		}, onUsersRefresh).
		RowAction(tableengine.ActionSchema{
			ID:      "edit",
			Label:   "Edit",
			Icon:    "pencil",
			Variant: tableengine.ActionVariantSecondary,
		}, onUserEdit).
		SelectedAction(tableengine.ActionSchema{
			ID:      "delete_selected",
			Label:   "Delete Selected",
			Icon:    "trash",
			Variant: tableengine.ActionVariantDanger,
			Hotkey:  "Delete",
		}, onDeleteSelectedUsers)

	return nil
}

func onUsersRefresh(ctx *tableengine.TableRuntimeContext) {
	ctx.Table("users").SetData(usersData(0, 20))
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

func onNormalizeUserNames(ctx *tableengine.TableRuntimeContext) {
	names := ctx.EventTable.Column
	data := usersData(0, 20)
	for _, row := range data.Rows {
		rowID := fmt.Sprint(row["id"])
		if name, ok := names[rowID].(string); ok {
			row["name"] = strings.ToUpper(strings.TrimSpace(name))
		}
	}
	ctx.Table("users").SetData(data)
}

func onNormalizeUserEmails(ctx *tableengine.TableRuntimeContext) {
	emails := ctx.EventTable.Column
	data := usersData(0, 20)
	for _, row := range data.Rows {
		rowID := fmt.Sprint(row["id"])
		if email, ok := emails[rowID].(string); ok {
			row["email"] = strings.ToLower(strings.TrimSpace(email))
		}
	}
	ctx.Table("users").SetData(data)
}

func onDeleteSelectedUsers(ctx *tableengine.TableRuntimeContext) {
	data := usersData(0, 20)
	rows := make([]map[string]any, 0, len(data.Rows))
	for _, row := range data.Rows {
		if !slices.Contains(ctx.EventTable.SelectedRows, fmt.Sprint(row["id"])) {
			rows = append(rows, row)
		}
	}
	data.Rows = rows
	data.Total = len(rows)
	ctx.Table("users").SetData(data)
}

func usersData(pageIndex, pageSize int) tableengine.TableData {
	rows := []map[string]any{
		{"id": 1, "name": "Behzod", "email": "behzod@example.com", "status": "active"},
		{"id": 2, "name": "Ali", "email": "ali@example.com", "status": "inactive"},
		{"id": 3, "name": "Madina", "email": "madina@example.com", "status": "active"},
		{"id": 4, "name": "Farid", "email": "farid@example.com", "status": "pending"},
		{"id": 5, "name": "Zarina", "email": "zarina@example.com", "status": "active"},
		{"id": 6, "name": "Rustam", "email": "rustam@example.com", "status": "inactive"},
		{"id": 7, "name": "Nilufar", "email": "nilufar@example.com", "status": "active"},
		{"id": 8, "name": "Kamol", "email": "kamol@example.com", "status": "pending"},
		{"id": 9, "name": "Malika", "email": "malika@example.com", "status": "active"},
		{"id": 10, "name": "Iskandar", "email": "iskandar@example.com", "status": "inactive"},
		{"id": 11, "name": "Shabnam", "email": "shabnam@example.com", "status": "active"},
		{"id": 12, "name": "Jamshed", "email": "jamshed@example.com", "status": "pending"},
	}
	return tableengine.TableData{
		Rows:      rows,
		Total:     len(rows),
		PageIndex: pageIndex,
		PageSize:  pageSize,
	}
}
