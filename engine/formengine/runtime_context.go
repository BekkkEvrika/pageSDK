package formengine

import (
	"encoding/json"
	"fmt"

	"github.com/BekkkEvrika/pageSDK/engine"
	inputs "github.com/BekkkEvrika/pageSDK/form"
)

type RuntimeNode interface {
	DSL() any
}

// RuntimeContext is used only by FormEngine event handlers.
type RuntimeContext struct {
	User       engine.User
	System     engine.SystemKeys
	Params     engine.Params
	FormState  *inputs.FormState
	Sender     *inputs.ElementState
	Mutations  []engine.Mutation
	Navigation []engine.NavigationItem
	Err        error
	formRoot   *inputs.Container
}

type RuntimeControl struct {
	ctx      *RuntimeContext
	input    *inputs.Input
	hasState bool
	state    inputs.ElementState
	Value    any
	Props    map[string]any
}

type RuntimeForm struct {
	ctx *RuntimeContext
}

type RuntimeSelect struct{ RuntimeControl }
type RuntimeDate struct{ RuntimeControl }
type RuntimeDatetime struct{ RuntimeControl }
type RuntimeText struct{ RuntimeControl }
type RuntimeNumber struct{ RuntimeControl }
type RuntimeCheckbox struct{ RuntimeControl }
type RuntimeLabel struct{ RuntimeControl }
type RuntimeSearch struct{ RuntimeControl }
type RuntimeTextarea struct{ RuntimeControl }
type RuntimeHidden struct{ RuntimeControl }
type RuntimeFile struct{ RuntimeControl }
type RuntimeButton struct{ RuntimeControl }

func NewRuntimeContext(req *engine.RequestContext) *RuntimeContext {
	params := req.Params
	if params == nil {
		params = engine.Params{}
	}
	return &RuntimeContext{
		User:   req.User,
		System: req.System,
		Params: params,
	}
}

func (ctx *RuntimeContext) Form() *RuntimeForm {
	return &RuntimeForm{ctx: ctx}
}

func (ctx *RuntimeContext) GetSelectById(id string) (*RuntimeSelect, error) {
	control, err := ctx.runtimeControlByIdAndType(id, inputs.InputTypeSelect)
	return &RuntimeSelect{RuntimeControl: *control}, err
}

func (ctx *RuntimeContext) GetDateById(id string) (*RuntimeDate, error) {
	control, err := ctx.runtimeControlByIdAndType(id, inputs.InputTypeDate)
	return &RuntimeDate{RuntimeControl: *control}, err
}

func (ctx *RuntimeContext) GetDatetimeById(id string) (*RuntimeDatetime, error) {
	control, err := ctx.runtimeControlByIdAndType(id, inputs.InputTypeDatetime)
	return &RuntimeDatetime{RuntimeControl: *control}, err
}

func (ctx *RuntimeContext) GetTextById(id string) (*RuntimeText, error) {
	control, err := ctx.runtimeControlByIdAndType(id, inputs.InputTypeText)
	return &RuntimeText{RuntimeControl: *control}, err
}

func (ctx *RuntimeContext) GetNumberById(id string) (*RuntimeNumber, error) {
	control, err := ctx.runtimeControlByIdAndType(id, inputs.InputTypeNumber)
	return &RuntimeNumber{RuntimeControl: *control}, err
}

func (ctx *RuntimeContext) GetCheckboxById(id string) (*RuntimeCheckbox, error) {
	control, err := ctx.runtimeControlByIdAndType(id, inputs.InputTypeCheckbox)
	return &RuntimeCheckbox{RuntimeControl: *control}, err
}

func (ctx *RuntimeContext) GetLabelById(id string) (*RuntimeLabel, error) {
	control, err := ctx.runtimeControlByIdAndType(id, inputs.InputTypeLabel)
	return &RuntimeLabel{RuntimeControl: *control}, err
}

func (ctx *RuntimeContext) GetSearchById(id string) (*RuntimeSearch, error) {
	control, err := ctx.runtimeControlByIdAndType(id, inputs.InputTypeSearch)
	return &RuntimeSearch{RuntimeControl: *control}, err
}

func (ctx *RuntimeContext) GetTextareaById(id string) (*RuntimeTextarea, error) {
	control, err := ctx.runtimeControlByIdAndType(id, inputs.InputTypeTextarea)
	return &RuntimeTextarea{RuntimeControl: *control}, err
}

func (ctx *RuntimeContext) GetHiddenById(id string) (*RuntimeHidden, error) {
	control, err := ctx.runtimeControlByIdAndType(id, inputs.InputTypeHidden)
	return &RuntimeHidden{RuntimeControl: *control}, err
}

