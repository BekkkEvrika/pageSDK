package page

import (
	"github.com/behzod/pageSDK/engine"
)

// UsersListPage — page списка пользователей.
// Использует TableEngine: GET /page/users.list
type UsersListPage struct {
	*engine.TableEngine
	dsl any
}

// NewUsersListPage — фабрика для регистрации в Manifest.
func NewUsersListPage() engine.Page {
	return &UsersListPage{
		TableEngine: &engine.TableEngine{},
	}
}

// Init вызывается на каждый request и только собирает DSL.
func (p *UsersListPage) Init(ctx *engine.BuildContext) error {
	p.dsl = map[string]any{
		"columns": []map[string]any{
			{"key": "id", "label": "ID"},
			{"key": "name", "label": "Имя"},
			{"key": "email", "label": "Email"},
		},
	}
	return nil
}

// DSL возвращает собранный declarative runtime model.
func (p *UsersListPage) DSL() any {
	return p.dsl
}
