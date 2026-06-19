package tableengine

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/BekkkEvrika/pageSDK/engine"
	"github.com/BekkkEvrika/pageSDK/table"
)

type runtimeTablePage struct {
	*TableEngine
	called *table.TableRuntimeContext
}

type rowActionTablePage struct {
	*TableEngine
	called *table.TableRuntimeContext
}

type separateRowRoutesPage struct {
	*TableEngine
	called string
}

type toolbarActionTablePage struct {
	*TableEngine
	called *table.TableRuntimeContext
}

type separateToolbarRoutesPage struct {
	*TableEngine
	called string
}

type scopedColumnActionsTablePage struct {
	*TableEngine
	called string
}

type selectedActionTablePage struct {
	*TableEngine
	called *table.TableRuntimeContext
}

func (p *runtimeTablePage) Init(ctx *engine.BuildContext) error {
	p.Table("users").
		Columns(p.Column("id"), p.Column("name")).
		OnReload(p.handle).
		OnFilter(p.handle).
		OnPagination(p.handle)
	return nil
}

func (p *runtimeTablePage) handle(ctx *table.TableRuntimeContext) {
	p.called = ctx
	ctx.Table("users").SetData(table.TableData{
		Rows:      []map[string]any{{"id": 10, "name": "Runtime"}},
		Total:     42,
		PageIndex: ctx.EventTable.PageIndex,
		PageSize:  ctx.EventTable.PageSize,
	})
}

func (p *rowActionTablePage) Init(ctx *engine.BuildContext) error {
	p.Table("users").
		Columns(p.Column("id"), p.Column("name"), p.Column("enabled")).
		RowAction(table.ActionSchema{
			ID:    "edit",
			Label: "Edit",
			Icon:  "pencil",
		}, p.handleEdit)
	return nil
}

func (p *rowActionTablePage) handleEdit(ctx *table.TableRuntimeContext) {
	p.called = ctx
}

func (p *separateRowRoutesPage) Init(ctx *engine.BuildContext) error {
	p.Table("users").
		Columns(p.Column("id"), p.Column("name")).
		RowAction(table.ActionSchema{ID: "delete", Label: "Delete"}, func(ctx *table.TableRuntimeContext) {
			p.called = "delete"
		}).
		RowAction(table.ActionSchema{ID: "edit", Label: "Edit"}, func(ctx *table.TableRuntimeContext) {
			p.called = "edit"
		})
	return nil
}

func (p *toolbarActionTablePage) Init(ctx *engine.BuildContext) error {
	p.Table("users").
		Columns(p.Column("id"), p.Column("name")).
		ToolbarAction(table.ActionSchema{
			ID:      "refresh",
			Label:   "Refresh",
			Icon:    "refresh",
			Hotkey:  "F5",
			Variant: table.ActionVariantSecondary,
		}, func(ctx *table.TableRuntimeContext) {
			p.called = ctx
			ctx.Table("users").SetData(table.TableData{
				Rows:  []map[string]any{{"id": 1, "name": "Refreshed"}},
				Total: 1,
			})
		})
	return nil
}

func (p *separateToolbarRoutesPage) Init(ctx *engine.BuildContext) error {
	p.Table("users").
		ToolbarAction(table.ActionSchema{ID: "export", Label: "Export"}, func(ctx *table.TableRuntimeContext) {
			p.called = "export"
		}).
		ToolbarAction(table.ActionSchema{ID: "refresh", Label: "Refresh"}, func(ctx *table.TableRuntimeContext) {
			p.called = "refresh"
		})
	return nil
}

func (p *scopedColumnActionsTablePage) Init(ctx *engine.BuildContext) error {
	p.Table("users").
		Columns(
			p.Column("id"),
			p.Column("name").AddAction(func(ctx *table.TableRuntimeContext) {
				p.called = "name:" + ctx.EventTable.ColumnID
			}, "normalize"),
			p.Column("email").AddAction(func(ctx *table.TableRuntimeContext) {
				p.called = "email:" + ctx.EventTable.ColumnID
			}, "normalize"),
		)
	return nil
}

func (p *selectedActionTablePage) Init(ctx *engine.BuildContext) error {
	p.Table("users").
		Columns(p.Column("id"), p.Column("name")).
		SelectedAction(table.ActionSchema{
			ID:     "delete_selected",
			Label:  "Delete Selected",
			Hotkey: "Delete",
		}, func(ctx *table.TableRuntimeContext) {
			p.called = ctx
		})
	return nil
}

