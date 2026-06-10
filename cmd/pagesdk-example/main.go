package main

import (
	pagesdk "github.com/BekkkEvrika/pageSDK"
	"github.com/BekkkEvrika/pageSDK/engine"
	inputs "github.com/BekkkEvrika/pageSDK/form"
)

func main() {
	application := pagesdk.New()

	if err := application.Bootstrap(projectInitial, ":8080"); err != nil {
		panic(err)
	}
}

func projectInitial(a *pagesdk.Application) {
	a.Manifest().Register("users.edit", NewUsersEditPage)
}

type UsersEditPage struct {
	*engine.FormEngine
}

func NewUsersEditPage() pagesdk.Page {
	return &UsersEditPage{
		FormEngine: &engine.FormEngine{},
	}
}

func (p *UsersEditPage) Init(ctx *engine.BuildContext) error {
	name := p.Text("name")
	name.SetLabel("User name")
	name.SetPlaceholder("Enter user name")
	name.SetOnChange(onNameChange)

	email := p.Text("email")
	email.SetLabel("Email")
	email.SetPlaceholder("user@example.com")

	status := p.Text("status")
	status.SetLabel("Status")
	status.SetReadOnly(true)

	p.Field(inputs.Input{Id: "nameChanged", Type: inputs.InputTypeCheckbox})
	changed, err := p.GetCheckboxById("nameChanged")
	if err != nil {
		return err
	}
	changed.SetLabel("Name changed")
	changed.SetReadOnly(true)

	save := p.Button("save")
	save.SetLabel("Save")
	save.SetVariant("primary")
	save.SetOnClick(onSave)

	return nil
}

func onSave(ctx *engine.RuntimeContext) {
	status, err := ctx.GetTextById("status")
	if err != nil {
		return
	}
	status.SetValue("Saved")
}

func onNameChange(ctx *engine.RuntimeContext) {
	nameChanged, err := ctx.GetCheckboxById("nameChanged")
	if err != nil {
		return
	}
	nameChanged.SetValue(true)
}
