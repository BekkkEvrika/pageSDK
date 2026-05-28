package page

import (
	"github.com/behzod/pageSDK/engine"
	inputs "github.com/behzod/pageSDK/form"
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
				},
			},
		},
		FormActions: &[]inputs.FormAction{
			{
				ID:      "save",
				Trigger: inputs.Click,
				Config: &inputs.FormActionConfig{
					Type:           inputs.APICall,
					URL:            "/event/users.edit/button/save",
					Method:         "POST",
					SuccessMessage: "Пользователь сохранён",
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
	ctx.Text("status").SetText("Saved")
	ctx.SetState("lastAction", ctx.Params["form.actionId"])
}

func OnNameChange(ctx *engine.RuntimeContext) {
	ctx.SetState("nameChanged", true)
}