func (ctx *RuntimeContext) GetFileById(id string) (*RuntimeFile, error) {
	control, err := ctx.runtimeControlByIdAndType(id, inputs.InputTypeFile)
	return &RuntimeFile{RuntimeControl: *control}, err
}

func (ctx *RuntimeContext) GetButtonById(id string) (*RuntimeButton, error) {
	control, err := ctx.runtimeControlByIdAndType(id, inputs.InputTypeButton)
	return &RuntimeButton{RuntimeControl: *control}, err
}

func (ctx *RuntimeContext) Remove(id string) {
	if _, err := ctx.runtimeInputById(id); err != nil {
		ctx.fail(err)
		return
	}
	ctx.remove("controls." + id)
}

func (ctx *RuntimeContext) OpenDialog(page string, params ...engine.Params) {
	ctx.Navigation = append(ctx.Navigation, engine.NavigationItem{Type: engine.NavigationOpenDialog, Page: page, Params: optionalParams(params)})
}

func (ctx *RuntimeContext) OpenTab(page string, params ...engine.Params) {
	ctx.Navigation = append(ctx.Navigation, engine.NavigationItem{Type: engine.NavigationOpenTab, Page: page, Params: optionalParams(params)})
}

func (ctx *RuntimeContext) Close() {
	ctx.Navigation = append(ctx.Navigation, engine.NavigationItem{Type: engine.NavigationClosePage})
}

func (ctx *RuntimeContext) CloseWithResult(result any) {
	ctx.Navigation = append(ctx.Navigation, engine.NavigationItem{Type: engine.NavigationCloseWithResult, Result: result})
}

func (ctx *RuntimeContext) SetError(err error) {
	ctx.fail(err)
}

func (ctx *RuntimeContext) Error() error {
	if ctx == nil {
		return nil
	}
	return ctx.Err
}

func (c *RuntimeControl) DSL() any {
	if c.input == nil {
		return inputs.Input{}
	}
	return *c.input
}

func (c *RuntimeControl) Input() *inputs.Input {
	return c.input
}

func (c *RuntimeControl) Element() inputs.ElementState {
	if c.hasState {
		return mergeRuntimeElementState(c.input, c.state)
	}
	if c.input == nil {
		return inputs.ElementState{
			Value: c.Value,
			Props: c.Props,
		}
	}
	return inputs.ElementState{
		Input: *c.input,
		Value: c.Value,
		Props: c.Props,
	}
}

func (c *RuntimeControl) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.Element())
}

func (c *RuntimeControl) SetLabel(label string) {
	if !c.valid() {
		return
	}
	c.input.Label = label
	c.state.Label = label
	c.ctx.update("controls."+c.input.Id+".label", label)
}

func (c *RuntimeControl) SetValue(value any) {
	if !c.valid() {
		return
	}
	c.Value = value
	c.state.Value = value
	c.ctx.update("controls."+c.input.Id+".value", value)
}

func (c *RuntimeText) SetHint(hint string) {
	if !c.valid() {
		return
	}
	c.input.Hint = hint
	c.state.Hint = hint
	c.ctx.update("controls."+c.input.Id+".hint", hint)
}

func (c *RuntimeControl) SetVisibility(visibility bool) {
	if !c.valid() {
		return
	}
	c.input.Visibility = visibility
	c.state.Visibility = visibility
	c.ctx.update("controls."+c.input.Id+".visibility", visibility)
}

func (f *RuntimeForm) Add(node any) {
	value := runtimeValue(node)
	if f.ctx.formRoot != nil {
		if input, ok := value.(inputs.Input); ok {
			f.ctx.formRoot.Fields = append(f.ctx.formRoot.Fields, input)
		}
	}
	f.ctx.add("form.controls", value)
}

func (f *RuntimeForm) Remove(id string) {
	f.ctx.Remove(id)
}

func runtimeValue(value any) any {
	if node, ok := value.(RuntimeNode); ok {
		return node.DSL()
	}
	return value
}

func (ctx *RuntimeContext) BindFormTree(root *inputs.Container) {
	ctx.formRoot = root
}

