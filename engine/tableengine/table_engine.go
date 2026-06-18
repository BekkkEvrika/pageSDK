package tableengine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"

	"github.com/BekkkEvrika/pageSDK/engine"
	"github.com/BekkkEvrika/pageSDK/table"
)

// TableEngine — движок для table/list pages.
// Реализует Engine interface.
//
// Генерируемые routes:
//
//	GET  /page/{key}                       — рендер таблицы (DSL + данные)
type TableEngine struct {
	dsl      table.TableSchema
	tableID  string
	handlers map[tableEventKey]table.TableEventHandler
}

type tableEventKey struct {
	TableID string
	Event   table.TableEventType
}

// TableDSL is kept for compatibility. New code should use table.TableSchema.
type TableDSL = table.TableSchema

// TableColumn is kept for compatibility. New code should use table.TableColumnSchema.
type TableColumn = table.TableColumnSchema

// SetDSL replaces the current table DSL owned by this engine instance.
func (t *TableEngine) SetDSL(dsl table.TableSchema) {
	t.dsl = dsl
}

// CreateTable stores the table DSL inside this engine instance.
func (t *TableEngine) CreateTable(schema table.TableSchema) {
	t.SetDSL(schema)
}

// Table starts a table DSL definition and returns its fluent builder.
func (t *TableEngine) Table(name ...string) *table.Builder {
	t.dsl = table.New(name...)
	t.tableID = ""
	if len(name) > 0 {
		t.tableID = name[0]
	}
	return table.NewBuilder(&t.dsl).Runtime(t.tableID, t)
}

// Schema returns the mutable table schema owned by this engine instance.
func (t *TableEngine) Schema() *table.TableSchema {
	t.ensureDSL()
	return &t.dsl
}

// Column creates a fluent column builder.
//
// If header is passed, the column is appended immediately for compatibility
// with the older p.Column("id", "ID") style.
func (t *TableEngine) Column(id string, header ...string) *table.ColumnBuilder {
	column := table.NewColumn(id)
	if len(header) > 0 {
		column.Header(header[0])
		t.ensureDSL()
		t.dsl.Columns = append(t.dsl.Columns, column.Schema())
		return table.NewColumnBuilder(&t.dsl.Columns[len(t.dsl.Columns)-1])
	}
	return column
}

// DisplayColumn appends a display-only table column to this engine instance.
func (t *TableEngine) DisplayColumn(id string, header ...string) *table.ColumnBuilder {
	column := table.NewColumn(id).Kind(table.TableColumnKindDisplay).AccessorKey("")
	if len(header) > 0 {
		column.Header(header[0])
		t.ensureDSL()
		t.dsl.Columns = append(t.dsl.Columns, column.Schema())
		return table.NewColumnBuilder(&t.dsl.Columns[len(t.dsl.Columns)-1])
	}
	return column
}

// Columns replaces table columns.
func (t *TableEngine) Columns(columns ...*table.ColumnBuilder) *table.Builder {
	return t.builder().Columns(columns...)
}

// Features replaces the feature config.
func (t *TableEngine) Features(features table.TableFeatureConfig) *table.Builder {
	return t.builder().Features(features)
}

// Actions replaces all table action groups.
func (t *TableEngine) Actions(actions table.TableActionGroups) *table.Builder {
	return t.builder().Actions(actions)
}

// Selection replaces the selection config.
func (t *TableEngine) Selection(selection table.TableSelectionSchema) *table.Builder {
	return t.builder().Selection(selection)
}

// Hotkeys replaces table hotkeys.
func (t *TableEngine) Hotkeys(hotkeys ...table.TableHotkeySchema) *table.Builder {
	return t.builder().Hotkeys(hotkeys...)
}

// State replaces the initial table state.
func (t *TableEngine) State(state table.TableStateConfig) *table.Builder {
	return t.builder().State(state)
}

