package engine

import (
	"errors"
	"testing"

	inputs "github.com/BekkkEvrika/pageSDK/form"
)

type testFormPage struct {
	*FormEngine
}

func (p *testFormPage) Init(ctx *BuildContext) error {
	p.CreateForm(inputs.Form{
		Containers: &[]inputs.Container{
			{
				Key: "main",
				Fields: []inputs.Input{
					{Id: "request", Type: inputs.InputTypeText},
				},
			},
		},
	})
	return nil
}

type testTablePage struct {
	*TableEngine
}

func (p *testTablePage) Init(ctx *BuildContext) error {
	p.Column("request", "Request")
	return nil
}

type testFormEventPage struct {
	*FormEngine
}

func (p *testFormEventPage) Init(ctx *BuildContext) error {
	save := p.Button("save")
	save.SetOnClick(testOnSave)

	name := p.Text("name")
	name.SetOnChange(testOnNameChange)
	p.Text("saved")
	p.Text("changed")

	return nil
}

type testFormStatePage struct {
	*FormEngine
}

type testMissingRuntimeControlPage struct {
	*FormEngine
}

type testFormRuntimeErrorPage struct {
	*FormEngine
}

type testTableRuntimeErrorPage struct {
	*TableEngine
}

func (p *testMissingRuntimeControlPage) Init(ctx *BuildContext) error {
	save := p.Button("save")
	save.SetOnClick(func(ctx *RuntimeContext) {
		missing, err := ctx.GetTextById("missing")
		if err != nil {
			return
		}
		missing.SetValue(true)
	})
	return nil
}

func (p *testFormRuntimeErrorPage) Init(ctx *BuildContext) error {
	save := p.Button("save")
	save.SetOnClick(func(ctx *RuntimeContext) {
		ctx.SetError(errors.New("form handler failed"))
	})
	return nil
}

func (p *testTableRuntimeErrorPage) Init(ctx *BuildContext) error {
	p.Column("request", "Request")
	return nil
}

func (p *testTableRuntimeErrorPage) HandleEvent(ctx *RuntimeContext, event Event) error {
	ctx.SetError(errors.New("table handler failed"))
	return nil
}

func (p *testFormStatePage) Init(ctx *BuildContext) error {
	p.Text("name")
	save := p.Button("save")
	p.Text("missing")
	p.Text("fieldId")
	p.Text("fieldType")
	p.Text("fieldLabel")
	p.Text("fieldPlaceholder")
	p.Text("fieldValue")
	p.Text("fieldMode")
	p.Text("actionId")
	p.Text("changedField")
	p.Text("senderId")
	p.Text("senderType")
	p.Text("senderValue")
	save.SetOnClick(func(ctx *RuntimeContext) {
		name, err := ctx.GetTextById("name")
		if err != nil {
			missing, err := ctx.GetTextById("missing")
			if err != nil {
				return
			}
			missing.SetValue(true)
			return
		}
		field := name.Element()
		fieldID, err := ctx.GetTextById("fieldId")
		if err != nil {
			return
		}
		fieldType, err := ctx.GetTextById("fieldType")
		if err != nil {
			return
		}
		fieldLabel, err := ctx.GetTextById("fieldLabel")
		if err != nil {
			return
		}
		fieldPlaceholder, err := ctx.GetTextById("fieldPlaceholder")
		if err != nil {
			return
		}
		fieldValue, err := ctx.GetTextById("fieldValue")
		if err != nil {
			return
		}
		fieldMode, err := ctx.GetTextById("fieldMode")
		if err != nil {
			return
		}
		actionID, err := ctx.GetTextById("actionId")
		if err != nil {
			return
		}
		changedField, err := ctx.GetTextById("changedField")
		if err != nil {
			return
		}
		fieldID.SetValue(field.Id)
		fieldType.SetValue(field.Type)
		fieldLabel.SetValue(field.Label)
		fieldPlaceholder.SetValue(field.Placeholder)
		fieldValue.SetValue(field.Value)
		fieldMode.SetValue(field.Props["mode"])
		actionID.SetValue(ctx.FormState.ActionID)
		changedField.SetValue(ctx.FormState.ChangedField)
		if ctx.Sender != nil {
			senderID, err := ctx.GetTextById("senderId")
			if err != nil {
				return
			}
			senderType, err := ctx.GetTextById("senderType")
			if err != nil {
				return
			}
			senderValue, err := ctx.GetTextById("senderValue")
			if err != nil {
				return
			}
			senderID.SetValue(ctx.Sender.Id)
			senderType.SetValue(ctx.Sender.Type)
			senderValue.SetValue(ctx.Sender.Value)
		}
	})
	return nil
}

