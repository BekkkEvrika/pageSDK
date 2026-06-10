package page

import (
	"github.com/BekkkEvrika/pageSDK/engine"
)

// UsersListPage — page списка пользователей.
// Использует TableEngine: GET /page/users.list
type UsersListPage struct {
	*engine.TableEngine
}

// NewUsersListPage — фабрика для регистрации в Manifest.
func NewUsersListPage() engine.Page {
	return &UsersListPage{
		TableEngine: &engine.TableEngine{},
	}
}

// Init вызывается на каждый request и только собирает DSL.
func (p *UsersListPage) Init(ctx *engine.BuildContext) error {
	p.Column("id", "ID")
	p.Column("name", "Имя")
	p.Column("email", "Email")
	return nil
}
