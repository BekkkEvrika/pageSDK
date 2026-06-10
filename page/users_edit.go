package page

import (
	"github.com/BekkkEvrika/pageSDK/engine"
	inputs "github.com/BekkkEvrika/pageSDK/form"
)

// UsersEditPage — page редактирования пользователя.
// Stateless: создаётся на каждый request, уничтожается после ответа.
//
// Embedding *engine.FormEngine даёт:
//   - реализацию GetEngine() — Application знает какой движок использовать
//   - routing semantics FormEngine (GET /page/{key}, static POST event routes)
type UsersEditPage struct {
	*engine.FormEngine
}

// NewUsersEditPage — фабрика для регистрации в Manifest.
func NewUsersEditPage() engine.Page {
	return &UsersEditPage{
		FormEngine: &engine.FormEngine{},
	}
}

// Init вызывается на каждый request и только собирает DSL.
func (p *UsersEditPage) Init(ctx *engine.BuildContext) error {
	p.CreateForm(inputs.Form{
		Containers: &[]inputs.Container{
			{
				Key:       "main",
				Direction: "vertical",
				Gap:       16,
				Fields: []inputs.Input{
					{
						Id:    "name",
						Type:  inputs.InputTypeText,
						Label: "Имя пользователя",
					},
					{
						Id:    "email",
						Type:  inputs.InputTypeText,
						Label: "Email",
					},
					{
						Id:    "save",
						Type:  inputs.InputTypeButton,
						Label: "Сохранить",
					},
					{
						Id:    "status",
						Type:  inputs.InputTypeText,
						Label: "Статус",
					},
					{
						Id:    "lastAction",
						Type:  inputs.InputTypeText,
						Label: "Последнее действие",
					},
					{
						Id:    "nameChanged",
						Type:  inputs.InputTypeCheckbox,
						Label: "Имя изменено",
					},
				},
			},
		},
	})

	save, err := p.GetButtonById("save")
	if err != nil {
		return err
	}
	save.SetOnClick(OnSave)
	name, err := p.GetTextById("name")
	if err != nil {
		return err
	}
	name.SetOnChange(OnNameChange)
	return nil
}

func OnSave(ctx *engine.RuntimeContext) {
	status, err := ctx.GetTextById("status")
	if err != nil {
		return
	}
	lastAction, err := ctx.GetTextById("lastAction")
	if err != nil {
		return
	}
	status.SetVisibility(!status.Element().Visibility)
	lastAction.SetValue(ctx.Params["form.actionId"])
}

func OnNameChange(ctx *engine.RuntimeContext) {
	nameChanged, err := ctx.GetCheckboxById("nameChanged")
	if err != nil {
		return
	}
	nameChanged.SetValue(true)
}
