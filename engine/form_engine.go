package engine

import (
	"encoding/json"
	"errors"
	"net/http"

	inputs "github.com/behzod/pageSDK/form"
)

// FormEngine — движок для form-based pages.
// Реализует Engine interface.
// Отвечает за routing semantics форм: render + event handling.
//
// Генерируемые routes:
//
//	GET  /page/{key}                      — рендер формы (DSL)
//	POST /event/{key}/{component}/{action} — обработка событий формы
type FormEngine struct{}

// ID возвращает identifier движка.
func (f *FormEngine) ID() string {
	return "form"
}

// Routes возвращает routes для form page.
// Вызывается один раз во время Bootstrap — детерминировано, без side effects.
func (f *FormEngine) Routes(pageKey string, page Page) []RouteDefinition {
	return []RouteDefinition{
		{
			Method:  http.MethodGet,
			Path:    "/page/" + pageKey,
			Handler: f.renderRoute(pageKey),
		},
		{
			Method:  http.MethodPost,
			Path:    "/event/" + pageKey + "/:component/:action",
			Handler: f.handleRoute(pageKey),
		},
	}
}

// Render создаёт DSL формы.
func (f *FormEngine) Render(ctx *RequestContext, page Page) (*RenderResult, error) {
	if err := page.Init(ctx.BuildContext()); err != nil {
		return nil, err
	}

	dslPage, ok := page.(DSLPage)
	if !ok {
		return nil, errors.New("form engine: page must implement DSL()")
	}

	return &RenderResult{
		PageKey: ctx.PageKey,
		Engine:  f.ID(),
		DSL:     dslPage.DSL(),
	}, nil
}

// Handle обрабатывает runtime events формы.
func (f *FormEngine) Handle(ctx *RequestContext, page Page) (*RuntimeResult, error) {
	if err := page.Init(ctx.BuildContext()); err != nil {
		return nil, err
	}

	runtimeCtx := ctx.RuntimeContext()
	handler, ok := page.(EventHandler)
	if ok {
		state, err := formState(ctx)
		if err != nil {
			return nil, err
		}
		event := Event{
			Component: ctx.Params["component"],
			Action:    ctx.Params["action"],
			Payload:   state,
		}
		if err := handler.HandleEvent(runtimeCtx, event); err != nil {
			return nil, err
		}
	}

	return &RuntimeResult{
		Mutations:  runtimeCtx.Mutations.Items(),
		Navigation: runtimeCtx.Navigation.Items(),
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
		return f.Render(ctx, page)
	}
}

func (f *FormEngine) handleRoute(pageKey string) RouteHandler {
	return func(ctx *RequestContext, page Page) (any, error) {
		ctx.PageKey = pageKey
		return f.Handle(ctx, page)
	}
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
	return state, nil
}