func testOnSave(ctx *RuntimeContext) {
	saved, err := ctx.GetTextById("saved")
	if err != nil {
		return
	}
	saved.SetValue(true)
}

func testOnNameChange(ctx *RuntimeContext) {
	changed, err := ctx.GetTextById("changed")
	if err != nil {
		return
	}
	changed.SetValue(true)
}

func TestFormRouteUsesRequestEngineInstance(t *testing.T) {
	bootstrapEngine := &FormEngine{}
	route := bootstrapEngine.Routes("test.form", nil)[0]
	page := &testFormPage{FormEngine: &FormEngine{}}

	result, err := route.Handler(&RequestContext{}, page)
	if err != nil {
		t.Fatalf("route handler returned error: %v", err)
	}

	render, ok := result.(*RenderResult)
	if !ok {
		t.Fatalf("expected *RenderResult, got %T", result)
	}
	form, ok := render.DSL.(inputs.Form)
	if !ok {
		t.Fatalf("expected form DSL, got %T", render.DSL)
	}
	if form.Containers == nil || len(*form.Containers) != 1 {
		t.Fatalf("expected request form container, got %#v", form.Containers)
	}
	if fields := (*form.Containers)[0].Fields; len(fields) != 1 || fields[0].Id != "request" {
		t.Fatalf("expected request engine field, got %#v", fields)
	}
	if bootstrapForm := bootstrapEngine.DSL().(inputs.Form); bootstrapForm.Containers != nil {
		t.Fatalf("bootstrap engine should not own request DSL: %#v", bootstrapForm)
	}
}

func TestCreateFormStoresFormDSL(t *testing.T) {
	engine := &FormEngine{}
	engine.CreateForm(inputs.Form{
		Containers: &[]inputs.Container{
			{
				Key: "main",
				Fields: []inputs.Input{
					{Id: "name", Type: inputs.InputTypeText},
				},
			},
		},
	})

	form, ok := engine.DSL().(inputs.Form)
	if !ok {
		t.Fatalf("expected form DSL, got %T", engine.DSL())
	}
	if form.Containers == nil || len(*form.Containers) != 1 {
		t.Fatalf("expected stored form container, got %#v", form.Containers)
	}
	if fields := (*form.Containers)[0].Fields; len(fields) != 1 || fields[0].Id != "name" {
		t.Fatalf("expected stored form field, got %#v", fields)
	}
}

