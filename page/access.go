package page

import "github.com/BekkkEvrika/pageSDK/access"

type AccessRegistry interface {
	RegisterAccessGroup(access.AccessGroup) error
}

var (
	UsersEditViewing = access.AccessGroup{
		Code: "users.edit.viewing",
		Name: "Просмотр формы пользователя",
	}
	UsersEditEditing = access.AccessGroup{
		Code: "users.edit.editing",
		Name: "Редактирование пользователя",
	}
	UsersEditPickerActions = access.AccessGroup{
		Code: "users.edit.picker_actions",
		Name: "Выбор пользователя из диалога",
	}
	UsersListViewing = access.AccessGroup{
		Code: "users.list.viewing",
		Name: "Просмотр списка пользователей",
	}
	UsersListActions = access.AccessGroup{
		Code: "users.list.actions",
		Name: "Действия списка пользователей",
	}
	UsersPickerActions = access.AccessGroup{
		Code: "users.picker.actions",
		Name: "Действия выбора пользователя",
	}
	AdminRolesViewing = access.AccessGroup{
		Code: "admin.roles.viewing",
		Name: "Просмотр ролей администратора",
	}
	CalculatorUsage = access.AccessGroup{
		Code: "calculator.usage",
		Name: "Использование калькулятора",
	}
)

func RegisterAccessGroups(registry AccessRegistry) error {
	for _, group := range []access.AccessGroup{
		UsersEditViewing,
		UsersEditEditing,
		UsersEditPickerActions,
		UsersListViewing,
		UsersListActions,
		UsersPickerActions,
		AdminRolesViewing,
		CalculatorUsage,
	} {
		if err := registry.RegisterAccessGroup(group); err != nil {
			return err
		}
	}
	return nil
}
