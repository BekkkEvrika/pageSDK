package page

import (
	"github.com/BekkkEvrika/pageSDK/access"
	"github.com/BekkkEvrika/pageSDK/engine"
	"github.com/BekkkEvrika/pageSDK/engine/tableengine"
)

// UsersListPage — page списка пользователей.
// Использует TableEngine: GET /page/users.list
type UsersListPage struct {
	*tableengine.TableEngine
}

// NewUsersListPage — фабрика для регистрации в Manifest.
func NewUsersListPage() engine.Page {
	return &UsersListPage{
		TableEngine: &tableengine.TableEngine{},
	}
}

// Init вызывается на каждый request и только собирает DSL.
func (p *UsersListPage) Init(ctx *engine.BuildContext) error {
	p.Table("users.simple").
		Access(UsersListViewing, access.NoAccessHidden).
		Columns(
			p.Column("id").Header("ID").Access(UsersListViewing, access.NoAccessHidden),
			p.Column("name").Header("Имя").Access(UsersListViewing, access.NoAccessHidden),
			p.Column("email").Header("Email").Access(UsersListViewing, access.NoAccessHidden),
		)
	return nil
}