func TestFormEngineGetsTypedInputsByID(t *testing.T) {
	engine := &FormEngine{}
	engine.CreateForm(inputs.Form{
		Containers: &[]inputs.Container{
			{
				Key: "main",
				Fields: []inputs.Input{
					{Id: "select", Type: inputs.InputTypeSelect},
					{Id: "date", Type: inputs.InputTypeDate},
					{Id: "datetime", Type: inputs.InputTypeDatetime},
					{Id: "text", Type: inputs.InputTypeText},
					{Id: "number", Type: inputs.InputTypeNumber},
					{Id: "checkbox", Type: inputs.InputTypeCheckbox},
					{Id: "label", Type: inputs.InputTypeLabel},
					{Id: "search", Type: inputs.InputTypeSearch},
					{Id: "textarea", Type: inputs.InputTypeTextarea},
					{Id: "hidden", Type: inputs.InputTypeHidden},
					{Id: "file", Type: inputs.InputTypeFile},
					{Id: "button", Type: inputs.InputTypeButton},
				},
				Containers: []inputs.Container{
					{
						Key: "nested",
						Fields: []inputs.Input{
							{Id: "nestedName", Type: inputs.InputTypeText},
						},
					},
				},
			},
		},
	})

	assertElement := func(id string, element interface{ Input() *inputs.Input }, err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("expected %s input: %v", id, err)
		}
		if element.Input().Id != id {
			t.Fatalf("expected %s input, got %#v", id, element.Input())
		}
	}

	getters := map[string]func() (interface{ Input() *inputs.Input }, error){
		"select": func() (interface{ Input() *inputs.Input }, error) {
			return engine.GetSelectById("select")
		},
		"date": func() (interface{ Input() *inputs.Input }, error) {
			return engine.GetDateById("date")
		},
		"datetime": func() (interface{ Input() *inputs.Input }, error) {
			return engine.GetDatetimeById("datetime")
		},
		"text": func() (interface{ Input() *inputs.Input }, error) {
			return engine.GetTextById("text")
		},
		"number": func() (interface{ Input() *inputs.Input }, error) {
			return engine.GetNumberById("number")
		},
		"checkbox": func() (interface{ Input() *inputs.Input }, error) {
			return engine.GetCheckboxById("checkbox")
		},
		"label": func() (interface{ Input() *inputs.Input }, error) {
			return engine.GetLabelById("label")
		},
		"search": func() (interface{ Input() *inputs.Input }, error) {
			return engine.GetSearchById("search")
		},
		"textarea": func() (interface{ Input() *inputs.Input }, error) {
			return engine.GetTextareaById("textarea")
		},
		"hidden": func() (interface{ Input() *inputs.Input }, error) {
			return engine.GetHiddenById("hidden")
		},
		"file": func() (interface{ Input() *inputs.Input }, error) {
			return engine.GetFileById("file")
		},
		"button": func() (interface{ Input() *inputs.Input }, error) {
			return engine.GetButtonById("button")
		},
	}
	for id, getter := range getters {
		element, err := getter()
		assertElement(id, element, err)
	}

	nested, err := engine.GetTextById("nestedName")
	if err != nil {
		t.Fatalf("expected nested text input: %v", err)
	}
	if nested.Input().Id != "nestedName" {
		t.Fatalf("expected nested text input nestedName, got %#v", nested)
	}
}

func TestFormEngineTypedInputMutatesStoredDSL(t *testing.T) {
	engine := &FormEngine{}
	engine.CreateForm(inputs.Form{
		Containers: &[]inputs.Container{
			{
				Key: "main",
				Fields: []inputs.Input{
					{Id: "save", Type: inputs.InputTypeButton},
				},
			},
		},
	})

	button, err := engine.GetButtonById("save")
	if err != nil {
		t.Fatalf("expected button input: %v", err)
	}
	button.SetLabel("Save")
	button.SetVariant("primary")
	button.SetActionID("save-user")

	input, err := engine.GetInputById("save")
	if err != nil {
		t.Fatalf("expected raw input: %v", err)
	}
	if input.Label != "Save" || input.Variant != "primary" || input.ActionID != "save-user" {
		t.Fatalf("expected stored DSL mutation, got %#v", input)
	}
}

func TestFormEngineTypedInputLookupErrors(t *testing.T) {
	engine := &FormEngine{}
	engine.CreateForm(inputs.Form{
		Containers: &[]inputs.Container{
			{
				Key: "main",
				Fields: []inputs.Input{
					{Id: "name", Type: inputs.InputTypeText},
				},
			},
		},
	})

	if _, err := engine.GetInputById("missing"); err == nil {
		t.Fatal("expected missing input error")
	}
	if _, err := engine.GetButtonById("name"); err == nil {
		t.Fatal("expected wrong input type error")
	}
}

func TestFormEngineGeneratesStaticEventRoutes(t *testing.T) {
	page := &testFormEventPage{FormEngine: &FormEngine{}}
	routes := page.FormEngine.Routes("test.form", page)

	paths := make(map[string]RouteHandler)
	for _, route := range routes {
		paths[route.Path] = route.Handler
	}

	if _, ok := paths["/event/test.form/button/save"]; !ok {
		t.Fatalf("expected button event route, got %#v", paths)
	}
	if _, ok := paths["/event/test.form/text/name"]; !ok {
		t.Fatalf("expected text event route, got %#v", paths)
	}
	if _, ok := paths["/event/test.form/:component/:action"]; ok {
		t.Fatalf("unexpected dynamic event route: %#v", paths)
	}
}

