package page

import (
	"github.com/BekkkEvrika/pageSDK/engine"
	"github.com/BekkkEvrika/pageSDK/engine/formengine"
	inputs "github.com/BekkkEvrika/pageSDK/form"
)

// UsersPickerPage — dialog page выбора пользователя.
// Frontend хранит callback route, полученный из navigation action users.edit.
type UsersPickerPage struct {
	*formengine.FormEngine
}

func NewUsersPickerPage() engine.Page {
	return &UsersPickerPage{
		FormEngine: &formengine.FormEngine{},
	}
}

func (p *UsersPickerPage) Init(ctx *engine.BuildContext) error {
	p.CreateForm(inputs.Form{
		Containers: &[]inputs.Container{
			{
				Key:       "main",
				Direction: "vertical",
				Gap:       12,
				Fields: []inputs.Input{
					{
						Id:       "group",
						Type:     inputs.InputTypeText,
						Label:    "Group ID",
						ReadOnly: true,
					},
					{
						Id:       "selected",
						Type:     inputs.InputTypeText,
						Label:    "Selected user",
						ReadOnly: true,
					},
					{
						Id:    "selectAda",
						Type:  inputs.InputTypeButton,
						Label: "Select Ada",
					},
					{
						Id:    "selectGrace",
						Type:  inputs.InputTypeButton,
						Label: "Select Grace",
					},
					{
						Id:    "cancel",
						Type:  inputs.InputTypeButton,
						Label: "Cancel",
					},
				},
			},
		},
	})

	group, err := p.GetTextById("group")
	if err != nil {
		return err
	}
	group.SetDefaultValue(ctx.Params["group_id"])

	ada, err := p.GetButtonById("selectAda")
	if err != nil {
		return err
	}
	ada.SetOnClick(OnSelectAda)

	grace, err := p.GetButtonById("selectGrace")
	if err != nil {
		return err
	}
	grace.SetOnClick(OnSelectGrace)

	cancel, err := p.GetButtonById("cancel")
	if err != nil {
		return err
	}
	cancel.SetOnClick(OnCancelPicker)
	return nil
}

func OnSelectAda(ctx *formengine.RuntimeContext) {
	ctx.CloseWithResult(map[string]any{
		"user_id":   77,
		"user_name": "Ada Lovelace",
	})
}

func OnSelectGrace(ctx *formengine.RuntimeContext) {
	ctx.CloseWithResult(map[string]any{
		"user_id":   88,
		"user_name": "Grace Hopper",
	})
}

func OnCancelPicker(ctx *formengine.RuntimeContext) {
	ctx.Close()
}
