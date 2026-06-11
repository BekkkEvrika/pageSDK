package tableengine

import "github.com/BekkkEvrika/pageSDK/engine"

// RuntimeContext is used only by TableEngine event handlers.
type RuntimeContext struct {
	User       engine.User
	System     engine.SystemKeys
	Params     engine.Params
	Mutations  []engine.Mutation
	Navigation []engine.NavigationItem
	Err        error
}

// Event describes an explicit table runtime event.
type Event struct {
	Component string
	Action    string
	Payload   any
}

// EventHandler can be implemented by table pages that support runtime events.
type EventHandler interface {
	HandleEvent(ctx *RuntimeContext, event Event) error
}

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

func (ctx *RuntimeContext) SetError(err error) {
	if err != nil && ctx.Err == nil {
		ctx.Err = err
	}
}

func (ctx *RuntimeContext) Error() error {
	if ctx == nil {
		return nil
	}
	return ctx.Err
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

func optionalParams(params []engine.Params) engine.Params {
	if len(params) == 0 {
		return nil
	}
	return params[0]
}
