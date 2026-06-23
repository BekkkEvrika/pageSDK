package main

import (
	pagesdk "github.com/BekkkEvrika/pageSDK"
	"github.com/BekkkEvrika/pageSDK/engine"
	"github.com/BekkkEvrika/pageSDK/engine/formengine"
	inputs "github.com/BekkkEvrika/pageSDK/form"
)

func main() {
	application := pagesdk.New()

	if err := application.Run(projectInitial, ":8080"); err != nil {
		panic(err)
	}
}

func projectInitial(a *pagesdk.Application) {
	a.Manifest().Register("users.edit", NewUsersEditPage)
	a.Manifest().Register("controls.combos", NewComboExamplePage)
}

type UsersEditPage struct {
	*formengine.FormEngine
}

func NewUsersEditPage() pagesdk.Page {
	return &UsersEditPage{
		FormEngine: &formengine.FormEngine{},
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

func onSave(ctx *formengine.RuntimeContext) {
	status, err := ctx.GetTextById("status")
	if err != nil {
		return
	}
	status.SetValue("Saved")
}

func onNameChange(ctx *formengine.RuntimeContext) {
	nameChanged, err := ctx.GetCheckboxById("nameChanged")
	if err != nil {
		return
	}
	nameChanged.SetValue(true)
}

type ComboExamplePage struct {
	*formengine.FormEngine
}

func NewComboExamplePage() pagesdk.Page {
	return &ComboExamplePage{
		FormEngine: &formengine.FormEngine{},
	}
}

func (p *ComboExamplePage) Init(ctx *engine.BuildContext) error {
	country := ctx.Params["country"]
	if country == "" {
		country = "tj"
	}

	p.Select("country").
		Label("Country").
		Placeholder("Choose a country").
		Options(countryOptions()).
		DefaultValue(country).
		OnChange(onCountryChange)

	p.Select("city").
		Label("City").
		Placeholder("Choose a city").
		Options(cityOptions(country)).
		DefaultValue(ctx.Params["city"]).
		OnChange(onCityChange)

	p.Select("language").
		Label("Interface language").
		Options(inputs.ComboItems{
			{ID: "tg", Text: "Тоҷикӣ"},
			{ID: "ru", Text: "Русский"},
			{ID: "en", Text: "English"},
		}).
		DefaultValue("ru").
		OnChange(onLanguageChange)

	p.Text("selection").
		Label("Current selection").
		ReadOnly(true)

	p.Button("submit").
		Label("Submit").
		Variant("primary").
		OnClick(onComboSubmit)

	return nil
}

func onCountryChange(ctx *formengine.RuntimeContext) {
	country, err := ctx.GetSelectById("country")
	if err != nil {
		return
	}
	city, err := ctx.GetSelectById("city")
	if err != nil {
		return
	}

	countryID := stringValue(country.Element().Value)
	city.SetOptions(cityOptions(countryID))
	city.SetValue("")
	updateComboSelection(ctx)
}

func onCityChange(ctx *formengine.RuntimeContext) {
	updateComboSelection(ctx)
}

func onLanguageChange(ctx *formengine.RuntimeContext) {
	updateComboSelection(ctx)
}

func onComboSubmit(ctx *formengine.RuntimeContext) {
	updateComboSelection(ctx)
	ctx.ShowSuccess("Combo example", "Selected values were received by the backend")
}

func updateComboSelection(ctx *formengine.RuntimeContext) {
	country, err := ctx.GetSelectById("country")
	if err != nil {
		return
	}
	city, err := ctx.GetSelectById("city")
	if err != nil {
		return
	}
	language, err := ctx.GetSelectById("language")
	if err != nil {
		return
	}
	selection, err := ctx.GetTextById("selection")
	if err != nil {
		return
	}

	selection.SetValue(
		"country=" + stringValue(country.Element().Value) +
			", city=" + stringValue(city.Element().Value) +
			", language=" + stringValue(language.Element().Value),
	)
}

func countryOptions() inputs.ComboItems {
	return inputs.ComboItems{
		{ID: "tj", Text: "Tajikistan"},
		{ID: "uz", Text: "Uzbekistan"},
		{ID: "kz", Text: "Kazakhstan"},
	}
}

func cityOptions(country string) inputs.ComboItems {
	switch country {
	case "uz":
		return inputs.ComboItems{
			{ID: "tashkent", Text: "Tashkent"},
			{ID: "samarkand", Text: "Samarkand"},
			{ID: "bukhara", Text: "Bukhara"},
		}
	case "kz":
		return inputs.ComboItems{
			{ID: "astana", Text: "Astana"},
			{ID: "almaty", Text: "Almaty"},
			{ID: "shymkent", Text: "Shymkent"},
		}
	default:
		return inputs.ComboItems{
			{ID: "dushanbe", Text: "Dushanbe"},
			{ID: "khujand", Text: "Khujand"},
			{ID: "bokhtar", Text: "Bokhtar"},
		}
	}
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	text, _ := value.(string)
	return text
}