// Data replaces inline table data.
func (t *TableEngine) Data(data any) *table.Builder {
	return t.builder().Data(data)
}

// SetRows configures inline table rows.
func (t *TableEngine) SetRows(rows []map[string]any) {
	t.ensureDSL()
	t.dsl.SetRows(rows)
}

// SetData configures inline table data.
func (t *TableEngine) SetData(data table.TableData) {
	t.ensureDSL()
	t.dsl.SetData(data)
}

// AddToolbarAction appends a toolbar action.
func (t *TableEngine) AddToolbarAction(action table.ActionSchema) {
	t.ensureDSL()
	t.dsl.AddToolbarAction(action)
}

// AddRowAction appends a row action.
func (t *TableEngine) AddRowAction(action table.ActionSchema) {
	t.ensureDSL()
	t.dsl.AddRowAction(action)
}

// DSL returns the table DSL owned by this engine instance.
func (t *TableEngine) DSL() any {
	t.ensureDSL()
	return t.dsl
}

// ID возвращает identifier движка.
func (t *TableEngine) ID() string {
	return "table"
}

// Routes возвращает routes для table page.
func (t *TableEngine) Routes(pageKey string, page engine.Page) []engine.RouteDefinition {
	routes := []engine.RouteDefinition{
		{
			Method:  http.MethodGet,
			Path:    "/page/" + pageKey,
			Handler: t.renderRoute(pageKey),
		},
	}
	if page == nil {
		return routes
	}
	if err := page.Init(&engine.BuildContext{}); err != nil {
		panic("table engine: init page " + pageKey + ": " + err.Error())
	}
	for _, key := range t.eventKeys() {
		eventKey := key
		routes = append(routes, engine.RouteDefinition{
			Method:  http.MethodPost,
			Path:    tableEventRoutePath(pageKey, eventKey),
			Handler: t.handleRoute(pageKey, eventKey),
		})
	}
	return routes
}

// Render создаёт DSL таблицы.
func (t *TableEngine) Render(ctx *engine.RequestContext, page engine.Page) (*engine.RenderResult, error) {
	if err := page.Init(ctx.BuildContext()); err != nil {
		return nil, err
	}
	t.bindEventRoutes(ctx.PageKey)

	return &engine.RenderResult{
		PageKey: ctx.PageKey,
		Engine:  t.ID(),
		DSL:     t.DSL(),
	}, nil
}

// Handle dispatches one supported table runtime event.
func (t *TableEngine) Handle(ctx *engine.RequestContext, page engine.Page) (*engine.RuntimeResult, error) {
	if err := page.Init(ctx.BuildContext()); err != nil {
		return nil, err
	}

	key := tableEventKey{
		TableID: ctx.Params["tableId"],
		Event:   table.TableEventType(ctx.Params["tableEvent"]),
	}
	handler := t.handlers[key]
	if handler == nil {
		return nil, fmt.Errorf("table engine: handler for %q/%q not found", key.TableID, key.Event)
	}

	runtimeCtx, err := t.runtimeContext(ctx, key)
	if err != nil {
		return nil, err
	}
	handler(runtimeCtx)
	if err := runtimeCtx.Error(); err != nil {
		return nil, err
	}

	return &engine.RuntimeResult{
		Mutations:  runtimeCtx.Mutations,
		Navigation: runtimeCtx.Navigation,
	}, nil
}

// GetEngine реализует Page interface через embedding.
func (t *TableEngine) GetEngine() engine.Engine {
	return t
}

// RegisterTableHandler registers a supported handler on this engine instance.
func (t *TableEngine) RegisterTableHandler(tableID string, event table.TableEventType, handler table.TableEventHandler) {
	if handler == nil || tableID == "" {
		return
	}
	if t.handlers == nil {
		t.handlers = map[tableEventKey]table.TableEventHandler{}
	}
	t.handlers[tableEventKey{TableID: tableID, Event: event}] = handler
}

