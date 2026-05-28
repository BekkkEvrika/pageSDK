package page

import (
	"github.com/behzod/pageSDK/engine"
)

// AdminRolesPage — page управления ролями.
// Использует TableEngine: GET /page/admin.roles
type AdminRolesPage struct {
	*engine.TableEngine
}

// NewAdminRolesPage — фабрика для регистрации в Manifest.
func NewAdminRolesPage() engine.Page {
	return &AdminRolesPage{
		TableEngine: &engine.TableEngine{},
	}
}

// Init вызывается на каждый request и только собирает DSL.
func (p *AdminRolesPage) Init(ctx *engine.BuildContext) error {
	p.Column("id", "ID")
	p.Column("name", "Роль")
	p.Column("permissions", "Права")
	return nil
}
