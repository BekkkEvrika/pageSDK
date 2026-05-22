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
//   - routing semantics FormEngine (GET /page/{key}, POST /event/{key}/{component}/{action})
type UsersEditPage struct {
	*engine.FormEngine
	dsl any
}

// NewUsersEditPage — фабрика для регистрации в Manifest.
func NewUsersEditPage() engine.Page {
	return &UsersEditPage{
		FormEngine: &engine.FormEngine{},
	}
}

// Init вызывается на каждый request и только собирает DSL.
func (p *UsersEditPage) Init(ctx *engine.BuildContext) error {
	p.dsl = inputs.Form{
		Containers: &[]inputs.Container{
			{
				Key:       "main",
				Direction: "vertical",
				Gap:       16,
				Fields: []inputs.Input{
					{
						Id:    "name",
						Type:  "text",
						Label: "Имя пользователя",
					},
					{
						Id:    "email",
						Type:  "email",
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
	}

	return nil
}

// DSL возвращает собранный declarative runtime model.
func (p *UsersEditPage) DSL() any {
	return p.dsl
}

// HandleEvent обрабатывает runtime events через explicit RuntimeContext.
func (p *UsersEditPage) HandleEvent(ctx *engine.RuntimeContext, event engine.Event) error {
	if event.Component == "button" && event.Action == "save" {
		state, ok := event.Payload.(*inputs.FormState)
		if !ok {
			return nil
		}
		ctx.Mutations.Update("form.status", map[string]any{"status": "ok"})
		ctx.Mutations.Update("form.lastAction", state.ActionID)
	}
	return nil
}