func (ctx *RuntimeContext) runtimeControlByIdAndType(id, inputType string) (*RuntimeControl, error) {
	input, err := ctx.runtimeInputById(id)
	if err != nil {
		ctx.fail(err)
		return newDetachedRuntimeControl(ctx, id, inputType), err
	}
	if input.Type != inputType {
		err := fmt.Errorf("runtime context: input %q has type %q, expected %q", id, input.Type, inputType)
		ctx.fail(err)
		return newDetachedRuntimeControl(ctx, id, inputType), err
	}
	control := &RuntimeControl{ctx: ctx, input: input, Value: defaultRuntimeValue(input)}
	if element, ok := ctx.elementStateByID(id); ok {
		applyRuntimeElementState(control, element)
	}
	return control, nil
}

func (ctx *RuntimeContext) elementStateByID(id string) (inputs.ElementState, bool) {
	if ctx.FormState == nil {
		return inputs.ElementState{}, false
	}
	for _, element := range ctx.FormState.Elements {
		if element.Id == id {
			return element, true
		}
	}
	if ctx.FormState.Fields != nil {
		field, ok := ctx.FormState.Fields[id]
		return field, ok
	}
	return inputs.ElementState{}, false
}

func (ctx *RuntimeContext) runtimeInputById(id string) (*inputs.Input, error) {
	if ctx.formRoot == nil {
		return nil, fmt.Errorf("runtime context: form tree is not bound")
	}
	if input := findInputByIdInContainer(ctx.formRoot, id); input != nil {
		return input, nil
	}
	return nil, fmt.Errorf("runtime context: input %q not found in DSL", id)
}

func findInputByIdInContainer(container *inputs.Container, id string) *inputs.Input {
	for fieldIndex := range container.Fields {
		if container.Fields[fieldIndex].Id == id {
			return &container.Fields[fieldIndex]
		}
	}
	for containerIndex := range container.Containers {
		if input := findInputByIdInContainer(&container.Containers[containerIndex], id); input != nil {
			return input
		}
	}
	return nil
}

func (c *RuntimeControl) valid() bool {
	if c == nil || c.ctx == nil {
		return false
	}
	if c.input == nil {
		c.ctx.fail(fmt.Errorf("runtime context: control is not bound to DSL input"))
		return false
	}
	if _, err := c.ctx.runtimeInputById(c.input.Id); err != nil {
		c.ctx.fail(err)
		return false
	}
	return true
}

func (ctx *RuntimeContext) fail(err error) {
	if err != nil && ctx.Err == nil {
		ctx.Err = err
	}
}

func newDetachedRuntimeControl(ctx *RuntimeContext, id, inputType string) *RuntimeControl {
	return &RuntimeControl{
		ctx:   ctx,
		input: &inputs.Input{Id: id, Type: inputType},
	}
}

func applyRuntimeElementState(control *RuntimeControl, element inputs.ElementState) {
	if element.Value == nil {
		element.Value = defaultRuntimeValue(control.input)
	}
	control.hasState = true
	control.state = element
	control.Value = element.Value
	control.Props = element.Props
}

func defaultRuntimeValue(input *inputs.Input) any {
	if input == nil {
		return nil
	}
	if input.DefaultValue != nil {
		return input.DefaultValue
	}
	switch input.Type {
	case inputs.InputTypeCheckbox:
		return false
	case inputs.InputTypeNumber:
		return 0
	case inputs.InputTypeButton:
		return false
	case inputs.InputTypeText,
		inputs.InputTypeSearch,
		inputs.InputTypeTextarea,
		inputs.InputTypeHidden,
		inputs.InputTypeDate,
		inputs.InputTypeDatetime,
		inputs.InputTypeLabel:
		return ""
	default:
		return nil
	}
}

func mergeRuntimeElementState(input *inputs.Input, state inputs.ElementState) inputs.ElementState {
	if input == nil {
		return state
	}
	if state.Id == "" {
		state.Id = input.Id
	}
	if state.Type == "" {
		state.Type = input.Type
	}
	if state.Label == "" {
		state.Label = input.Label
	}
	if state.Name == "" {
		state.Name = input.Name
	}
	if state.ActionID == "" {
		state.ActionID = input.ActionID
	}
	return state
}

func (ctx *RuntimeContext) update(path string, value any) {
	ctx.Mutations = append(ctx.Mutations, engine.Mutation{Type: engine.MutationUpdate, Path: path, Value: value})
}

func (ctx *RuntimeContext) add(path string, value any) {
	ctx.Mutations = append(ctx.Mutations, engine.Mutation{Type: engine.MutationAdd, Path: path, Value: value})
}

func (ctx *RuntimeContext) remove(path string) {
	ctx.Mutations = append(ctx.Mutations, engine.Mutation{Type: engine.MutationRemove, Path: path})
}

func optionalParams(params []engine.Params) engine.Params {
	if len(params) == 0 {
		return nil
	}
	return params[0]
}