func TestColumnBuildsPackageTableSchema(t *testing.T) {
	engine := &TableEngine{}

	engine.Columns(
		engine.Column("email").
			Header("Email").
			AccessorKey("email").
			Hidden(true).
			Hideable(false).
			Sortable(true).
			Searchable(true),
	)

	dsl, ok := engine.DSL().(table.TableSchema)
	if !ok {
		t.Fatalf("DSL type = %T, want table.TableSchema", engine.DSL())
	}
	if len(dsl.Columns) != 1 {
		t.Fatalf("columns len = %d, want 1", len(dsl.Columns))
	}

	got := dsl.Columns[0]
	if got.ID != "email" {
		t.Fatalf("column ID = %q, want email", got.ID)
	}
	if got.Header != "Email" {
		t.Fatalf("column Header = %q, want Email", got.Header)
	}
	if got.AccessorKey != "email" {
		t.Fatalf("column AccessorKey = %q, want email", got.AccessorKey)
	}
	if got.Kind != table.TableColumnKindAccessor {
		t.Fatalf("column Kind = %q, want accessor", got.Kind)
	}
	if !got.Hidden {
		t.Fatalf("column Hidden = %v, want true", got.Hidden)
	}
	if got.Hideable {
		t.Fatalf("column Hideable = %v, want false", got.Hideable)
	}
	if !got.Sortable || !got.Searchable {
		t.Fatalf("column flags not preserved: sortable=%v searchable=%v", got.Sortable, got.Searchable)
	}

	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"hidden":true`) {
		t.Fatalf("hidden column was not serialized: %s", encoded)
	}
}

func TestTableEngineSupportsRowsAndActions(t *testing.T) {
	engine := &TableEngine{}

	engine.Table("users").Title("Users")
	engine.Columns(engine.Column("id").Header("ID").DataType(table.TableColumnDataTypeNumber))
	engine.SetRows([]map[string]any{
		{"id": 1},
		{"id": 2},
	})
	engine.AddToolbarAction(table.ActionSchema{
		ID:      "refresh",
		Label:   "Refresh",
		Method:  table.HTTPMethodPOST,
		Variant: table.ActionVariantSecondary,
		Hotkey:  "F5",
	})

	dsl := engine.DSL().(table.TableSchema)
	if dsl.Title != "Users" {
		t.Fatalf("title = %q, want Users", dsl.Title)
	}
	if dsl.Data == nil || dsl.Data.Total != 2 {
		t.Fatalf("data = %#v, want total 2", dsl.Data)
	}
	if dsl.Actions == nil || len(dsl.Actions.Toolbar) != 1 {
		t.Fatalf("toolbar actions = %#v, want one action", dsl.Actions)
	}
	if dsl.Actions.Toolbar[0].Hotkey != "F5" {
		t.Fatalf("toolbar action hotkey = %q, want F5", dsl.Actions.Toolbar[0].Hotkey)
	}

	encoded, err := json.Marshal(dsl)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(encoded, &payload); err != nil {
		t.Fatal(err)
	}
	if _, exists := payload["hotkeys"]; exists {
		t.Fatalf("table DSL must not contain a separate hotkeys field: %s", encoded)
	}
}

func TestColumnWithHeaderAppendsForCompatibility(t *testing.T) {
	engine := &TableEngine{}

	engine.Column("id", "ID")
	engine.Column("name", "Name").Searchable(true)
	engine.Column("email", "Email")

	dsl := engine.DSL().(table.TableSchema)
	if len(dsl.Columns) != 3 {
		t.Fatalf("columns len = %d, want 3", len(dsl.Columns))
	}
	if dsl.Columns[0].ID != "id" || dsl.Columns[1].ID != "name" || dsl.Columns[2].ID != "email" {
		t.Fatalf("unexpected columns: %#v", dsl.Columns)
	}
	if !dsl.Columns[1].Searchable {
		t.Fatalf("returned compatibility builder did not mutate stored column: %#v", dsl.Columns[1])
	}
}

func TestTableBuilderFluentAPI(t *testing.T) {
	engine := &TableEngine{}

	engine.Table("users").
		Title("Users").
		RequestURL("/api/users").
		RowIDKey("id").
		Columns(
			engine.Column("id").Header("ID").AccessorKey("id"),
			engine.Column("name").Header("Name").AccessorKey("name").Searchable(true),
			engine.Column("status").
				Header("Status").
				AccessorKey("status").
				CellType(table.TableColumnCellTypeBadge).
				ValueStyle("active", table.TableCellVariantSuccess).
				ValueStyle("inactive", table.TableCellVariantDanger).
				ValueStyle("pending", table.TableCellVariantWarning),
		).
		Features(table.TableFeatureConfig{
			Sorting:      true,
			Filtering:    true,
			Pagination:   true,
			RowSelection: true,
		}).
		Selection(table.TableSelectionSchema{
			Mode:     table.TableSelectionModeMultiple,
			Checkbox: true,
		})

	dsl := engine.DSL().(table.TableSchema)
	if dsl.Title != "Users" || dsl.RequestURL != "/api/users" || dsl.RowIDKey != "id" {
		t.Fatalf("unexpected table metadata: %#v", dsl)
	}
	if len(dsl.Columns) != 3 {
		t.Fatalf("columns len = %d, want 3", len(dsl.Columns))
	}
	if !dsl.Columns[1].Searchable {
		t.Fatalf("name column must be searchable: %#v", dsl.Columns[1])
	}
	if dsl.Columns[2].CellType != table.TableColumnCellTypeBadge {
		t.Fatalf("status cell type = %q, want badge", dsl.Columns[2].CellType)
	}
	styles := dsl.Columns[2].ValueStyles
	if styles["active"].Variant != table.TableCellVariantSuccess ||
		styles["inactive"].Variant != table.TableCellVariantDanger ||
		styles["pending"].Variant != table.TableCellVariantWarning {
		t.Fatalf("status value styles = %#v", styles)
	}
	if dsl.Features == nil || !dsl.Features.Sorting || !dsl.Features.Pagination {
		t.Fatalf("features not preserved: %#v", dsl.Features)
	}
	if dsl.Selection == nil || dsl.Selection.Mode != table.TableSelectionModeMultiple || !dsl.Selection.Checkbox {
		t.Fatalf("selection not preserved: %#v", dsl.Selection)
	}
}

func TestTableBuilderAppliesSchemaDefaultsAndSimpleData(t *testing.T) {
	engine := &TableEngine{}

	engine.Table("users").
		Columns(
			engine.Column("id"),
			engine.Column("name").Searchable(true),
			engine.Column("status").CellType(table.TableColumnCellTypeBadge),
		).
		Data([]map[string]any{
			{"id": 1, "name": "Behzod", "status": "active"},
			{"id": 2, "name": "Ali", "status": "inactive"},
		})

	dsl := engine.DSL().(table.TableSchema)
	if dsl.Title != "Users" {
		t.Fatalf("title = %q, want Users", dsl.Title)
	}
	if dsl.RequestURL != "/api/users" {
		t.Fatalf("request URL = %q, want /api/users", dsl.RequestURL)
	}
	if dsl.RowIDKey != "id" {
		t.Fatalf("row ID key = %q, want id", dsl.RowIDKey)
	}
	if dsl.Features != nil {
		t.Fatalf("runtime features must require handlers or explicit config: %#v", dsl.Features)
	}
	if dsl.State == nil || dsl.State.PageSize != 20 {
		t.Fatalf("default state not applied: %#v", dsl.State)
	}
	if dsl.Data == nil || dsl.Data.Total != 2 {
		t.Fatalf("data = %#v, want total 2", dsl.Data)
	}
	if dsl.Columns[0].Header != "id" || dsl.Columns[0].AccessorKey != "id" {
		t.Fatalf("column defaults not applied: %#v", dsl.Columns[0])
	}
}

func TestTableRuntimeRoutesAndFeatures(t *testing.T) {
	page := &runtimeTablePage{TableEngine: &TableEngine{}}

	routes := page.TableEngine.Routes("users.list", page)
	if len(routes) != 4 {
		t.Fatalf("routes len = %d, want render plus three events", len(routes))
	}

	wantPaths := []string{
		"/page/users.list",
		"/event/users.list/table/users/reload",
		"/event/users.list/table/users/filter",
		"/event/users.list/table/users/pagination",
	}
	for i, want := range wantPaths {
		if routes[i].Path != want {
			t.Fatalf("route[%d] path = %q, want %q", i, routes[i].Path, want)
		}
	}

	dsl := page.DSL().(table.TableSchema)
	if dsl.Features == nil || !dsl.Features.Reload || !dsl.Features.Filtering || !dsl.Features.Pagination {
		t.Fatalf("event features were not enabled: %#v", dsl.Features)
	}
}

func TestTableRenderExposesRegisteredEventURLs(t *testing.T) {
	page := &runtimeTablePage{TableEngine: &TableEngine{}}

	render, err := page.TableEngine.Render(&engine.RequestContext{
		PageKey: "users.list",
	}, page)
	if err != nil {
		t.Fatalf("render returned error: %v", err)
	}

	dsl, ok := render.DSL.(table.TableSchema)
	if !ok {
		t.Fatalf("DSL type = %T, want table.TableSchema", render.DSL)
	}
	if dsl.ID != "users" {
		t.Fatalf("table id = %q, want users", dsl.ID)
	}
	if dsl.Events == nil {
		t.Fatal("table events are missing from rendered DSL")
	}

	assertRoute := func(name string, route *table.TableEventRoute, wantURL string) {
		t.Helper()
		if route == nil {
			t.Fatalf("%s route is missing", name)
		}
		if route.URL != wantURL || route.Method != table.HTTPMethodPOST {
			t.Fatalf("%s route = %#v, want POST %s", name, route, wantURL)
		}
	}
	assertRoute("reload", dsl.Events.Reload, "/event/users.list/table/users/reload")
	assertRoute("filter", dsl.Events.Filter, "/event/users.list/table/users/filter")
	assertRoute("pagination", dsl.Events.Pagination, "/event/users.list/table/users/pagination")
}

func TestTablePaginationUsesTypedRuntimeContext(t *testing.T) {
	bootstrapPage := &runtimeTablePage{TableEngine: &TableEngine{}}
	routes := bootstrapPage.TableEngine.Routes("users.list", bootstrapPage)
	runtimePage := &runtimeTablePage{TableEngine: &TableEngine{}}

	pageIndex := 2
	pageSize := 25
	body, err := json.Marshal(table.TableEventRequest{
		PageIndex: &pageIndex,
		PageSize:  &pageSize,
		Filters: []table.TableFilterState{
			{ID: "status", Value: "active", Operator: table.TableFilterOperatorEq},
		},
		Params: map[string]any{"tenantId": 17},
		Extra:  map[string]any{"source": "toolbar"},
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := routes[3].Handler(&engine.RequestContext{
		Params: engine.Params{"locale": "en"},
		User:   engine.User{"id": 7},
		System: engine.SystemKeys{"tenant": "main"},
		Body:   body,
	}, runtimePage)
	if err != nil {
		t.Fatalf("pagination handler returned error: %v", err)
	}

	runtime, ok := result.(*engine.RuntimeResult)
	if !ok {
		t.Fatalf("result type = %T, want *engine.RuntimeResult", result)
	}
	if runtimePage.called == nil {
		t.Fatal("table handler was not called")
	}
	ctx := runtimePage.called
	if ctx.EventTable.TableID != "users" || ctx.EventTable.Event != table.TableEventPagination {
		t.Fatalf("unexpected event table: %#v", ctx.EventTable)
	}
	if ctx.EventTable.PageIndex != 2 || ctx.EventTable.PageSize != 25 {
		t.Fatalf("unexpected pagination: %#v", ctx.EventTable)
	}
	if len(ctx.EventTable.Filters) != 1 || ctx.EventTable.Filters[0].ID != "status" {
		t.Fatalf("unexpected filters: %#v", ctx.EventTable.Filters)
	}
	if ctx.Params["locale"] != "en" || ctx.Params["tenantId"] != float64(17) {
		t.Fatalf("unexpected params: %#v", ctx.Params)
	}
	if ctx.Extra["source"] != "toolbar" {
		t.Fatalf("unexpected extra: %#v", ctx.Extra)
	}
	if len(runtime.Mutations) != 1 {
		t.Fatalf("mutations = %#v, want one data update", runtime.Mutations)
	}
	mutation := runtime.Mutations[0]
	if mutation.Type != engine.MutationUpdate || mutation.Path != "tables.users.data" {
		t.Fatalf("unexpected mutation: %#v", mutation)
	}
	data, ok := mutation.Value.(table.TableData)
	if !ok || data.Total != 42 || data.PageIndex != 2 || data.PageSize != 25 {
		t.Fatalf("unexpected mutation data: %#v", mutation.Value)
	}
}

func TestTableRuntimeRejectsInvalidPayload(t *testing.T) {
	bootstrapPage := &runtimeTablePage{TableEngine: &TableEngine{}}
	routes := bootstrapPage.TableEngine.Routes("users.list", bootstrapPage)

	_, err := routes[1].Handler(&engine.RequestContext{
		Body: []byte(`{"pageIndex":`),
	}, &runtimeTablePage{TableEngine: &TableEngine{}})
	if err == nil {
		t.Fatal("expected invalid table payload error")
	}
}

func TestTableRuntimeRejectsDuplicatedStatePayload(t *testing.T) {
	bootstrapPage := &runtimeTablePage{TableEngine: &TableEngine{}}
	routes := bootstrapPage.TableEngine.Routes("users.list", bootstrapPage)

	_, err := routes[3].Handler(&engine.RequestContext{
		Body: []byte(`{
			"state":{"pageIndex":0,"pageSize":10,"filters":[]},
			"pageIndex":0,
			"pageSize":10,
			"filters":[]
		}`),
	}, &runtimeTablePage{TableEngine: &TableEngine{}})
	if err == nil {
		t.Fatal("expected duplicated state payload error")
	}
}

func TestTableToolbarActionExposesRouteInActionDSL(t *testing.T) {
	page := &toolbarActionTablePage{TableEngine: &TableEngine{}}

	routes := page.TableEngine.Routes("users.list", page)
	if len(routes) != 2 {
		t.Fatalf("routes len = %d, want render plus toolbar action", len(routes))
	}
	if routes[1].Path != "/event/users.list/table/users/toolbar/refresh" {
		t.Fatalf("toolbar action route = %q", routes[1].Path)
	}

	render, err := page.TableEngine.Render(&engine.RequestContext{PageKey: "users.list"}, page)
	if err != nil {
		t.Fatalf("render returned error: %v", err)
	}
	dsl := render.DSL.(table.TableSchema)
	if dsl.Events != nil {
		t.Fatalf("toolbar actions must not create table events object: %#v", dsl.Events)
	}
	if dsl.Actions == nil || len(dsl.Actions.Toolbar) != 1 {
		t.Fatalf("toolbar actions missing from DSL: %#v", dsl.Actions)
	}
	action := dsl.Actions.Toolbar[0]
	if action.ID != "refresh" ||
		action.URL != "/event/users.list/table/users/toolbar/refresh" ||
		action.Method != table.HTTPMethodPOST ||
		action.Hotkey != "F5" {
		t.Fatalf("unexpected toolbar action DSL: %#v", action)
	}
}

func TestTableToolbarActionRunsWithoutClientPayload(t *testing.T) {
	bootstrapPage := &toolbarActionTablePage{TableEngine: &TableEngine{}}
	routes := bootstrapPage.TableEngine.Routes("users.list", bootstrapPage)
	runtimePage := &toolbarActionTablePage{TableEngine: &TableEngine{}}

	result, err := routes[1].Handler(&engine.RequestContext{
		Params: engine.Params{"tenantId": "17"},
		User:   engine.User{"id": 7},
		System: engine.SystemKeys{"tenant": "main"},
	}, runtimePage)
	if err != nil {
		t.Fatalf("toolbar action returned error: %v", err)
	}
	runtime, ok := result.(*engine.RuntimeResult)
	if !ok {
		t.Fatalf("result type = %T, want *engine.RuntimeResult", result)
	}
	if runtimePage.called == nil {
		t.Fatal("toolbar action handler was not called")
	}

	ctx := runtimePage.called
	if ctx.EventTable.Event != table.TableEventToolbarAction ||
		ctx.EventTable.ActionID != "refresh" ||
		ctx.EventTable.TableID != "users" {
		t.Fatalf("unexpected toolbar event: %#v", ctx.EventTable)
	}
	if ctx.Params["tenantId"] != "17" || ctx.User["id"] != 7 || ctx.System["tenant"] != "main" {
		t.Fatalf("backend context missing: params=%#v user=%#v system=%#v", ctx.Params, ctx.User, ctx.System)
	}
	if ctx.EventTable.Row != nil || ctx.Extra != nil {
		t.Fatalf("toolbar action must not receive row or client extra: event=%#v extra=%#v", ctx.EventTable, ctx.Extra)
	}
	if len(runtime.Mutations) != 1 || runtime.Mutations[0].Path != "tables.users.data" {
		t.Fatalf("unexpected toolbar action result: %#v", runtime)
	}
}

func TestTableToolbarActionRejectsClientPayload(t *testing.T) {
	bootstrapPage := &toolbarActionTablePage{TableEngine: &TableEngine{}}
	routes := bootstrapPage.TableEngine.Routes("users.list", bootstrapPage)

	_, err := routes[1].Handler(&engine.RequestContext{
		Body: []byte(`{"pageIndex":2}`),
	}, &toolbarActionTablePage{TableEngine: &TableEngine{}})
	if err == nil {
		t.Fatal("expected toolbar action payload error")
	}
}

func TestEachToolbarActionRouteCallsOnlyItsOwnHandler(t *testing.T) {
	bootstrapPage := &separateToolbarRoutesPage{TableEngine: &TableEngine{}}
	routes := bootstrapPage.TableEngine.Routes("users.list", bootstrapPage)
	if len(routes) != 3 {
		t.Fatalf("routes len = %d, want render plus two toolbar routes", len(routes))
	}

	routeByPath := map[string]engine.RouteHandler{}
	for _, route := range routes {
		routeByPath[route.Path] = route.Handler
	}

	refreshPage := &separateToolbarRoutesPage{TableEngine: &TableEngine{}}
	_, err := routeByPath["/event/users.list/table/users/toolbar/refresh"](
		&engine.RequestContext{},
		refreshPage,
	)
	if err != nil {
		t.Fatalf("refresh route returned error: %v", err)
	}
	if refreshPage.called != "refresh" {
		t.Fatalf("refresh route called %q handler", refreshPage.called)
	}

	exportPage := &separateToolbarRoutesPage{TableEngine: &TableEngine{}}
	_, err = routeByPath["/event/users.list/table/users/toolbar/export"](
		&engine.RequestContext{},
		exportPage,
	)
	if err != nil {
		t.Fatalf("export route returned error: %v", err)
	}
	if exportPage.called != "export" {
		t.Fatalf("export route called %q handler", exportPage.called)
	}
}

func TestTableColumnActionsAreScopedByColumn(t *testing.T) {
	bootstrapPage := &scopedColumnActionsTablePage{TableEngine: &TableEngine{}}
	routes := bootstrapPage.TableEngine.Routes("users.list", bootstrapPage)
	if len(routes) != 3 {
		t.Fatalf("routes len = %d, want render plus two column actions", len(routes))
	}

	routeByPath := map[string]engine.RouteHandler{}
	for _, route := range routes {
		routeByPath[route.Path] = route.Handler
	}
	emailPath := "/event/users.list/table/users/column/email/normalize"
	namePath := "/event/users.list/table/users/column/name/normalize"
	if routeByPath[emailPath] == nil || routeByPath[namePath] == nil {
		t.Fatalf("scoped column routes are missing: %#v", routes)
	}

	renderPage := &scopedColumnActionsTablePage{TableEngine: &TableEngine{}}
	render, err := renderPage.TableEngine.Render(
		&engine.RequestContext{PageKey: "users.list"},
		renderPage,
	)
	if err != nil {
		t.Fatalf("render returned error: %v", err)
	}
	dsl := render.DSL.(table.TableSchema)
	assertColumnAction := func(columnID, label, url string) {
		t.Helper()
		for _, column := range dsl.Columns {
			if column.ID != columnID {
				continue
			}
			if len(column.Actions) != 1 {
				t.Fatalf("column %q actions = %#v, want one", columnID, column.Actions)
			}
			action := column.Actions[0]
			if action.ID != "normalize" || action.Label != label ||
				action.URL != url || action.Method != table.HTTPMethodPOST {
				t.Fatalf("column %q action = %#v", columnID, action)
			}
			return
		}
		t.Fatalf("column %q not found", columnID)
	}
	assertColumnAction("name", "Normalize", namePath)
	assertColumnAction("email", "Normalize", emailPath)

	namePage := &scopedColumnActionsTablePage{TableEngine: &TableEngine{}}
	_, err = routeByPath[namePath](
		&engine.RequestContext{Body: []byte(`{"column":{"1":" Behzod "}}`)},
		namePage,
	)
	if err != nil {
		t.Fatalf("name action returned error: %v", err)
	}
	if namePage.called != "name:name" {
		t.Fatalf("name route called %q handler", namePage.called)
	}

	emailPage := &scopedColumnActionsTablePage{TableEngine: &TableEngine{}}
	_, err = routeByPath[emailPath](
		&engine.RequestContext{Body: []byte(`{"column":{"1":"BEHZOD@EXAMPLE.COM"}}`)},
		emailPage,
	)
	if err != nil {
		t.Fatalf("email action returned error: %v", err)
	}
	if emailPage.called != "email:email" {
		t.Fatalf("email route called %q handler", emailPage.called)
	}
}

func TestTableSelectedActionReceivesRowKeys(t *testing.T) {
	bootstrapPage := &selectedActionTablePage{TableEngine: &TableEngine{}}
	routes := bootstrapPage.TableEngine.Routes("users.list", bootstrapPage)
	if len(routes) != 2 || routes[1].Path != "/event/users.list/table/users/selected/delete_selected" {
		t.Fatalf("unexpected selected action routes: %#v", routes)
	}

	render, err := bootstrapPage.TableEngine.Render(
		&engine.RequestContext{PageKey: "users.list"},
		bootstrapPage,
	)
	if err != nil {
		t.Fatalf("render returned error: %v", err)
	}
	action := render.DSL.(table.TableSchema).Actions.Selected[0]
	if action.URL != "/event/users.list/table/users/selected/delete_selected" ||
		action.Method != table.HTTPMethodPOST ||
		action.Hotkey != "Delete" {
		t.Fatalf("unexpected selected action DSL: %#v", action)
	}

	runtimePage := &selectedActionTablePage{TableEngine: &TableEngine{}}
	_, err = routes[1].Handler(&engine.RequestContext{
		Body: []byte(`{"selectedRows":["1","2"]}`),
	}, runtimePage)
	if err != nil {
		t.Fatalf("selected action returned error: %v", err)
	}
	if runtimePage.called == nil {
		t.Fatal("selected action handler was not called")
	}
	ctx := runtimePage.called
	if ctx.EventTable.Event != table.TableEventSelectedAction ||
		ctx.EventTable.ActionID != "delete_selected" ||
		len(ctx.EventTable.SelectedRows) != 2 ||
		ctx.EventTable.SelectedRows[0] != "1" ||
		ctx.EventTable.SelectedRows[1] != "2" {
		t.Fatalf("unexpected selected action context: %#v", ctx.EventTable)
	}
}

func TestTableSelectedActionRejectsMissingOrEmptyRowKeys(t *testing.T) {
	bootstrapPage := &selectedActionTablePage{TableEngine: &TableEngine{}}
	routes := bootstrapPage.TableEngine.Routes("users.list", bootstrapPage)

	_, err := routes[1].Handler(
		&engine.RequestContext{Body: []byte(`{"selectedRows":[]}`)},
		&selectedActionTablePage{TableEngine: &TableEngine{}},
	)
	if err == nil {
		t.Fatal("expected missing selected rows error")
	}

	_, err = routes[1].Handler(
		&engine.RequestContext{Body: []byte(`{"selectedRows":["1",""]}`)},
		&selectedActionTablePage{TableEngine: &TableEngine{}},
	)
	if err == nil {
		t.Fatal("expected empty selected row key error")
	}
}

func TestTableRowActionExposesRouteInActionDSL(t *testing.T) {
	page := &rowActionTablePage{TableEngine: &TableEngine{}}

	routes := page.TableEngine.Routes("users.list", page)
	if len(routes) != 2 {
		t.Fatalf("routes len = %d, want render plus row action", len(routes))
	}
	if routes[1].Path != "/event/users.list/table/users/row/edit" {
		t.Fatalf("row action route = %q", routes[1].Path)
	}

	render, err := page.TableEngine.Render(&engine.RequestContext{PageKey: "users.list"}, page)
	if err != nil {
		t.Fatalf("render returned error: %v", err)
	}
	dsl := render.DSL.(table.TableSchema)
	if dsl.Events != nil {
		t.Fatalf("row actions must not create table events object: %#v", dsl.Events)
	}
	if dsl.Actions == nil || len(dsl.Actions.Row) != 1 {
		t.Fatalf("row actions missing from DSL: %#v", dsl.Actions)
	}
	action := dsl.Actions.Row[0]
	if action.ID != "edit" || action.URL != "/event/users.list/table/users/row/edit" || action.Method != table.HTTPMethodPOST {
		t.Fatalf("unexpected row action DSL: %#v", action)
	}
}

func TestTableRowActionReceivesCurrentRowValues(t *testing.T) {
	bootstrapPage := &rowActionTablePage{TableEngine: &TableEngine{}}
	routes := bootstrapPage.TableEngine.Routes("users.list", bootstrapPage)
	runtimePage := &rowActionTablePage{TableEngine: &TableEngine{}}

	result, err := routes[1].Handler(&engine.RequestContext{
		Body: []byte(`{
			"row": {
				"id": 7,
				"name": "Edited name",
				"enabled": true,
				"quantity": 12
			},
			"params": {"tenantId": 17},
			"extra": {"source": "row-button"}
		}`),
	}, runtimePage)
	if err != nil {
		t.Fatalf("row action returned error: %v", err)
	}
	if _, ok := result.(*engine.RuntimeResult); !ok {
		t.Fatalf("result type = %T, want *engine.RuntimeResult", result)
	}
	if runtimePage.called == nil {
		t.Fatal("row action handler was not called")
	}

	ctx := runtimePage.called
	if ctx.EventTable.Event != table.TableEventRowAction || ctx.EventTable.ActionID != "edit" {
		t.Fatalf("unexpected row event: %#v", ctx.EventTable)
	}
	if ctx.EventTable.Row["id"] != float64(7) {
		t.Fatalf("row id = %#v", ctx.EventTable.Row["id"])
	}
	if ctx.EventTable.Row["name"] != "Edited name" {
		t.Fatalf("input value was not preserved: %#v", ctx.EventTable.Row)
	}
	if ctx.EventTable.Row["enabled"] != true || ctx.EventTable.Row["quantity"] != float64(12) {
		t.Fatalf("row values were not preserved: %#v", ctx.EventTable.Row)
	}
	if ctx.Params["tenantId"] != float64(17) || ctx.Extra["source"] != "row-button" {
		t.Fatalf("params or extra missing: params=%#v extra=%#v", ctx.Params, ctx.Extra)
	}
}

func TestTableRowActionRequiresRowAndRowID(t *testing.T) {
	bootstrapPage := &rowActionTablePage{TableEngine: &TableEngine{}}
	routes := bootstrapPage.TableEngine.Routes("users.list", bootstrapPage)

	_, err := routes[1].Handler(&engine.RequestContext{
		Body: []byte(`{"params":{}}`),
	}, &rowActionTablePage{TableEngine: &TableEngine{}})
	if err == nil {
		t.Fatal("expected missing row error")
	}

	_, err = routes[1].Handler(&engine.RequestContext{
		Body: []byte(`{"row":{"name":"No id"}}`),
	}, &rowActionTablePage{TableEngine: &TableEngine{}})
	if err == nil {
		t.Fatal("expected missing row id error")
	}

	_, err = routes[1].Handler(&engine.RequestContext{
		Body: []byte(`{"row":{"id":7,"name":"Missing enabled"}}`),
	}, &rowActionTablePage{TableEngine: &TableEngine{}})
	if err == nil {
		t.Fatal("expected missing accessor key error")
	}
}

func TestEachRowActionRouteCallsOnlyItsOwnHandler(t *testing.T) {
	bootstrapPage := &separateRowRoutesPage{TableEngine: &TableEngine{}}
	routes := bootstrapPage.TableEngine.Routes("users.list", bootstrapPage)
	if len(routes) != 3 {
		t.Fatalf("routes len = %d, want render plus two row routes", len(routes))
	}

	routeByPath := map[string]engine.RouteHandler{}
	for _, route := range routes {
		routeByPath[route.Path] = route.Handler
	}
	body := []byte(`{"row":{"id":7,"name":"Alice"}}`)

	editPage := &separateRowRoutesPage{TableEngine: &TableEngine{}}
	_, err := routeByPath["/event/users.list/table/users/row/edit"](
		&engine.RequestContext{Body: body},
		editPage,
	)
	if err != nil {
		t.Fatalf("edit route returned error: %v", err)
	}
	if editPage.called != "edit" {
		t.Fatalf("edit route called %q handler", editPage.called)
	}

	deletePage := &separateRowRoutesPage{TableEngine: &TableEngine{}}
	_, err = routeByPath["/event/users.list/table/users/row/delete"](
		&engine.RequestContext{Body: body},
		deletePage,
	)
	if err != nil {
		t.Fatalf("delete route returned error: %v", err)
	}
	if deletePage.called != "delete" {
		t.Fatalf("delete route called %q handler", deletePage.called)
	}
}
