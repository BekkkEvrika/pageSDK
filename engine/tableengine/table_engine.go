package tableengine

import (
	"net/http"

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
	dsl table.TableSchema
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
	return table.NewBuilder(&t.dsl)
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
	return []engine.RouteDefinition{
		{
			Method:  http.MethodGet,
			Path:    "/page/" + pageKey,
			Handler: t.renderRoute(pageKey),
		},
	}
}

// Render создаёт DSL таблицы.
func (t *TableEngine) Render(ctx *engine.RequestContext, page engine.Page) (*engine.RenderResult, error) {
	if err := page.Init(ctx.BuildContext()); err != nil {
		return nil, err
	}

	return &engine.RenderResult{
		PageKey: ctx.PageKey,
		Engine:  t.ID(),
		DSL:     t.DSL(),
	}, nil
}

// Handle intentionally does not implement table runtime behavior yet.
func (t *TableEngine) Handle(ctx *engine.RequestContext, page engine.Page) (*engine.RuntimeResult, error) {
	return &engine.RuntimeResult{}, nil
}

// GetEngine реализует Page interface через embedding.
func (t *TableEngine) GetEngine() engine.Engine {
	return t
}

func (t *TableEngine) ensureDSL() {
	if t.dsl.Columns == nil {
		t.dsl = table.New()
	}
}

func (t *TableEngine) builder() *table.Builder {
	t.ensureDSL()
	return table.NewBuilder(&t.dsl)
}

func (t *TableEngine) renderRoute(pageKey string) engine.RouteHandler {
	return func(ctx *engine.RequestContext, page engine.Page) (any, error) {
		ctx.PageKey = pageKey
		return page.GetEngine().Render(ctx, page)
	}
}
