package page

import (
	"errors"
	"strings"

	"github.com/BekkkEvrika/pageSDK/engine"
	"github.com/BekkkEvrika/pageSDK/engine/formengine"
)

// UsersEditPage — page редактирования пользователя.
// Stateless: создаётся на каждый request, уничтожается после ответа.
//
// Embedding *formengine.FormEngine даёт:
//   - реализацию GetEngine() — Application знает какой движок использовать
//   - routing semantics FormEngine (GET /page/{key}, static POST event routes)
type UsersEditPage struct {
	*formengine.FormEngine
}

// NewUsersEditPage — фабрика для регистрации в Manifest.
func NewUsersEditPage() engine.Page {
	return &UsersEditPage{
		FormEngine: &formengine.FormEngine{},
	}
}

// Init вызывается на каждый request и только собирает DSL.
func (p *UsersEditPage) Init(ctx *engine.BuildContext) error {
	p.Text("name").
		Label("Имя пользователя").
		DefaultValue(ctx.Params["name"]).
		OnChange(OnNameChange)
	p.Text("email").
		Label("Email").
		DefaultValue(ctx.Params["email"])
	p.Button("save").
		Label("Сохранить").
		OnClick(OnSave)
	p.Button("pickUser").
		Label("Выбрать пользователя").
		OnClick(OnPickUser)
	p.Text("selectedUser").
		Label("Выбранный пользователь").
		ReadOnly(true)
	p.Text("status").
		Label("Статус").
		DefaultValue(ctx.Params["status"])
	p.Text("lastAction").
		Label("Последнее действие")
	p.Checkbox("nameChanged").
		Label("Имя изменено")

	return nil
}

func OnSave(ctx *formengine.RuntimeContext) {
	name, err := ctx.GetTextById("name")
	if err != nil {
		return
	}
	email, err := ctx.GetTextById("email")
	if err != nil {
		return
	}
	status, err := ctx.GetTextById("status")
	if err != nil {
		return
	}
	lastAction, err := ctx.GetTextById("lastAction")
	if err != nil {
		return
	}

	if strings.TrimSpace(stringValue(name.Element().Value)) == "" {
		ctx.SetError(errors.New("name is required"))
		return
	}
	if strings.TrimSpace(stringValue(email.Element().Value)) == "" {
		ctx.SetError(errors.New("email is required"))
		return
	}

	status.SetVisibility(!status.Element().Visibility)
	lastAction.SetValue(ctx.Params["form.actionId"])
}

func OnNameChange(ctx *formengine.RuntimeContext) {
	nameChanged, err := ctx.GetCheckboxById("nameChanged")
	if err != nil {
		return
	}
	nameChanged.SetValue(true)
}

func OnPickUser(ctx *formengine.RuntimeContext) {
	ctx.OpenDialog("users.picker", formengine.OpenOptions{
		Extra: map[string]any{
			"group_id": 10,
		},
		Callback: OnUserSelected,
	})
}

func OnUserSelected(ctx *formengine.RuntimeContext) {
	userID := ctx.Extra["user_id"]
	selectedUser, err := ctx.GetTextById("selectedUser")
	if err != nil {
		return
	}
	selectedUser.SetValue(userID)
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}
