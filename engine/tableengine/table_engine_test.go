package tableengine

import (
	"encoding/json"
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

func TestColumnBuildsPackageTableSchema(t *testing.T) {
	engine := &TableEngine{}

	engine.Columns(
		engine.Column("email").
			Header("Email").
			AccessorKey("email").
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
	if !got.Sortable || !got.Searchable {
		t.Fatalf("column flags not preserved: sortable=%v searchable=%v", got.Sortable, got.Searchable)
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
			engine.Column("status").Header("Status").AccessorKey("status").CellType(table.TableColumnCellTypeBadge),
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