func TestFormEngineIncludesRegisteredEventsInRenderedActions(t *testing.T) {
	page := &testFormEventPage{FormEngine: &FormEngine{}}
	result, err := page.FormEngine.Render(&RequestContext{PageKey: "test.form"}, page)
	if err != nil {
		t.Fatalf("render returned error: %v", err)
	}

	form, ok := result.DSL.(inputs.Form)
	if !ok {
		t.Fatalf("expected form DSL, got %T", result.DSL)
	}
	if form.FormActions == nil {
		t.Fatal("expected rendered form actions")
	}
	actions := *form.FormActions
	if len(actions) != 2 {
		t.Fatalf("expected registered event actions without duplicates, got %#v", actions)
	}

	assertAction := func(id string, trigger inputs.FormActionTrigger, actionType inputs.FormActionType, url string) {
		t.Helper()
		for _, action := range actions {
			if action.ID == id && action.Trigger == trigger {
				if action.Config == nil {
					t.Fatalf("expected config for action %#v", action)
				}
				if action.Config.Type != actionType || action.Config.URL != url || action.Config.Method != "POST" {
					t.Fatalf("unexpected action config for %s: %#v", id, action.Config)
				}
				return
			}
		}
		t.Fatalf("expected action %s/%s in %#v", id, trigger, actions)
	}

	assertAction("save", inputs.Click, inputs.APICall, "/event/test.form/button/save")
	assertAction("name", inputs.Change, inputs.ChangeAPICall, "/event/test.form/text/name")
}

func TestFormEngineOwnsComponentsHandlersAndRouteMetadata(t *testing.T) {
	page := &testFormEventPage{FormEngine: &FormEngine{}}
	page.FormEngine.Routes("test.form", page)

	if page.FormEngine.root.Key != "root" {
		t.Fatalf("expected root container, got %#v", page.FormEngine.root)
	}
	if len(page.FormEngine.root.Containers) != 1 || page.FormEngine.root.Containers[0].Key != "main" {
		t.Fatalf("expected component tree under root, got %#v", page.FormEngine.root)
	}
	if got := page.FormEngine.components["save"]; got.ID != "save" || got.Type != inputs.InputTypeButton || got.Path != "root.main.save" {
		t.Fatalf("expected save button component, got %#v", got)
	}
	if got := page.FormEngine.components["name"]; got.ID != "name" || got.Type != inputs.InputTypeText || got.Path != "root.main.name" {
		t.Fatalf("expected name text component, got %#v", got)
	}
	if page.FormEngine.handlers[inputs.InputTypeButton]["save"] == nil {
		t.Fatalf("expected button/save handler registry, got %#v", page.FormEngine.handlers)
	}
	if page.FormEngine.handlers[inputs.InputTypeText]["name"] == nil {
		t.Fatalf("expected text/name handler registry, got %#v", page.FormEngine.handlers)
	}
	if len(page.FormEngine.eventRoutes) != 2 {
		t.Fatalf("expected generated event route metadata, got %#v", page.FormEngine.eventRoutes)
	}
	if page.FormEngine.eventRoutes[0].Path != "/event/test.form/button/save" {
		t.Fatalf("expected deterministic first event route, got %#v", page.FormEngine.eventRoutes)
	}
	if page.FormEngine.eventRoutes[1].Path != "/event/test.form/text/name" {
		t.Fatalf("expected deterministic second event route, got %#v", page.FormEngine.eventRoutes)
	}
}

func TestFormEngineStaticEventRouteInvokesFreshPageHandler(t *testing.T) {
	page := &testFormEventPage{FormEngine: &FormEngine{}}
	routes := page.FormEngine.Routes("test.form", page)

	var handler RouteHandler
	for _, route := range routes {
		if route.Path == "/event/test.form/button/save" {
			handler = route.Handler
			break
		}
	}
	if handler == nil {
		t.Fatal("expected button event route handler")
	}

	result, err := handler(&RequestContext{}, &testFormEventPage{FormEngine: &FormEngine{}})
	if err != nil {
		t.Fatalf("route handler returned error: %v", err)
	}
	runtime, ok := result.(*RuntimeResult)
	if !ok {
		t.Fatalf("expected *RuntimeResult, got %T", result)
	}
	if len(runtime.Mutations) != 1 || runtime.Mutations[0].Path != "controls.saved.value" {
		t.Fatalf("expected save mutation, got %#v", runtime.Mutations)
	}
}

