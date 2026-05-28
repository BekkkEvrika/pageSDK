package engine

import (
	"encoding/json"

	inputs "github.com/behzod/pageSDK/form"
)

type RuntimeNode interface {
	DSL() any
}

type RuntimeControl struct {
	ctx   *RuntimeContext
	input inputs.Input
}

type RuntimeForm struct {
	ctx *RuntimeContext
}

func (ctx *RuntimeContext) SetState(key string, value any) {
	if ctx.State == nil {
		ctx.State = map[string]any{}
	}
	ctx.State[key] = value
	ctx.update("state."+key, value)
}

func (ctx *RuntimeContext) Text(id string) *RuntimeControl {
	return &RuntimeControl{
		ctx:   ctx,
		input: inputs.Input{Id: id, Type: inputs.InputTypeText},
	}
}

func (ctx *RuntimeContext) Form() *RuntimeForm {
	return &RuntimeForm{ctx: ctx}
}

func (ctx *RuntimeContext) Remove(id string) {
	ctx.remove("controls." + id)
}

func (ctx *RuntimeContext) OpenDialog(page string, params ...Params) {
	ctx.Navigation = append(ctx.Navigation, NavigationItem{Type: NavigationOpenDialog, Page: page, Params: optionalParams(params)})
}

func (ctx *RuntimeContext) OpenTab(page string, params ...Params) {
	ctx.Navigation = append(ctx.Navigation, NavigationItem{Type: NavigationOpenTab, Page: page, Params: optionalParams(params)})
}

func (ctx *RuntimeContext) Close() {
	ctx.Navigation = append(ctx.Navigation, NavigationItem{Type: NavigationClosePage})
}

func (ctx *RuntimeContext) CloseWithResult(result any) {
	ctx.Navigation = append(ctx.Navigation, NavigationItem{Type: NavigationCloseWithResult, Result: result})
}

func (c *RuntimeControl) DSL() any {
	return c.input
}

func (c *RuntimeControl) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.input)
}

func (c *RuntimeControl) SetText(text string) {
	c.input.Label = text
	c.ctx.update("controls."+c.input.Id+".text", text)
}

func (c *RuntimeControl) SetLabel(label string) {
	c.input.Label = label
	c.ctx.update("controls."+c.input.Id+".label", label)
}

func (c *RuntimeControl) SetValue(value any) {
	c.ctx.update("controls."+c.input.Id+".value", value)
}

func (c *RuntimeControl) SetVisible(visible bool) {
	c.ctx.update("controls."+c.input.Id+".visible", visible)
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

func (ctx *RuntimeContext) bindFormTree(root *inputs.Container) {
	ctx.formRoot = root
}

func (ctx *RuntimeContext) update(path string, value any) {
	ctx.Mutations = append(ctx.Mutations, Mutation{Type: MutationUpdate, Path: path, Value: value})
}

func (ctx *RuntimeContext) add(path string, value any) {
	ctx.Mutations = append(ctx.Mutations, Mutation{Type: MutationAdd, Path: path, Value: value})
}

func (ctx *RuntimeContext) remove(path string) {
	ctx.Mutations = append(ctx.Mutations, Mutation{Type: MutationRemove, Path: path})
}

func optionalParams(params []Params) Params {
	if len(params) == 0 {
		return nil
	}
	return params[0]
}
