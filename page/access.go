package page

import "github.com/BekkkEvrika/pageSDK/access"

type AccessRegistry interface {
	RegisterAccessGroup(access.AccessGroup) error
}

var (
	UsersEditViewing = access.AccessGroup{
		Code:       "users.edit.viewing",
		Name:       "Просмотр формы пользователя",
		Type:       access.AccessGroupUI,
		ParentCode: access.PageAccessGroupCode("users.edit"),
		Enabled:    true,
	}
	UsersEditEditing = access.AccessGroup{
		Code:       "users.edit.editing",
		Name:       "Редактирование пользователя",
		Type:       access.AccessGroupUI,
		ParentCode: access.PageAccessGroupCode("users.edit"),
		Enabled:    true,
	}
	UsersEditPickerActions = access.AccessGroup{
		Code:       "users.edit.picker_actions",
		Name:       "Выбор пользователя из диалога",
		Type:       access.AccessGroupAction,
		ParentCode: access.PageAccessGroupCode("users.edit"),
		Enabled:    true,
	}
	UsersListViewing = access.AccessGroup{
		Code:       "users.list.viewing",
		Name:       "Просмотр списка пользователей",
		Type:       access.AccessGroupUI,
		ParentCode: access.PageAccessGroupCode("users.list"),
		Enabled:    true,
	}
	UsersListActions = access.AccessGroup{
		Code:       "users.list.actions",
		Name:       "Действия списка пользователей",
		Type:       access.AccessGroupAction,
		ParentCode: access.PageAccessGroupCode("users.list"),
		Enabled:    true,
	}
	UsersPickerActions = access.AccessGroup{
		Code:       "users.picker.actions",
		Name:       "Действия выбора пользователя",
		Type:       access.AccessGroupAction,
		ParentCode: access.PageAccessGroupCode("users.picker"),
		Enabled:    true,
	}
	AdminRolesViewing = access.AccessGroup{
		Code:       "admin.roles.viewing",
		Name:       "Просмотр ролей администратора",
		Type:       access.AccessGroupUI,
		ParentCode: access.PageAccessGroupCode("admin.roles"),
		Enabled:    true,
	}
	CalculatorUsage = access.AccessGroup{
		Code:       "calculator.usage",
		Name:       "Использование калькулятора",
		Type:       access.AccessGroupAction,
		ParentCode: access.PageAccessGroupCode("calculator"),
		Enabled:    true,
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