func TestFormEngineRuntimeContextReceivesFullFormState(t *testing.T) {
	page := &testFormStatePage{FormEngine: &FormEngine{}}
	routes := page.FormEngine.Routes("test.form", page)

	var handler RouteHandler
	for _, route := range routes {
		if route.Path == "/event/test.form/button/save" {
			handler = route.Handler
			break
		}
	}
	if handler == nil {
		t.Fatal("expected button event route handler")
	}

	body := []byte(`{
		"elements": [
			{
				"id": "name",
				"type": "text",
				"label": "Name",
				"placeholder": "Enter name",
				"value": "Alice",
				"mode": "editable"
			},
			{
				"id": "save",
				"type": "button",
				"label": "Save",
				"value": true
			}
		],
		"sender": {
			"id": "save",
			"type": "button",
			"label": "Save",
			"value": true
		},
		"actionId": "save",
		"trigger": "click",
		"changedField": "save"
	}`)
	result, err := handler(&RequestContext{Body: body}, &testFormStatePage{FormEngine: &FormEngine{}})
	if err != nil {
		t.Fatalf("route handler returned error: %v", err)
	}
	runtime, ok := result.(*RuntimeResult)
	if !ok {
		t.Fatalf("expected *RuntimeResult, got %T", result)
	}
	got := map[string]any{}
	for _, mutation := range runtime.Mutations {
		got[mutation.Path] = mutation.Value
	}
	want := map[string]any{
		"controls.fieldId.value":          "name",
		"controls.fieldType.value":        inputs.InputTypeText,
		"controls.fieldLabel.value":       "Name",
		"controls.fieldPlaceholder.value": "Enter name",
		"controls.fieldValue.value":       "Alice",
		"controls.fieldMode.value":        "editable",
		"controls.actionId.value":         "save",
		"controls.changedField.value":     "save",
		"controls.senderId.value":         "save",
		"controls.senderType.value":       inputs.InputTypeButton,
		"controls.senderValue.value":      true,
	}
	for path, value := range want {
		if got[path] != value {
			t.Fatalf("expected mutation %s=%#v, got %#v", path, value, runtime.Mutations)
		}
	}
}

