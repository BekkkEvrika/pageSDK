package page

import (
	"github.com/behzod/pageSDK/engine"
)

// AdminRolesPage — page управления ролями.
// Использует TableEngine: GET /page/admin.roles
type AdminRolesPage struct {
	*engine.TableEngine
	dsl any
}

// NewAdminRolesPage — фабрика для регистрации в Manifest.
func NewAdminRolesPage() engine.Page {
	return &AdminRolesPage{
		TableEngine: &engine.TableEngine{},
	}
}

// Init вызывается на каждый request и только собирает DSL.
func (p *AdminRolesPage) Init(ctx *engine.BuildContext) error {
	p.dsl = map[string]any{
		"columns": []map[string]any{
			{"key": "id", "label": "ID"},
			{"key": "name", "label": "Роль"},
			{"key": "permissions", "label": "Права"},
		},
	}
	return nil
}

// DSL возвращает собранный declarative runtime model.
func (p *AdminRolesPage) DSL() any {
	return p.dsl
}
