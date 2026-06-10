package engine

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	inputs "github.com/BekkkEvrika/pageSDK/form"
)

// FormEngine — движок для form-based pages.
// Реализует Engine interface.
// Отвечает за routing semantics форм: render + event handling.
//
// Генерируемые routes:
//
//	GET  /page/{key}                         — рендер формы (DSL)
//	POST /event/{key}/{component}/{actionID} — static routes for registered listeners
type FormEngine struct {
	root        inputs.Container
	formActions []inputs.FormAction
	components  map[string]formComponent
	handlers    map[string]map[string]formEventHandler
	eventRoutes []formEventRoute
}

type formComponent struct {
	ID       string
	Type     string
	ParentID string
	Path     string
}

type formEventKey struct {
	Component string
	Action    string
}

type formEventHandler func(ctx *RuntimeContext)

type formEventRoute struct {
	Key    formEventKey
	Method string
	Path   string
}

// CreateForm stores the form DSL inside this engine instance.
func (f *FormEngine) CreateForm(form inputs.Form) {
	f.root = newRootContainer()
	f.formActions = nil
	f.components = map[string]formComponent{}
	if form.Containers != nil {
		f.root.Containers = append(f.root.Containers, (*form.Containers)...)
		f.registerComponentsInContainers(f.root.Containers, f.root.Key, f.root.Key)
	}
	if form.FormActions != nil {
		f.formActions = append(f.formActions, (*form.FormActions)...)
	}
}

// SetForm replaces the current form DSL owned by this engine instance.
func (f *FormEngine) SetForm(form inputs.Form) {
	f.CreateForm(form)
}

// Container adds a top-level form container to this engine instance.
func (f *FormEngine) Container(container inputs.Container) {
	containers := f.containers()
	*containers = append(*containers, container)
	f.registerContainer((*containers)[len(*containers)-1], f.root.Key, f.root.Key)
}

// Field appends an input to the default container.
func (f *FormEngine) Field(input inputs.Input) {
	container := f.defaultContainer()
	container.Fields = append(container.Fields, input)
	f.registerComponent(input, container.Key, containerPath(f.root.Key, container.Key))
}

// Text appends a text input to the default container.
func (f *FormEngine) Text(id string) *Text {
	f.Field(inputs.Input{Id: id, Type: inputs.InputTypeText, Label: id})
	text, _ := f.GetTextById(id)
	return text
}

// Button appends a button field and click form action to this engine instance.
func (f *FormEngine) Button(actionID string) *Button {
	f.Field(inputs.Input{Id: actionID, Type: inputs.InputTypeButton, Label: actionID, ActionID: actionID})
	f.Action(inputs.FormAction{
		ID:      actionID,
		Trigger: inputs.Click,
		Config: &inputs.FormActionConfig{
			Type:   inputs.APICall,
			URL:    eventRoutePath("", inputs.InputTypeButton, actionID),
			Method: http.MethodPost,
		},
	})
	button, _ := f.GetButtonById(actionID)
	return button
}

// Action appends a form action to this engine instance.
func (f *FormEngine) Action(action inputs.FormAction) {
	actions := f.actions()
	*actions = append(*actions, action)
}

// DSL returns the form DSL owned by this engine instance.
func (f *FormEngine) DSL() any {
	if f.root.Key == "" {
		return inputs.Form{}
	}
	form := inputs.Form{}
	if f.root.Containers != nil {
		form.Containers = &f.root.Containers
	}
	if f.formActions != nil {
		form.FormActions = &f.formActions
	}
	return form
}

// GetInputById returns any input field by id from this engine instance.
func (f *FormEngine) GetInputById(id string) (*inputs.Input, error) {
	input := f.findInputById(id)
	if input == nil {
		return nil, fmt.Errorf("form engine: input %q not found", id)
	}
	return input, nil
}

// GetSelectById returns a select input by id from this engine instance.
func (f *FormEngine) GetSelectById(id string) (*Select, error) {
	input, err := f.getInputByIdAndType(id, inputs.InputTypeSelect)
	if err != nil {
		return nil, err
	}
	return newSelect(f, input), nil
}

// GetDateById returns a date input by id from this engine instance.
func (f *FormEngine) GetDateById(id string) (*Date, error) {
	input, err := f.getInputByIdAndType(id, inputs.InputTypeDate)
	if err != nil {
		return nil, err
	}
	return newDate(f, input), nil
}

