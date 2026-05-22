package engine

import (
	"errors"
	"net/http"
)

// TableEngine — движок для table/list pages.
// Реализует Engine interface.
//
// Генерируемые routes:
//
//	GET  /page/{key}                       — рендер таблицы (DSL + данные)
//	POST /event/{key}/{component}/{action} — обработка событий таблицы
type TableEngine struct{}

// ID возвращает identifier движка.
func (t *TableEngine) ID() string {
	return "table"
}

// Routes возвращает routes для table page.
func (t *TableEngine) Routes(pageKey string, page Page) []RouteDefinition {
	return []RouteDefinition{
		{
			Method:  http.MethodGet,
			Path:    "/page/" + pageKey,
			Handler: t.renderRoute(pageKey),
		},
		{
			Method:  http.MethodPost,
			Path:    "/event/" + pageKey + "/:component/:action",
			Handler: t.handleRoute(pageKey),
		},
	}
}

// Render создаёт DSL таблицы.
func (t *TableEngine) Render(ctx *RequestContext, page Page) (*RenderResult, error) {
	if err := page.Init(ctx.BuildContext()); err != nil {
		return nil, err
	}

	dslPage, ok := page.(DSLPage)
	if !ok {
		return nil, errors.New("table engine: page must implement DSL()")
	}

	return &RenderResult{
		PageKey: ctx.PageKey,
		Engine:  t.ID(),
		DSL:     dslPage.DSL(),
	}, nil
}

// Handle обрабатывает runtime events таблицы.
func (t *TableEngine) Handle(ctx *RequestContext, page Page) (*RuntimeResult, error) {
	if err := page.Init(ctx.BuildContext()); err != nil {
		return nil, err
	}

	runtimeCtx := ctx.RuntimeContext()
	handler, ok := page.(EventHandler)
	if ok {
		event := Event{
			Component: ctx.Params["component"],
			Action:    ctx.Params["action"],
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

// GetEngine реализует Page interface через embedding.
func (t *TableEngine) GetEngine() Engine {
	return t
}

func (t *TableEngine) renderRoute(pageKey string) RouteHandler {
	return func(ctx *RequestContext, page Page) (any, error) {
		ctx.PageKey = pageKey
		return t.Render(ctx, page)
	}
}

func (t *TableEngine) handleRoute(pageKey string) RouteHandler {
	return func(ctx *RequestContext, page Page) (any, error) {
		ctx.PageKey = pageKey
		return t.Handle(ctx, page)
	}
}
