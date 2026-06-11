package engine

import (
	"net/http"
)

// TableEngine — движок для table/list pages.
// Реализует Engine interface.
//
// Генерируемые routes:
//
//	GET  /page/{key}                       — рендер таблицы (DSL + данные)
//	POST /event/{key}/{component}/{action} — обработка событий таблицы
type TableEngine struct {
	dsl TableDSL
}

type TableDSL struct {
	Columns []TableColumn `json:"columns,omitempty"`
}

type TableColumn struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

// SetDSL replaces the current table DSL owned by this engine instance.
func (t *TableEngine) SetDSL(dsl TableDSL) {
	t.dsl = dsl
}

// Column appends a table column to this engine instance.
func (t *TableEngine) Column(key, label string) {
	t.dsl.Columns = append(t.dsl.Columns, TableColumn{Key: key, Label: label})
}

// DSL returns the table DSL owned by this engine instance.
func (t *TableEngine) DSL() any {
	return t.dsl
}

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

	return &RenderResult{
		PageKey: ctx.PageKey,
		Engine:  t.ID(),
		DSL:     t.DSL(),
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
		if err := runtimeCtx.Error(); err != nil {
			return nil, err
		}
	}

	return &RuntimeResult{
		Mutations:  runtimeCtx.Mutations,
		Navigation: runtimeCtx.Navigation,
	}, nil
}

// GetEngine реализует Page interface через embedding.
func (t *TableEngine) GetEngine() Engine {
	return t
}

func (t *TableEngine) renderRoute(pageKey string) RouteHandler {
	return func(ctx *RequestContext, page Page) (any, error) {
		ctx.PageKey = pageKey
		return page.GetEngine().Render(ctx, page)
	}
}

func (t *TableEngine) handleRoute(pageKey string) RouteHandler {
	return func(ctx *RequestContext, page Page) (any, error) {
		ctx.PageKey = pageKey
		return page.GetEngine().Handle(ctx, page)
	}
}
