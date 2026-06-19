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
	TableID  string
	Event    table.TableEventType
	ActionID string
	ColumnID string
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

// Action creates a fluent table action builder.
func (t *TableEngine) Action(id string, handler table.TableEventHandler) *table.ActionBuilder {
	return table.NewAction(id, handler)
}

// ToolbarAction appends a toolbar action and registers its handler.
func (t *TableEngine) ToolbarAction(action table.ActionSchema, handler table.TableEventHandler) *table.Builder {
	return t.builder().ToolbarAction(action, handler)
}

// ToolbarActions appends multiple toolbar actions and registers their handlers.
func (t *TableEngine) ToolbarActions(actions ...*table.ActionBuilder) *table.Builder {
	return t.builder().ToolbarActions(actions...)
}

// RowAction appends a row action and registers its handler.
func (t *TableEngine) RowAction(action table.ActionSchema, handler table.TableEventHandler) *table.Builder {
	return t.builder().RowAction(action, handler)
}

// SelectedAction appends an action for selected rows and registers its handler.
func (t *TableEngine) SelectedAction(action table.ActionSchema, handler table.TableEventHandler) *table.Builder {
	return t.builder().SelectedAction(action, handler)
}

// Selection replaces the selection config.
func (t *TableEngine) Selection(selection table.TableSelectionSchema) *table.Builder {
	return t.builder().Selection(selection)
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

// AddSelectedAction appends an action for selected rows.
func (t *TableEngine) AddSelectedAction(action table.ActionSchema) {
	t.ensureDSL()
	t.dsl.AddSelectedAction(action)
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

// Handle is not used for table routes. Every table event has its own route handler.
func (t *TableEngine) Handle(ctx *engine.RequestContext, page engine.Page) (*engine.RuntimeResult, error) {
	return nil, fmt.Errorf("table engine: direct Handle call is not supported")
}

func (t *TableEngine) executeEvent(ctx *engine.RequestContext, page engine.Page, key tableEventKey) (*engine.RuntimeResult, error) {
	if err := page.Init(ctx.BuildContext()); err != nil {
		return nil, err
	}

	handler := t.handlers[key]
	if handler == nil {
		return nil, fmt.Errorf("table engine: handler for table %q event %q action %q not found", key.TableID, key.Event, key.ActionID)
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

// RegisterRowActionHandler registers a row action handler.
func (t *TableEngine) RegisterRowActionHandler(tableID, actionID string, handler table.TableEventHandler) {
	if handler == nil || tableID == "" || actionID == "" {
		return
	}
	if t.handlers == nil {
		t.handlers = map[tableEventKey]table.TableEventHandler{}
	}
	t.handlers[tableEventKey{
		TableID:  tableID,
		Event:    table.TableEventRowAction,
		ActionID: actionID,
	}] = handler
}

// RegisterToolbarActionHandler registers a toolbar action handler.
func (t *TableEngine) RegisterToolbarActionHandler(tableID, actionID string, handler table.TableEventHandler) {
	if handler == nil || tableID == "" || actionID == "" {
		return
	}
	if t.handlers == nil {
		t.handlers = map[tableEventKey]table.TableEventHandler{}
	}
	t.handlers[tableEventKey{
		TableID:  tableID,
		Event:    table.TableEventToolbarAction,
		ActionID: actionID,
	}] = handler
}

// RegisterColumnActionHandler registers a handler for one concrete column.
func (t *TableEngine) RegisterColumnActionHandler(
	tableID string,
	columnID string,
	actionID string,
	handler table.TableEventHandler,
) {
	if handler == nil || tableID == "" || columnID == "" || actionID == "" {
		return
	}
	if t.handlers == nil {
		t.handlers = map[tableEventKey]table.TableEventHandler{}
	}
	t.handlers[tableEventKey{
		TableID:  tableID,
		Event:    table.TableEventColumnAction,
		ActionID: actionID,
		ColumnID: columnID,
	}] = handler
}

// RegisterSelectedActionHandler registers a selected-row action handler.
func (t *TableEngine) RegisterSelectedActionHandler(tableID, actionID string, handler table.TableEventHandler) {
	t.registerActionHandler(tableID, actionID, table.TableEventSelectedAction, handler)
}

func (t *TableEngine) registerActionHandler(
	tableID string,
	actionID string,
	event table.TableEventType,
	handler table.TableEventHandler,
) {
	if handler == nil || tableID == "" || actionID == "" {
		return
	}
	if t.handlers == nil {
		t.handlers = map[tableEventKey]table.TableEventHandler{}
	}
	t.handlers[tableEventKey{
		TableID:  tableID,
		Event:    event,
		ActionID: actionID,
	}] = handler
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
		requestEngine, ok := page.GetEngine().(*TableEngine)
		if !ok {
			return nil, fmt.Errorf("table engine: page %q returned unexpected engine %T", pageKey, page.GetEngine())
		}
		return requestEngine.executeEvent(ctx, page, key)
	}
}

func (t *TableEngine) eventKeys() []tableEventKey {
	keys := make([]tableEventKey, 0, len(t.handlers))
	for key := range t.handlers {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].TableID == keys[j].TableID {
			leftOrder := tableEventOrder(keys[i].Event)
			rightOrder := tableEventOrder(keys[j].Event)
			if leftOrder == rightOrder {
				if keys[i].ColumnID != keys[j].ColumnID {
					return keys[i].ColumnID < keys[j].ColumnID
				}
				return keys[i].ActionID < keys[j].ActionID
			}
			return leftOrder < rightOrder
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
	hasTableEvents := false
	for _, key := range t.eventKeys() {
		switch key.Event {
		case table.TableEventRowAction:
			t.bindRowActionRoute(pageKey, key)
			continue
		case table.TableEventToolbarAction:
			t.bindToolbarActionRoute(pageKey, key)
			continue
		case table.TableEventColumnAction:
			t.bindColumnActionRoute(pageKey, key)
			continue
		case table.TableEventSelectedAction:
			t.bindSelectedActionRoute(pageKey, key)
			continue
		}
		hasTableEvents = true
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
	if hasTableEvents {
		t.dsl.Events = routes
	} else {
		t.dsl.Events = nil
	}
}

func (t *TableEngine) runtimeContext(req *engine.RequestContext, key tableEventKey) (*table.TableRuntimeContext, error) {
	state := table.TableStateConfig{}
	if t.dsl.State != nil {
		state = *t.dsl.State
	}
	var payloadParams map[string]any
	var extra map[string]any
	var row map[string]any
	var column map[string]any
	var selectedRows []string

	switch key.Event {
	case table.TableEventRowAction:
		payload := table.TableRowActionRequest{}
		if err := decodeTablePayload(req.Body, key.Event, &payload); err != nil {
			return nil, err
		}
		if err := t.validateActionRow("row action", key.ActionID, payload.Row); err != nil {
			return nil, err
		}
		payloadParams = payload.Params
		extra = payload.Extra
		row = payload.Row
	case table.TableEventToolbarAction:
		// Toolbar actions are backend commands. The static route identifies the
		// action, so any client payload is ignored.
	case table.TableEventColumnAction:
		payload := table.TableColumnActionRequest{}
		if err := decodeTablePayload(req.Body, key.Event, &payload); err != nil {
			return nil, err
		}
		if len(payload.Column) == 0 {
			return nil, fmt.Errorf("table engine: column action %q requires column values", key.ActionID)
		}
		column = payload.Column
	case table.TableEventSelectedAction:
		payload := table.TableSelectedActionRequest{}
		if err := decodeTablePayload(req.Body, key.Event, &payload); err != nil {
			return nil, err
		}
		if len(payload.SelectedRows) == 0 {
			return nil, fmt.Errorf("table engine: selected action %q requires selected rows", key.ActionID)
		}
		for i, selectedRow := range payload.SelectedRows {
			if selectedRow == "" {
				return nil, fmt.Errorf("table engine: selected action %q requires non-empty row key at index %d", key.ActionID, i)
			}
		}
		selectedRows = payload.SelectedRows
	default:
		payload := table.TableEventRequest{}
		if err := decodeTablePayload(req.Body, key.Event, &payload); err != nil {
			return nil, err
		}
		mergeTableState(&state, payload)
		payloadParams = payload.Params
		extra = payload.Extra
	}

	params := make(map[string]any, len(req.Params)+len(payloadParams))
	for name, value := range req.Params {
		params[name] = value
	}
	for name, value := range payloadParams {
		params[name] = value
	}

	return &table.TableRuntimeContext{
		State:  state,
		User:   req.User,
		System: req.System,
		Params: params,
		Extra:  extra,
		EventTable: &table.TableEventContext{
			TableID:      key.TableID,
			Event:        key.Event,
			ActionID:     key.ActionID,
			ColumnID:     key.ColumnID,
			Row:          row,
			Column:       column,
			SelectedRows: selectedRows,
			PageIndex:    state.PageIndex,
			PageSize:     state.PageSize,
			Filters:      state.Filters,
		},
	}, nil
}

func decodeTablePayload(body []byte, event table.TableEventType, target any) error {
	if len(body) == 0 {
		return nil
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("table engine: decode %s payload: %w", event, err)
	}
	return nil
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
	switch key.Event {
	case table.TableEventRowAction:
		return "/event/" + pageKey + "/table/" + key.TableID + "/row/" + key.ActionID
	case table.TableEventToolbarAction:
		return "/event/" + pageKey + "/table/" + key.TableID + "/toolbar/" + key.ActionID
	case table.TableEventColumnAction:
		return "/event/" + pageKey + "/table/" + key.TableID + "/column/" + key.ColumnID + "/" + key.ActionID
	case table.TableEventSelectedAction:
		return "/event/" + pageKey + "/table/" + key.TableID + "/selected/" + key.ActionID
	}
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
	case table.TableEventRowAction:
		return 3
	case table.TableEventToolbarAction:
		return 4
	case table.TableEventColumnAction:
		return 5
	case table.TableEventSelectedAction:
		return 6
	default:
		return 7
	}
}

func (t *TableEngine) bindToolbarActionRoute(pageKey string, key tableEventKey) {
	if t.dsl.Actions == nil {
		return
	}
	for i := range t.dsl.Actions.Toolbar {
		action := &t.dsl.Actions.Toolbar[i]
		if action.ID != key.ActionID {
			continue
		}
		action.URL = tableEventRoutePath(pageKey, key)
		action.Method = table.HTTPMethodPOST
		return
	}
}

func (t *TableEngine) bindColumnActionRoute(pageKey string, key tableEventKey) {
	for i := range t.dsl.Columns {
		column := &t.dsl.Columns[i]
		if column.ID != key.ColumnID {
			continue
		}
		t.bindActionRoute(column.Actions, pageKey, key)
		return
	}
}

func (t *TableEngine) bindSelectedActionRoute(pageKey string, key tableEventKey) {
	if t.dsl.Actions == nil {
		return
	}
	t.bindActionRoute(t.dsl.Actions.Selected, pageKey, key)
}

func (t *TableEngine) bindActionRoute(actions []table.ActionSchema, pageKey string, key tableEventKey) {
	for i := range actions {
		if actions[i].ID != key.ActionID {
			continue
		}
		actions[i].URL = tableEventRoutePath(pageKey, key)
		actions[i].Method = table.HTTPMethodPOST
		return
	}
}

func (t *TableEngine) bindRowActionRoute(pageKey string, key tableEventKey) {
	if t.dsl.Actions == nil {
		return
	}
	for i := range t.dsl.Actions.Row {
		action := &t.dsl.Actions.Row[i]
		if action.ID != key.ActionID {
			continue
		}
		action.URL = tableEventRoutePath(pageKey, key)
		action.Method = table.HTTPMethodPOST
		return
	}
}

func (t *TableEngine) validateActionRow(kind, actionID string, row map[string]any) error {
	if row == nil {
		return fmt.Errorf("table engine: %s %q requires row", kind, actionID)
	}
	if rowIDKey := t.dsl.RowIDKey; rowIDKey != "" {
		if _, ok := row[rowIDKey]; !ok {
			return fmt.Errorf("table engine: %s %q requires row key %q", kind, actionID, rowIDKey)
		}
	}
	for _, column := range t.dsl.Columns {
		if column.Kind != table.TableColumnKindAccessor || column.AccessorKey == "" {
			continue
		}
		if _, ok := row[column.AccessorKey]; !ok {
			return fmt.Errorf("table engine: %s %q requires accessor key %q", kind, actionID, column.AccessorKey)
		}
	}
	return nil
}
