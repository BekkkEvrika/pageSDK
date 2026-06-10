package main

import (
	pagesdk "github.com/BekkkEvrika/pageSDK"
	"github.com/BekkkEvrika/pageSDK/page"
)

func main() {
	application := pagesdk.New()

	if err := application.Bootstrap(projectInitial, ":8080"); err != nil {
		panic(err)
	}
}

// projectInitial is an example project entry point.
// It registers all pages in the application manifest.
func projectInitial(a *pagesdk.Application) {
	m := a.Manifest()

	// m.Register("users.list", page.NewUsersListPage)
	m.Register("users.edit", page.NewUsersEditPage)
	// m.Register("admin.roles", page.NewAdminRolesPage)
}
