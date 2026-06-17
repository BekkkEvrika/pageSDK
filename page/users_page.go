package page

import (
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
		})

	return nil
}
