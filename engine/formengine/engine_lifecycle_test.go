package formengine

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/BekkkEvrika/pageSDK/engine"
	"github.com/BekkkEvrika/pageSDK/engine/tableengine"
	inputs "github.com/BekkkEvrika/pageSDK/form"
	"github.com/BekkkEvrika/pageSDK/table"
)

type BuildContext = engine.BuildContext
type Params = engine.Params
type RequestContext = engine.RequestContext
type RenderResult = engine.RenderResult
type RouteHandler = engine.RouteHandler
type RuntimeResult = engine.RuntimeResult
type TableDSL = table.TableSchema
type TableEngine = tableengine.TableEngine

const MutationAdd = engine.MutationAdd
const MutationRemove = engine.MutationRemove
const MutationUpdate = engine.MutationUpdate

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
	p.Button("save").
		Label("Save").
		Variant("primary").
		OnClick(testOnSave)

	p.Text("name").
		Label("Name").
		Placeholder("Enter name").
		OnChange(testOnNameChange)
	p.Text("saved")
	p.Text("changed")

	return nil
}

type testFormStatePage struct {
	*FormEngine
}

type testDefaultRuntimeValuePage struct {
	*FormEngine
}

type testMissingRuntimeControlPage struct {
	*FormEngine
}

type testFormRuntimeErrorPage struct {
	*FormEngine
}