func TestFormEngineRuntimeMutationRequiresExistingDSLControl(t *testing.T) {
	page := &testMissingRuntimeControlPage{FormEngine: &FormEngine{}}
	routes := page.FormEngine.Routes("test.form", page)

	var handler RouteHandler
	for _, route := range routes {
		if route.Path == "/event/test.form/button/save" {
			handler = route.Handler
			break
		}
	}
	if handler == nil {
		t.Fatal("expected button event route handler")
	}

	_, err := handler(&RequestContext{}, &testMissingRuntimeControlPage{FormEngine: &FormEngine{}})
	if err == nil {
		t.Fatal("expected missing runtime control error")
	}
	if got := err.Error(); got != `runtime context: input "missing" not found in DSL` {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFormEngineReturnsRuntimeContextError(t *testing.T) {
	page := &testFormRuntimeErrorPage{FormEngine: &FormEngine{}}
	routes := page.FormEngine.Routes("test.form", page)

	var handler RouteHandler
	for _, route := range routes {
		if route.Path == "/event/test.form/button/save" {
			handler = route.Handler
			break
		}
	}
	if handler == nil {
		t.Fatal("expected button event route handler")
	}

	_, err := handler(&RequestContext{}, &testFormRuntimeErrorPage{FormEngine: &FormEngine{}})
	if err == nil {
		t.Fatal("expected runtime context error")
	}
	if got := err.Error(); got != "form handler failed" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRuntimeContextExplicitOperations(t *testing.T) {
	ctx := (&RequestContext{}).RuntimeContext()
	root := newRootContainer()
	root.Fields = []inputs.Input{
		{Id: "title", Type: inputs.InputTypeText},
		{Id: "loading", Type: inputs.InputTypeText},
		{Id: "old_button", Type: inputs.InputTypeButton},
	}
	ctx.bindFormTree(&root)

	title, err := ctx.GetTextById("title")
	if err != nil {
		t.Fatalf("expected title control: %v", err)
	}
	loading, err := ctx.GetTextById("loading")
	if err != nil {
		t.Fatalf("expected loading control: %v", err)
	}
	title.SetLabel("Saved")
	loading.SetValue(false)
	loading.SetVisibility(false)
	ctx.Form().Add(inputs.Input{Id: "dynamic_text", Type: inputs.InputTypeText})
	ctx.Remove("old_button")
	ctx.OpenDialog("users.edit")
	ctx.OpenTab("analytics.dashboard")
	ctx.Close()
	ctx.CloseWithResult(Params{"id": "42"})

	if len(ctx.Mutations) != 5 {
		t.Fatalf("expected explicit mutations, got %#v", ctx.Mutations)
	}
	if ctx.Mutations[0].Type != MutationUpdate || ctx.Mutations[0].Path != "controls.title.label" {
		t.Fatalf("expected title text update mutation, got %#v", ctx.Mutations[0])
	}
	if ctx.Mutations[1].Type != MutationUpdate || ctx.Mutations[1].Path != "controls.loading.value" {
		t.Fatalf("expected loading value update mutation, got %#v", ctx.Mutations[1])
	}
	if ctx.Mutations[2].Type != MutationUpdate || ctx.Mutations[2].Path != "controls.loading.visibility" {
		t.Fatalf("expected loading visibility update mutation, got %#v", ctx.Mutations[2])
	}
	if ctx.Mutations[3].Type != MutationAdd || ctx.Mutations[3].Path != "form.controls" {
		t.Fatalf("expected add mutation, got %#v", ctx.Mutations[3])
	}
	if findInputByIdInContainer(&root, "dynamic_text") == nil {
		t.Fatalf("expected add operation to update runtime tree, got %#v", root)
	}
	if ctx.Mutations[4].Type != MutationRemove || ctx.Mutations[4].Path != "controls.old_button" {
		t.Fatalf("expected remove mutation, got %#v", ctx.Mutations[4])
	}
	if len(ctx.Navigation) != 4 {
		t.Fatalf("expected navigation actions, got %#v", ctx.Navigation)
	}
}

func TestTableRouteUsesRequestEngineInstance(t *testing.T) {
	bootstrapEngine := &TableEngine{}
	route := bootstrapEngine.Routes("test.table", nil)[0]
	page := &testTablePage{TableEngine: &TableEngine{}}

	result, err := route.Handler(&RequestContext{}, page)
	if err != nil {
		t.Fatalf("route handler returned error: %v", err)
	}

	render, ok := result.(*RenderResult)
	if !ok {
		t.Fatalf("expected *RenderResult, got %T", result)
	}
	table, ok := render.DSL.(TableDSL)
	if !ok {
		t.Fatalf("expected table DSL, got %T", render.DSL)
	}
	if len(table.Columns) != 1 || table.Columns[0].Key != "request" {
		t.Fatalf("expected request engine column, got %#v", table.Columns)
	}
	if bootstrapTable := bootstrapEngine.DSL().(TableDSL); len(bootstrapTable.Columns) != 0 {
		t.Fatalf("bootstrap engine should not own request DSL: %#v", bootstrapTable)
	}
}

func TestTableEngineReturnsRuntimeContextError(t *testing.T) {
	page := &testTableRuntimeErrorPage{TableEngine: &TableEngine{}}
	handler := page.TableEngine.Routes("test.table", page)[1].Handler

	_, err := handler(&RequestContext{
		Params: Params{
			"component": "table",
			"action":    "rowClick",
		},
	}, &testTableRuntimeErrorPage{TableEngine: &TableEngine{}})
	if err == nil {
		t.Fatal("expected runtime context error")
	}
	if got := err.Error(); got != "table handler failed" {
		t.Fatalf("unexpected error: %v", err)
	}
}
