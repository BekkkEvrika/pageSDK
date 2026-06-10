package main

import (
	"github.com/behzod/pageSDK/app"
	"github.com/behzod/pageSDK/page"
)

func main() {
	// Bootstrap lifecycle:
	// 1. Application создаётся
	// 2. project.Initial(app) — регистрирует pages в manifest
	// 3. Bootstrap() — auto route generation + gin start

	application := app.New()

	application.Bootstrap(projectInitial, ":8080")
}

// projectInitial — точка входа проекта.
// Регистрирует все pages в manifest.
// Application НЕ знает о деталях pages — только ключи и фабрики.
func projectInitial(a *app.Application) {
	m := a.Manifest()

	//m.Register("users.list", page.NewUsersListPage)
	m.Register("users.edit", page.NewUsersEditPage)
	//m.Register("admin.roles", page.NewAdminRolesPage)
}