func (t *TableEngine) ensureDSL() {
	if t.dsl.Columns == nil {
		t.dsl = table.New()
	}
}

func (t *TableEngine) builder() *table.Builder {
	t.ensureDSL()
	return table.NewBuilder(&t.dsl).Runtime(t.tableID, t)
}

func (t *TableEngine) renderRoute(pageKey string) engine.RouteHandler {
	return func(ctx *engine.RequestContext, page engine.Page) (any, error) {
		ctx.PageKey = pageKey
		return page.GetEngine().Render(ctx, page)
	}
}

func (t *TableEngine) handleRoute(pageKey string, key tableEventKey) engine.RouteHandler {
	return func(ctx *engine.RequestContext, page engine.Page) (any, error) {
		ctx.PageKey = pageKey
		if ctx.Params == nil {
			ctx.Params = engine.Params{}
		}
		ctx.Params["tableId"] = key.TableID
		ctx.Params["tableEvent"] = string(key.Event)
		return page.GetEngine().Handle(ctx, page)
	}
}

func (t *TableEngine) eventKeys() []tableEventKey {
	keys := make([]tableEventKey, 0, len(t.handlers))
	for key := range t.handlers {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].TableID == keys[j].TableID {
			return tableEventOrder(keys[i].Event) < tableEventOrder(keys[j].Event)
		}
		return keys[i].TableID < keys[j].TableID
	})
	return keys
}

func (t *TableEngine) bindEventRoutes(pageKey string) {
	if pageKey == "" || len(t.handlers) == 0 {
		t.dsl.Events = nil
		return
	}
	routes := &table.TableEventRoutes{}
	for _, key := range t.eventKeys() {
		route := &table.TableEventRoute{
			URL:    tableEventRoutePath(pageKey, key),
			Method: table.HTTPMethodPOST,
		}
		switch key.Event {
		case table.TableEventReload:
			routes.Reload = route
		case table.TableEventFilter:
			routes.Filter = route
		case table.TableEventPagination:
			routes.Pagination = route
		}
	}
	t.dsl.Events = routes
}

func (t *TableEngine) runtimeContext(req *engine.RequestContext, key tableEventKey) (*table.TableRuntimeContext, error) {
	payload := table.TableEventRequest{}
	if len(req.Body) > 0 {
		decoder := json.NewDecoder(bytes.NewReader(req.Body))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&payload); err != nil {
			return nil, fmt.Errorf("table engine: decode %s payload: %w", key.Event, err)
		}
	}

	state := table.TableStateConfig{}
	if t.dsl.State != nil {
		state = *t.dsl.State
	}
	mergeTableState(&state, payload)

	params := make(map[string]any, len(req.Params)+len(payload.Params))
	for name, value := range req.Params {
		params[name] = value
	}
	for name, value := range payload.Params {
		params[name] = value
	}

	return &table.TableRuntimeContext{
		State:  state,
		User:   req.User,
		System: req.System,
		Params: params,
		Extra:  payload.Extra,
		EventTable: &table.TableEventContext{
			TableID:   key.TableID,
			Event:     key.Event,
			PageIndex: state.PageIndex,
			PageSize:  state.PageSize,
			Filters:   state.Filters,
		},
	}, nil
}

func mergeTableState(state *table.TableStateConfig, payload table.TableEventRequest) {
	if payload.PageIndex != nil {
		state.PageIndex = *payload.PageIndex
	}
	if payload.PageSize != nil {
		state.PageSize = *payload.PageSize
	}
	if payload.Filters != nil {
		state.Filters = payload.Filters
	}
}

func tableEventRoutePath(pageKey string, key tableEventKey) string {
	return "/event/" + pageKey + "/table/" + key.TableID + "/" + string(key.Event)
}

func tableEventOrder(event table.TableEventType) int {
	switch event {
	case table.TableEventReload:
		return 0
	case table.TableEventFilter:
		return 1
	case table.TableEventPagination:
		return 2
	default:
		return 3
	}
}