// GetDatetimeById returns a datetime input by id from this engine instance.
func (f *FormEngine) GetDatetimeById(id string) (*Datetime, error) {
	input, err := f.getInputByIdAndType(id, inputs.InputTypeDatetime)
	if err != nil {
		return nil, err
	}
	return newDatetime(f, input), nil
}

// GetTextById returns a text input by id from this engine instance.
func (f *FormEngine) GetTextById(id string) (*Text, error) {
	input, err := f.getInputByIdAndType(id, inputs.InputTypeText)
	if err != nil {
		return nil, err
	}
	return newText(f, input), nil
}

// GetNumberById returns a number input by id from this engine instance.
func (f *FormEngine) GetNumberById(id string) (*Number, error) {
	input, err := f.getInputByIdAndType(id, inputs.InputTypeNumber)
	if err != nil {
		return nil, err
	}
	return newNumber(f, input), nil
}

// GetCheckboxById returns a checkbox input by id from this engine instance.
func (f *FormEngine) GetCheckboxById(id string) (*Checkbox, error) {
	input, err := f.getInputByIdAndType(id, inputs.InputTypeCheckbox)
	if err != nil {
		return nil, err
	}
	return newCheckbox(f, input), nil
}

// GetLabelById returns a label input by id from this engine instance.
func (f *FormEngine) GetLabelById(id string) (*Label, error) {
	input, err := f.getInputByIdAndType(id, inputs.InputTypeLabel)
	if err != nil {
		return nil, err
	}
	return newLabel(f, input), nil
}

// GetSearchById returns a search input by id from this engine instance.
func (f *FormEngine) GetSearchById(id string) (*Search, error) {
	input, err := f.getInputByIdAndType(id, inputs.InputTypeSearch)
	if err != nil {
		return nil, err
	}
	return newSearch(f, input), nil
}

// GetTextareaById returns a textarea input by id from this engine instance.
func (f *FormEngine) GetTextareaById(id string) (*Textarea, error) {
	input, err := f.getInputByIdAndType(id, inputs.InputTypeTextarea)
	if err != nil {
		return nil, err
	}
	return newTextarea(f, input), nil
}

// GetHiddenById returns a hidden input by id from this engine instance.
func (f *FormEngine) GetHiddenById(id string) (*Hidden, error) {
	input, err := f.getInputByIdAndType(id, inputs.InputTypeHidden)
	if err != nil {
		return nil, err
	}
	return newHidden(f, input), nil
}

// GetFileById returns a file input by id from this engine instance.
func (f *FormEngine) GetFileById(id string) (*File, error) {
	input, err := f.getInputByIdAndType(id, inputs.InputTypeFile)
	if err != nil {
		return nil, err
	}
	return newFile(f, input), nil
}

// GetButtonById returns a button input by id from this engine instance.
func (f *FormEngine) GetButtonById(id string) (*Button, error) {
	input, err := f.getInputByIdAndType(id, inputs.InputTypeButton)
	if err != nil {
		return nil, err
	}
	return newButton(f, input), nil
}

// ID возвращает identifier движка.
func (f *FormEngine) ID() string {
	return "form"
}

// Routes возвращает static routes для form page.
// Вызывается один раз во время Bootstrap после Init() sample page.
func (f *FormEngine) Routes(pageKey string, page Page) []RouteDefinition {
	if page != nil {
		if err := page.Init(&BuildContext{}); err != nil {
			panic("form engine: init page " + pageKey + ": " + err.Error())
		}
	}

	routes := []RouteDefinition{
		{
			Method:  http.MethodGet,
			Path:    "/page/" + pageKey,
			Handler: f.renderRoute(pageKey),
		},
	}
	f.generateEventRoutes(pageKey)
	for _, route := range f.eventRoutes {
		eventKey := route.Key
		routes = append(routes, RouteDefinition{
			Method:  route.Method,
			Path:    route.Path,
			Handler: f.handleRoute(pageKey, eventKey),
		})
	}
	return routes
}

// Render создаёт DSL формы.
func (f *FormEngine) Render(ctx *RequestContext, page Page) (*RenderResult, error) {
	if err := page.Init(ctx.BuildContext()); err != nil {
		return nil, err
	}
	f.bindFormActionRoutes(ctx.PageKey)

	return &RenderResult{
		PageKey: ctx.PageKey,
		Engine:  f.ID(),
		DSL:     f.DSL(),
	}, nil
}

