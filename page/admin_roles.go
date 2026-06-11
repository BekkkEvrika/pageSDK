package page

import (
	"github.com/BekkkEvrika/pageSDK/engine"
	"github.com/BekkkEvrika/pageSDK/engine/tableengine"
)

// AdminRolesPage — page управления ролями.
// Использует TableEngine: GET /page/admin.roles
type AdminRolesPage struct {
	*tableengine.TableEngine
}

// NewAdminRolesPage — фабрика для регистрации в Manifest.
func NewAdminRolesPage() engine.Page {
	return &AdminRolesPage{
		TableEngine: &tableengine.TableEngine{},
	}
}

// Init вызывается на каждый request и только собирает DSL.
func (p *AdminRolesPage) Init(ctx *engine.BuildContext) error {
	p.Column("id", "ID")
	p.Column("name", "Роль")
	p.Column("permissions", "Права")
	return nil
}