type testNavigationCallbackPage struct {
	*FormEngine
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

func (p *testNavigationCallbackPage) Init(ctx *BuildContext) error {
	p.Text("selected_user")
	p.Button("pick").SetOnClick(func(ctx *RuntimeContext) {
		ctx.OpenDialog("users.picker", OpenOptions{
			Extra: map[string]any{
				"group_id": 10,
			},
			Callback: onUserSelected,
		})
	})
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

func (p *testDefaultRuntimeValuePage) Init(ctx *BuildContext) error {
	name := p.Text("name")
	name.SetDefaultValue("guest")
	p.Text("note")
	p.Text("result")
	p.Text("emptyResult")
	p.Field(inputs.Input{Id: "enabled", Type: inputs.InputTypeCheckbox})
	checkbox, err := p.GetCheckboxById("enabled")
	if err != nil {
		return err
	}
	checkbox.SetOnChange(func(ctx *RuntimeContext) {
		name, err := ctx.GetTextById("name")
		if err != nil {
			return
		}
		note, err := ctx.GetTextById("note")
		if err != nil {
			return
		}
		result, err := ctx.GetTextById("result")
		if err != nil {
			return
		}
		emptyResult, err := ctx.GetTextById("emptyResult")
		if err != nil {
			return
		}
		result.SetValue(name.Value.(string))
		emptyResult.SetValue(note.Value.(string))
	})
	return nil
}

func testOnSave(ctx *RuntimeContext) {
	saved, err := ctx.GetTextById("saved")
	if err != nil {
		return
	}
	saved.SetValue(true)
	ctx.ShowSuccess("Saved", "User was saved")
}

func testOnNameChange(ctx *RuntimeContext) {
	changed, err := ctx.GetTextById("changed")
	if err != nil {
		return
	}
	changed.SetValue(true)
}

func onUserSelected(ctx *RuntimeContext) {
	selectedUser, err := ctx.GetTextById("selected_user")
	if err != nil {
		return
	}
	selectedUser.SetValue(ctx.Extra["user_id"])
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

func TestLabelExposesOnlyLabelSpecificMutators(t *testing.T) {
	if _, ok := reflect.TypeOf(&Label{}).MethodByName("SetDefaultValue"); ok {
		t.Fatal("label must not expose SetDefaultValue")
	}
	if _, ok := reflect.TypeOf(&RuntimeLabel{}).MethodByName("SetValue"); ok {
		t.Fatal("runtime label must not expose SetValue")
	}
	if got := defaultRuntimeValue(&inputs.Input{Id: "label", Type: inputs.InputTypeLabel}); got != nil {
		t.Fatalf("expected label to have no runtime value, got %#v", got)
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
	if _, ok := paths["/event/test.form/dialog/:dialog"]; !ok {
		t.Fatalf("expected dialog event route, got %#v", paths)
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
	if len(runtime.Dialogs) != 1 || runtime.Dialogs[0].Level != engine.DialogSuccess {
		t.Fatalf("expected success dialog, got %#v", runtime.Dialogs)
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

func TestRuntimeControlUsesDefaultValueWhenEventPayloadOmitsValue(t *testing.T) {
	page := &testDefaultRuntimeValuePage{FormEngine: &FormEngine{}}
	routes := page.FormEngine.Routes("test.form", page)

	var handler RouteHandler
	for _, route := range routes {
		if route.Path == "/event/test.form/checkbox/enabled" {
			handler = route.Handler
			break
		}
	}
	if handler == nil {
		t.Fatal("expected checkbox event route handler")
	}

	body := []byte(`{
		"elements": [
			{"id": "name", "type": "text"},
			{"id": "enabled", "type": "checkbox", "value": true}
		],
		"changedField": "enabled"
	}`)
	result, err := handler(&RequestContext{Body: body}, &testDefaultRuntimeValuePage{FormEngine: &FormEngine{}})
	if err != nil {
		t.Fatalf("route handler returned error: %v", err)
	}
	runtime, ok := result.(*RuntimeResult)
	if !ok {
		t.Fatalf("expected *RuntimeResult, got %T", result)
	}
	if len(runtime.Mutations) != 2 {
		t.Fatalf("expected two mutations, got %#v", runtime.Mutations)
	}
	got := map[string]any{}
	for _, mutation := range runtime.Mutations {
		got[mutation.Path] = mutation.Value
	}
	if got["controls.result.value"] != "guest" {
		t.Fatalf("expected default value mutation, got %#v", runtime.Mutations)
	}
	if got["controls.emptyResult.value"] != "" {
		t.Fatalf("expected empty string fallback mutation, got %#v", runtime.Mutations)
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
	ctx := NewRuntimeContext(&RequestContext{})
	root := newRootContainer()
	root.Fields = []inputs.Input{
		{Id: "title", Type: inputs.InputTypeText},
		{Id: "loading", Type: inputs.InputTypeText},
		{Id: "old_button", Type: inputs.InputTypeButton},
	}
	ctx.BindFormTree(&root)

	title, err := ctx.GetTextById("title")
	if err != nil {
		t.Fatalf("expected title control: %v", err)
	}
	loading, err := ctx.GetTextById("loading")
	if err != nil {
		t.Fatalf("expected loading control: %v", err)
	}
	title.SetLabel("Saved")
	title.SetHint("Saved successfully")
	loading.SetValue(false)
	loading.SetVisibility(false)
	ctx.Form().Add(inputs.Input{Id: "dynamic_text", Type: inputs.InputTypeText})
	ctx.Remove("old_button")
	ctx.OpenDialog("users.edit")
	ctx.ShowDialog(engine.Dialog{
		Title:       "Saved",
		Description: "Changes were saved",
		Level:       engine.DialogSuccess,
		Actions: []engine.DialogAction{
			{Name: "OK", Value: "ok"},
		},
	})
	ctx.OpenTab("analytics.dashboard")
	ctx.Close()
	ctx.CloseWithResult(Params{"id": "42"})

	if len(ctx.Mutations) != 6 {
		t.Fatalf("expected explicit mutations, got %#v", ctx.Mutations)
	}
	if ctx.Mutations[0].Type != MutationUpdate || ctx.Mutations[0].Path != "controls.title.label" {
		t.Fatalf("expected title text update mutation, got %#v", ctx.Mutations[0])
	}
	if ctx.Mutations[1].Type != MutationUpdate || ctx.Mutations[1].Path != "controls.title.hint" || ctx.Mutations[1].Value != "Saved successfully" {
		t.Fatalf("expected title hint update mutation, got %#v", ctx.Mutations[1])
	}
	if root.Fields[0].Hint != "Saved successfully" {
		t.Fatalf("expected title hint in runtime tree, got %#v", root.Fields[0])
	}
	if ctx.Mutations[2].Type != MutationUpdate || ctx.Mutations[2].Path != "controls.loading.value" {
		t.Fatalf("expected loading value update mutation, got %#v", ctx.Mutations[2])
	}
	if ctx.Mutations[3].Type != MutationUpdate || ctx.Mutations[3].Path != "controls.loading.visibility" {
		t.Fatalf("expected loading visibility update mutation, got %#v", ctx.Mutations[3])
	}
	if ctx.Mutations[4].Type != MutationAdd || ctx.Mutations[4].Path != "form.controls" {
		t.Fatalf("expected add mutation, got %#v", ctx.Mutations[4])
	}
	if !rootHasField(&root, "dynamic_text") {
		t.Fatalf("expected add operation to update runtime tree, got %#v", root)
	}
	if ctx.Mutations[5].Type != MutationRemove || ctx.Mutations[5].Path != "controls.old_button" {
		t.Fatalf("expected remove mutation, got %#v", ctx.Mutations[5])
	}
	if len(ctx.Navigation) != 4 {
		t.Fatalf("expected navigation actions, got %#v", ctx.Navigation)
	}
	if len(ctx.Dialogs) != 1 {
		t.Fatalf("expected dialog action, got %#v", ctx.Dialogs)
	}
	if ctx.Dialogs[0].Title != "Saved" || ctx.Dialogs[0].Level != engine.DialogSuccess || ctx.Dialogs[0].Actions[0].Value != "ok" {
		t.Fatalf("unexpected dialog action: %#v", ctx.Dialogs[0])
	}
}

func TestRuntimeContextDialogHelpers(t *testing.T) {
	ctx := NewRuntimeContext(&RequestContext{})

	ctx.ShowMessage("Message", "Plain message")
	ctx.ShowWarning("Warning", "Careful")
	ctx.ShowError("Error", "Something failed")
	ctx.ShowSuccess("Success", "Done")
	ctx.ShowYesNo("Confirm", "Continue?", func(value string) {})
	ctx.ShowOKCancel("Edit", "Save changes?", func(value string) {})
	ctx.ShowDialog(engine.Dialog{
		Title:       "Custom",
		Description: "Choose custom action",
		Level:       engine.DialogWarning,
		Actions: []engine.DialogAction{
			{Name: "Retry", Value: "retry"},
			{Name: "Ignore", Value: "ignore"},
		},
	}, func(value string) {})

	if len(ctx.Dialogs) != 7 {
		t.Fatalf("expected seven dialogs, got %#v", ctx.Dialogs)
	}
	tests := []struct {
		index   int
		level   engine.DialogLevel
		actions []engine.DialogAction
	}{
		{index: 0, level: engine.DialogInfo, actions: []engine.DialogAction{{Name: "OK", Value: "ok"}}},
		{index: 1, level: engine.DialogWarning, actions: []engine.DialogAction{{Name: "OK", Value: "ok"}}},
		{index: 2, level: engine.DialogError, actions: []engine.DialogAction{{Name: "OK", Value: "ok"}}},
		{index: 3, level: engine.DialogSuccess, actions: []engine.DialogAction{{Name: "OK", Value: "ok"}}},
		{index: 4, level: engine.DialogInfo, actions: []engine.DialogAction{{Name: "Yes", Value: "yes"}, {Name: "No", Value: "no"}}},
		{index: 5, level: engine.DialogInfo, actions: []engine.DialogAction{{Name: "OK", Value: "ok"}, {Name: "Cancel", Value: "cancel"}}},
		{index: 6, level: engine.DialogWarning, actions: []engine.DialogAction{{Name: "Retry", Value: "retry"}, {Name: "Ignore", Value: "ignore"}}},
	}

	for _, test := range tests {
		dialog := ctx.Dialogs[test.index]
		if dialog.Level != test.level {
			t.Fatalf("dialog %d: expected level %q, got %q", test.index, test.level, dialog.Level)
		}
		if len(dialog.Actions) != len(test.actions) {
			t.Fatalf("dialog %d: expected actions %#v, got %#v", test.index, test.actions, dialog.Actions)
		}
		for i, action := range test.actions {
			got := dialog.Actions[i]
			got.URL = ""
			got.Method = ""
			if got != action {
				t.Fatalf("dialog %d action %d: expected %#v, got %#v", test.index, i, action, dialog.Actions[i])
			}
		}
	}
	for _, index := range []int{4, 5, 6} {
		for _, action := range ctx.Dialogs[index].Actions {
			if action.URL == "" || action.Method != "POST" {
				t.Fatalf("dialog %d: expected callback route on action, got %#v", index, action)
			}
		}
	}
}

func TestDialogCallbackRouteInvokesHandlerWithActionValue(t *testing.T) {
	var selected string
	ctx := NewRuntimeContext(&RequestContext{PageKey: "test.form"})
	ctx.ShowYesNo("Confirm", "Continue?", func(value string) {
		selected = value
	})
	if len(ctx.Dialogs) != 1 || len(ctx.Dialogs[0].Actions) != 2 {
		t.Fatalf("expected yes/no dialog, got %#v", ctx.Dialogs)
	}
	action := ctx.Dialogs[0].Actions[0]
	if action.URL == "" {
		t.Fatalf("expected callback URL, got %#v", action)
	}

	dialogID := strings.TrimPrefix(action.URL, "/event/test.form/dialog/")
	result, err := handleDialogCallback(&RequestContext{
		Params: Params{"dialog": dialogID},
		Body:   []byte(`{"value":"yes"}`),
	})
	if err != nil {
		t.Fatalf("dialog callback returned error: %v", err)
	}
	if result == nil {
		t.Fatal("expected runtime result")
	}
	if selected != "yes" {
		t.Fatalf("expected selected value yes, got %q", selected)
	}

	_, err = handleDialogCallback(&RequestContext{
		Params: Params{"dialog": dialogID},
		Body:   []byte(`{"value":"yes"}`),
	})
	if err == nil {
		t.Fatal("expected one-shot dialog handler to be removed")
	}
}

func TestNavigationCallbackRouteIsGeneratedAndDispatched(t *testing.T) {
	page := &testNavigationCallbackPage{FormEngine: &FormEngine{}}
	var openHandler RouteHandler
	var callbackHandler RouteHandler
	for _, route := range page.FormEngine.Routes("test.form", page) {
		switch route.Path {
		case "/event/test.form/button/pick":
			openHandler = route.Handler
		case "/event/test.form/callback/:callback":
			callbackHandler = route.Handler
		}
	}
	if openHandler == nil {
		t.Fatal("expected pick button event route")
	}
	if callbackHandler == nil {
		t.Fatal("expected navigation callback route")
	}

	openResult, err := openHandler(&RequestContext{}, &testNavigationCallbackPage{FormEngine: &FormEngine{}})
	if err != nil {
		t.Fatalf("open handler returned error: %v", err)
	}
	runtime, ok := openResult.(*RuntimeResult)
	if !ok {
		t.Fatalf("expected runtime result, got %T", openResult)
	}
	if len(runtime.Navigation) != 1 {
		t.Fatalf("expected one navigation action, got %#v", runtime.Navigation)
	}
	action := runtime.Navigation[0]
	if action.Type != engine.NavigationOpen || action.Mode != engine.NavigationModeDialog || action.Page != "users.picker" {
		t.Fatalf("unexpected navigation action: %#v", action)
	}
	if action.Extra["group_id"] != 10 {
		t.Fatalf("expected group_id extra, got %#v", action.Extra)
	}
	if action.Callback != "/event/test.form/callback/on_user_selected" {
		t.Fatalf("expected generated callback route, got %q", action.Callback)
	}

	callbackResult, err := callbackHandler(&RequestContext{
		Params: Params{"callback": "on_user_selected"},
		Body:   []byte(`{"user_id":77}`),
	}, &testNavigationCallbackPage{FormEngine: &FormEngine{}})
	if err != nil {
		t.Fatalf("callback handler returned error: %v", err)
	}
	callbackRuntime, ok := callbackResult.(*RuntimeResult)
	if !ok {
		t.Fatalf("expected runtime result, got %T", callbackResult)
	}
	if len(callbackRuntime.Mutations) != 1 {
		t.Fatalf("expected one callback mutation, got %#v", callbackRuntime.Mutations)
	}
	if callbackRuntime.Mutations[0].Path != "controls.selected_user.value" || callbackRuntime.Mutations[0].Value != float64(77) {
		t.Fatalf("expected selected user mutation, got %#v", callbackRuntime.Mutations[0])
	}
}

func TestRuntimeResultMarshalsDialogs(t *testing.T) {
	result := engine.RuntimeResult{
		Dialogs: []engine.Dialog{
			{
				Title:       "Validation",
				Description: "Name is required",
				Level:       engine.DialogWarning,
				Actions: []engine.DialogAction{
					{Name: "Close", Value: "close"},
				},
			},
		},
	}

	body, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal runtime result: %v", err)
	}
	want := `{"dialogs":[{"title":"Validation","description":"Name is required","level":"warning","actions":[{"name":"Close","value":"close"}]}]}`
	if string(body) != want {
		t.Fatalf("unexpected runtime result JSON:\nwant %s\n got %s", want, string(body))
	}
}

func TestFormControlSetDefaultValueAcceptsAnyInputValue(t *testing.T) {
	engine := &FormEngine{}
	engine.Field(inputs.Input{Id: "flag", Type: inputs.InputTypeCheckbox})
	checkbox, err := engine.GetCheckboxById("flag")
	if err != nil {
		t.Fatalf("expected checkbox control: %v", err)
	}
	checkbox.SetDefaultValue(false)
	if checkbox.Input().DefaultValue != false {
		t.Fatalf("expected bool default value, got %#v", checkbox.Input().DefaultValue)
	}

	engine.Field(inputs.Input{Id: "count", Type: inputs.InputTypeNumber})
	number, err := engine.GetNumberById("count")
	if err != nil {
		t.Fatalf("expected number control: %v", err)
	}
	number.SetDefaultValue(12.5)
	if number.Input().DefaultValue != 12.5 {
		t.Fatalf("expected numeric default value, got %#v", number.Input().DefaultValue)
	}
}

func rootHasField(root *inputs.Container, id string) bool {
	for i := range root.Fields {
		if root.Fields[i].Id == id {
			return true
		}
	}
	return false
}

func TestTableRouteUsesRequestEngineInstance(t *testing.T) {
	bootstrapEngine := &TableEngine{}
	routes := bootstrapEngine.Routes("test.table", nil)
	if len(routes) != 1 {
		t.Fatalf("table engine routes len = %d, want 1 render route", len(routes))
	}
	route := routes[0]
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
	if len(table.Columns) != 1 || table.Columns[0].ID != "request" {
		t.Fatalf("expected request engine column, got %#v", table.Columns)
	}
	if bootstrapTable := bootstrapEngine.DSL().(TableDSL); len(bootstrapTable.Columns) != 0 {
		t.Fatalf("bootstrap engine should not own request DSL: %#v", bootstrapTable)
	}
}