// Handle обрабатывает runtime events формы.
func (f *FormEngine) Handle(ctx *RequestContext, page Page) (*RuntimeResult, error) {
	if err := page.Init(ctx.BuildContext()); err != nil {
		return nil, err
	}

	state, err := formState(ctx)
	if err != nil {
		return nil, err
	}
	runtimeCtx := ctx.RuntimeContext()
	runtimeCtx.FormState = state
	runtimeCtx.Sender = state.Sender
	runtimeCtx.bindFormTree(&f.root)
	runtimeCtx.Params["form.actionId"] = state.ActionID
	handler := f.handler(formEventKey{
		Component: ctx.Params["component"],
		Action:    ctx.Params["action"],
	})
	if handler == nil {
		return nil, fmt.Errorf("form engine: handler for %q/%q not found", ctx.Params["component"], ctx.Params["action"])
	}
	handler(runtimeCtx)
	if runtimeCtx.err != nil {
		return nil, runtimeCtx.err
	}

	return &RuntimeResult{
		Mutations:  runtimeCtx.Mutations,
		Navigation: runtimeCtx.Navigation,
	}, nil
}

// GetEngine реализует Page interface — возвращает себя как Engine.
// Встраивается в конкретные page structs через embedding.
func (f *FormEngine) GetEngine() Engine {
	return f
}

func (f *FormEngine) renderRoute(pageKey string) RouteHandler {
	return func(ctx *RequestContext, page Page) (any, error) {
		ctx.PageKey = pageKey
		return page.GetEngine().Render(ctx, page)
	}
}

func (f *FormEngine) handleRoute(pageKey string, eventKey formEventKey) RouteHandler {
	return func(ctx *RequestContext, page Page) (any, error) {
		ctx.PageKey = pageKey
		if ctx.Params == nil {
			ctx.Params = Params{}
		}
		ctx.Params["component"] = eventKey.Component
		ctx.Params["action"] = eventKey.Action
		return page.GetEngine().Handle(ctx, page)
	}
}

func (f *FormEngine) containers() *[]inputs.Container {
	f.ensureRoot()
	return &f.root.Containers
}

func (f *FormEngine) actions() *[]inputs.FormAction {
	return &f.formActions
}

func (f *FormEngine) defaultContainer() *inputs.Container {
	containers := f.containers()
	if len(*containers) == 0 {
		*containers = append(*containers, inputs.Container{
			Key:       "main",
			Direction: "vertical",
			Gap:       16,
		})
	}
	return &(*containers)[0]
}

func (f *FormEngine) registerFormEvent(component, action string, trigger inputs.FormActionTrigger, input *inputs.Input, handler formEventHandler) {
	f.registerComponent(*input, "", "")
	f.upsertGeneratedFormAction(component, action, trigger, "")
	if f.handlers == nil {
		f.handlers = map[string]map[string]formEventHandler{}
	}
	if f.handlers[component] == nil {
		f.handlers[component] = map[string]formEventHandler{}
	}
	f.handlers[component][action] = handler
}

func (f *FormEngine) handler(key formEventKey) formEventHandler {
	if f.handlers == nil {
		return nil
	}
	return f.handlers[key.Component][key.Action]
}

func (f *FormEngine) eventKeys() []formEventKey {
	var keys []formEventKey
	for component, actions := range f.handlers {
		for action := range actions {
			keys = append(keys, formEventKey{Component: component, Action: action})
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Component == keys[j].Component {
			return keys[i].Action < keys[j].Action
		}
		return keys[i].Component < keys[j].Component
	})
	return keys
}

func (f *FormEngine) generateEventRoutes(pageKey string) {
	f.eventRoutes = f.eventRoutes[:0]
	for _, key := range f.eventKeys() {
		f.eventRoutes = append(f.eventRoutes, formEventRoute{
			Key:    key,
			Method: http.MethodPost,
			Path:   "/event/" + pageKey + "/" + key.Component + "/" + key.Action,
		})
	}
}

func (f *FormEngine) upsertGeneratedFormAction(component, action string, trigger inputs.FormActionTrigger, pageKey string) {
	generated := inputs.FormAction{
		ID:      action,
		Trigger: trigger,
		Config: &inputs.FormActionConfig{
			Type:   formActionType(trigger),
			URL:    eventRoutePath(pageKey, component, action),
			Method: http.MethodPost,
		},
	}
	found := false
	actions := f.formActions[:0]
	for i := range f.formActions {
		if f.formActions[i].ID == generated.ID && f.formActions[i].Trigger == generated.Trigger {
			if !found {
				actions = append(actions, generated)
				found = true
			}
			continue
		}
		actions = append(actions, f.formActions[i])
	}
	if !found {
		actions = append(actions, generated)
	}
	f.formActions = actions
}

func (f *FormEngine) bindFormActionRoutes(pageKey string) {
	if pageKey == "" {
		return
	}
	for _, key := range f.eventKeys() {
		trigger, ok := f.formActionTrigger(key)
		if !ok {
			continue
		}
		f.upsertGeneratedFormAction(key.Component, key.Action, trigger, pageKey)
	}
}

func (f *FormEngine) formActionTrigger(key formEventKey) (inputs.FormActionTrigger, bool) {
	for _, action := range f.formActions {
		if action.ID == key.Action && action.Config != nil && strings.HasSuffix(action.Config.URL, "/"+key.Component+"/"+key.Action) {
			return action.Trigger, true
		}
	}
	return "", false
}

func formActionType(trigger inputs.FormActionTrigger) inputs.FormActionType {
	if trigger == inputs.Change {
		return inputs.ChangeAPICall
	}
	return inputs.APICall
}

func eventRoutePath(pageKey, component, action string) string {
	if pageKey == "" {
		pageKey = "{page}"
	}
	return "/event/" + pageKey + "/" + component + "/" + action
}

func (f *FormEngine) registerComponent(input inputs.Input, parentID, parentPath string) {
	if input.Id == "" {
		return
	}
	if f.components == nil {
		f.components = map[string]formComponent{}
	}
	path := input.Id
	if parentPath != "" {
		path = parentPath + "." + input.Id
	} else if existing, ok := f.components[input.Id]; ok {
		parentID = existing.ParentID
		path = existing.Path
	}
	f.components[input.Id] = formComponent{ID: input.Id, Type: input.Type, ParentID: parentID, Path: path}
}

func (f *FormEngine) registerContainer(container inputs.Container, parentID, parentPath string) {
	path := containerPath(parentPath, container.Key)
	for _, field := range container.Fields {
		f.registerComponent(field, container.Key, path)
	}
	f.registerComponentsInContainers(container.Containers, container.Key, path)
}

func (f *FormEngine) registerComponentsInContainers(containers []inputs.Container, parentID, parentPath string) {
	for _, container := range containers {
		f.registerContainer(container, parentID, parentPath)
	}
}

func (f *FormEngine) ensureRoot() {
	if f.root.Key == "" {
		f.root = newRootContainer()
	}
}

func newRootContainer() inputs.Container {
	return inputs.Container{
		Key:       "root",
		Direction: "vertical",
	}
}

func containerPath(parentPath, containerID string) string {
	if parentPath == "" {
		return containerID
	}
	if containerID == "" {
		return parentPath
	}
	return parentPath + "." + containerID
}

func (f *FormEngine) getInputByIdAndType(id, inputType string) (*inputs.Input, error) {
	input, err := f.GetInputById(id)
	if err != nil {
		return nil, err
	}
	if input.Type != inputType {
		return nil, fmt.Errorf("form engine: input %q has type %q, expected %q", id, input.Type, inputType)
	}
	return input, nil
}

func (f *FormEngine) findInputById(id string) *inputs.Input {
	if f.root.Key == "" {
		return nil
	}
	return findInputByIdInContainers(f.root.Containers, id)
}

func findInputByIdInContainers(containers []inputs.Container, id string) *inputs.Input {
	for containerIndex := range containers {
		container := &containers[containerIndex]
		for fieldIndex := range container.Fields {
			if container.Fields[fieldIndex].Id == id {
				return &container.Fields[fieldIndex]
			}
		}
		if input := findInputByIdInContainers(container.Containers, id); input != nil {
			return input
		}
	}
	return nil
}

func formState(ctx *RequestContext) (*inputs.FormState, error) {
	state := &inputs.FormState{}
	if len(ctx.Body) > 0 {
		if err := json.Unmarshal(ctx.Body, state); err != nil {
			return nil, err
		}
	}
	if state.ActionID == "" {
		state.ActionID = ctx.Params["action"]
	}
	normalizeFormState(state)
	return state, nil
}

func normalizeFormState(state *inputs.FormState) {
	if len(state.Elements) == 0 && len(state.Fields) > 0 {
		keys := make([]string, 0, len(state.Fields))
		for key := range state.Fields {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			element := state.Fields[key]
			if element.Id == "" {
				element.Id = key
			}
			state.Elements = append(state.Elements, element)
		}
	}
	if state.Sender != nil {
		return
	}
	senderID := state.ChangedField
	if senderID == "" {
		senderID = state.ActionID
	}
	if senderID == "" {
		return
	}
	for i := range state.Elements {
		if state.Elements[i].Id == senderID {
			state.Sender = &state.Elements[i]
			return
		}
	}
}
