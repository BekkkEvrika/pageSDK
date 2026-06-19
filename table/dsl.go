package table

import (
	"strings"
	"unicode"
)

// New creates an empty table DSL schema.
func New(name ...string) TableSchema {
	schema := TableSchema{
		Columns:      []TableColumnSchema{},
		RowIDKey:     "id",
		State:        &TableStateConfig{PageSize: 20},
		EmptyMessage: "No data",
	}
	if len(name) > 0 && name[0] != "" {
		schema.ID = name[0]
		schema.Title = titleFromName(name[0])
		schema.RequestURL = "/api/" + name[0]
	}
	return schema
}

// NewBuilder creates a fluent builder over schema.
func NewBuilder(schema *TableSchema) *Builder {
	if schema.Columns == nil {
		*schema = New()
	}
	return &Builder{schema: schema}
}

// NewColumn creates a fluent column builder.
func NewColumn(id string) *ColumnBuilder {
	return NewColumnBuilder(&TableColumnSchema{
		ID:          id,
		Header:      id,
		Kind:        TableColumnKindAccessor,
		AccessorKey: id,
		CellType:    TableColumnCellTypeText,
		DataType:    TableColumnDataTypeString,
	})
}

// NewColumnBuilder creates a fluent builder over column.
func NewColumnBuilder(column *TableColumnSchema) *ColumnBuilder {
	return &ColumnBuilder{column: column}
}

// Builder mutates a TableSchema owned by an engine instance.
type Builder struct {
	schema        *TableSchema
	tableID       string
	registrar     TableEventRegistrar
	enabledEvents map[TableEventType]bool
}

// Schema returns the mutable schema.
func (b *Builder) Schema() *TableSchema {
	return b.schema
}

// Runtime binds table event registration to this builder.
func (b *Builder) Runtime(tableID string, registrar TableEventRegistrar) *Builder {
	b.tableID = tableID
	b.registrar = registrar
	return b
}

// Title sets the table title.
func (b *Builder) Title(title string) *Builder {
	b.schema.Title = title
	return b
}

// RequestURL sets the remote data request URL.
func (b *Builder) RequestURL(url string) *Builder {
	b.schema.RequestURL = url
	return b
}

// RowIDKey sets the field used as a stable row identifier.
func (b *Builder) RowIDKey(key string) *Builder {
	b.schema.RowIDKey = key
	return b
}

// EmptyMessage sets text displayed when the table has no rows.
func (b *Builder) EmptyMessage(message string) *Builder {
	b.schema.EmptyMessage = message
	return b
}

// SubRowsKey sets the key that contains nested rows.
func (b *Builder) SubRowsKey(key string) *Builder {
	b.schema.SubRowsKey = key
	return b
}

// SubRowsRequestURL sets the request URL for nested rows.
func (b *Builder) SubRowsRequestURL(url string) *Builder {
	b.schema.SubRowsRequestURL = url
	return b
}

// Columns replaces table columns with the provided column builders.
func (b *Builder) Columns(columns ...*ColumnBuilder) *Builder {
	b.schema.Columns = make([]TableColumnSchema, 0, len(columns))
	for _, column := range columns {
		if column == nil {
			continue
		}
		b.schema.Columns = append(b.schema.Columns, column.Schema())
		if b.registrar == nil || b.tableID == "" {
			continue
		}
		registrar, ok := b.registrar.(TableColumnActionRegistrar)
		if !ok {
			continue
		}
		for _, pending := range column.pendingActions {
			registrar.RegisterColumnActionHandler(
				b.tableID,
				column.column.ID,
				pending.actionID,
				pending.handler,
			)
		}
	}
	return b
}

// Features replaces the feature config.
func (b *Builder) Features(features TableFeatureConfig) *Builder {
	b.applyEventFeatures(&features)
	b.schema.Features = &features
	return b
}

// Actions replaces all table action groups.
func (b *Builder) Actions(actions TableActionGroups) *Builder {
	b.schema.Actions = &actions
	return b
}

// Selection replaces the selection config.
func (b *Builder) Selection(selection TableSelectionSchema) *Builder {
	b.schema.Selection = &selection
	return b
}

// State replaces the initial table state.
func (b *Builder) State(state TableStateConfig) *Builder {
	b.schema.State = &state
	return b
}

// Data replaces inline table data.
func (b *Builder) Data(data any) *Builder {
	b.schema.Data = normalizeData(data)
	return b
}

// OnReload registers a reload handler and enables reload support.
func (b *Builder) OnReload(handler TableEventHandler) *Builder {
	return b.on(TableEventReload, handler)
}

// OnFilter registers a filter handler and enables filtering.
func (b *Builder) OnFilter(handler TableEventHandler) *Builder {
	return b.on(TableEventFilter, handler)
}

// OnPagination registers a pagination handler and enables pagination.
func (b *Builder) OnPagination(handler TableEventHandler) *Builder {
	return b.on(TableEventPagination, handler)
}

// ToolbarAction appends a toolbar action and registers its handler.
func (b *Builder) ToolbarAction(action ActionSchema, handler TableEventHandler) *Builder {
	if action.ID == "" || handler == nil || b.registrar == nil || b.tableID == "" {
		return b
	}
	registrar, ok := b.registrar.(TableToolbarActionRegistrar)
	if !ok {
		return b
	}
	if action.Label == "" {
		action.Label = titleFromName(action.ID)
	}
	actions := b.schema.Actions
	if actions == nil {
		actions = &TableActionGroups{}
		b.schema.Actions = actions
	}
	actions.Toolbar = append(actions.Toolbar, action)
	registrar.RegisterToolbarActionHandler(b.tableID, action.ID, handler)
	return b
}

// RowAction appends a row action and registers its handler.
func (b *Builder) RowAction(action ActionSchema, handler TableEventHandler) *Builder {
	if action.ID == "" || handler == nil || b.registrar == nil || b.tableID == "" {
		return b
	}
	if action.Label == "" {
		action.Label = titleFromName(action.ID)
	}
	actions := b.schema.Actions
	if actions == nil {
		actions = &TableActionGroups{}
		b.schema.Actions = actions
	}
	actions.Row = append(actions.Row, action)
	b.registrar.RegisterRowActionHandler(b.tableID, action.ID, handler)
	return b
}

// SelectedAction appends a selected-row action and registers its handler.
func (b *Builder) SelectedAction(action ActionSchema, handler TableEventHandler) *Builder {
	if action.ID == "" || handler == nil || b.registrar == nil || b.tableID == "" {
		return b
	}
	registrar, ok := b.registrar.(TableSelectedActionRegistrar)
	if !ok {
		return b
	}
	if action.Label == "" {
		action.Label = titleFromName(action.ID)
	}
	actions := b.schema.Actions
	if actions == nil {
		actions = &TableActionGroups{}
		b.schema.Actions = actions
	}
	actions.Selected = append(actions.Selected, action)
	registrar.RegisterSelectedActionHandler(b.tableID, action.ID, handler)
	return b
}

// SetTitle sets the table title.
func (b *Builder) SetTitle(title string) {
	b.Title(title)
}

// SetRequestURL sets the remote data request URL.
func (b *Builder) SetRequestURL(url string) {
	b.RequestURL(url)
}

// SetRowIDKey sets the field used as a stable row identifier.
func (b *Builder) SetRowIDKey(key string) {
	b.RowIDKey(key)
}

// SetEmptyMessage sets text displayed when the table has no rows.
func (b *Builder) SetEmptyMessage(message string) {
	b.EmptyMessage(message)
}

// SetSubRowsKey sets the key that contains nested rows.
func (b *Builder) SetSubRowsKey(key string) {
	b.SubRowsKey(key)
}

// SetSubRowsRequestURL sets the request URL for nested rows.
func (b *Builder) SetSubRowsRequestURL(url string) {
	b.SubRowsRequestURL(url)
}

// SetFeatures replaces the feature config.
func (b *Builder) SetFeatures(features TableFeatureConfig) {
	b.Features(features)
}

// SetActions replaces all table action groups.
func (b *Builder) SetActions(actions TableActionGroups) {
	b.Actions(actions)
}

// SetSelection replaces the selection config.
func (b *Builder) SetSelection(selection TableSelectionSchema) {
	b.Selection(selection)
}

// SetState replaces the initial table state.
func (b *Builder) SetState(state TableStateConfig) {
	b.State(state)
}

// SetData replaces inline table data.
func (b *Builder) SetData(data any) {
	b.Data(data)
}

// ColumnBuilder builds a TableColumnSchema.
type ColumnBuilder struct {
	column         *TableColumnSchema
	pendingActions []pendingColumnAction
}

type pendingColumnAction struct {
	actionID string
	handler  TableEventHandler
}

// Schema returns a copy of the built column schema.
func (b *ColumnBuilder) Schema() TableColumnSchema {
	if b == nil || b.column == nil {
		return TableColumnSchema{}
	}
	return *b.column
}

// Header sets the column header.
func (b *ColumnBuilder) Header(header string) *ColumnBuilder {
	b.column.Header = header
	return b
}

// AccessorKey sets the row field used by this column.
func (b *ColumnBuilder) AccessorKey(key string) *ColumnBuilder {
	b.column.AccessorKey = key
	return b
}

// Kind sets the column kind.
func (b *ColumnBuilder) Kind(kind TableColumnKind) *ColumnBuilder {
	b.column.Kind = kind
	return b
}

// Hidden sets whether the column is hidden by default.
func (b *ColumnBuilder) Hidden(hidden bool) *ColumnBuilder {
	b.column.Hidden = hidden
	return b
}

// Sortable toggles column sorting.
func (b *ColumnBuilder) Sortable(sortable bool) *ColumnBuilder {
	b.column.Sortable = sortable
	return b
}

// Filterable toggles column filtering.
func (b *ColumnBuilder) Filterable(filterable bool) *ColumnBuilder {
	b.column.Filterable = filterable
	return b
}

// Searchable toggles global search participation.
func (b *ColumnBuilder) Searchable(searchable bool) *ColumnBuilder {
	b.column.Searchable = searchable
	return b
}

// AddAction appends an action to this column. The handler is registered when
// the column is attached to a runtime table through Builder.Columns.
func (b *ColumnBuilder) AddAction(handler TableEventHandler, name string) *ColumnBuilder {
	if b == nil || b.column == nil || handler == nil || name == "" {
		return b
	}
	b.column.Actions = append(b.column.Actions, ActionSchema{
		ID:    name,
		Label: titleFromName(name),
	})
	b.pendingActions = append(b.pendingActions, pendingColumnAction{
		actionID: name,
		handler:  handler,
	})
	return b
}

// Hideable toggles column visibility control.
func (b *ColumnBuilder) Hideable(hideable bool) *ColumnBuilder {
	b.column.Hideable = hideable
	return b
}

// Resizable toggles column resizing.
func (b *ColumnBuilder) Resizable(resizable bool) *ColumnBuilder {
	b.column.Resizable = resizable
	return b
}

// Width sets the preferred column width.
func (b *ColumnBuilder) Width(width int) *ColumnBuilder {
	b.column.Width = width
	return b
}

// MinWidth sets the minimum column width.
func (b *ColumnBuilder) MinWidth(width int) *ColumnBuilder {
	b.column.MinWidth = width
	return b
}

// MaxWidth sets the maximum column width.
func (b *ColumnBuilder) MaxWidth(width int) *ColumnBuilder {
	b.column.MaxWidth = width
	return b
}

// Align sets column alignment.
func (b *ColumnBuilder) Align(align TableColumnAlign) *ColumnBuilder {
	b.column.Align = align
	return b
}

// DataType sets the column data type.
func (b *ColumnBuilder) DataType(dataType TableColumnDataType) *ColumnBuilder {
	b.column.DataType = dataType
	return b
}

// CellType sets the column cell renderer type.
func (b *ColumnBuilder) CellType(cellType TableColumnCellType) *ColumnBuilder {
	b.column.CellType = cellType
	return b
}

// ValueStyle assigns a semantic visual variant to one concrete cell value.
func (b *ColumnBuilder) ValueStyle(value string, variant TableCellVariant) *ColumnBuilder {
	if b == nil || b.column == nil || value == "" {
		return b
	}
	if b.column.ValueStyles == nil {
		b.column.ValueStyles = map[string]TableValueStyle{}
	}
	b.column.ValueStyles[value] = TableValueStyle{Variant: variant}
	return b
}

// Format sets the column value format.
func (b *ColumnBuilder) Format(format TableColumnFormat) *ColumnBuilder {
	b.column.Format = &format
	return b
}

// SetHeader changes the column header.
func (b *ColumnBuilder) SetHeader(header string) {
	b.Header(header)
}

// SetAccessorKey changes the row field used by this column.
func (b *ColumnBuilder) SetAccessorKey(key string) {
	b.AccessorKey(key)
}

// SetKind changes the column kind.
func (b *ColumnBuilder) SetKind(kind TableColumnKind) {
	b.Kind(kind)
}

// SetHidden sets whether the column is hidden by default.
func (b *ColumnBuilder) SetHidden(hidden bool) {
	b.Hidden(hidden)
}

// SetSortable toggles column sorting.
func (b *ColumnBuilder) SetSortable(sortable bool) {
	b.Sortable(sortable)
}

// SetFilterable toggles column filtering.
func (b *ColumnBuilder) SetFilterable(filterable bool) {
	b.Filterable(filterable)
}

// SetSearchable toggles global search participation.
func (b *ColumnBuilder) SetSearchable(searchable bool) {
	b.Searchable(searchable)
}

// SetHideable toggles column visibility control.
func (b *ColumnBuilder) SetHideable(hideable bool) {
	b.Hideable(hideable)
}

// SetResizable toggles column resizing.
func (b *ColumnBuilder) SetResizable(resizable bool) {
	b.Resizable(resizable)
}

// SetWidth sets the preferred column width.
func (b *ColumnBuilder) SetWidth(width int) {
	b.Width(width)
}

// SetWidthBounds sets min and max column widths.
func (b *ColumnBuilder) SetWidthBounds(minWidth, maxWidth int) {
	b.MinWidth(minWidth).MaxWidth(maxWidth)
}

// SetAlign changes column alignment.
func (b *ColumnBuilder) SetAlign(align TableColumnAlign) {
	b.Align(align)
}

// SetDataType changes the column data type.
func (b *ColumnBuilder) SetDataType(dataType TableColumnDataType) {
	b.DataType(dataType)
}

// SetCellType changes the column cell renderer type.
func (b *ColumnBuilder) SetCellType(cellType TableColumnCellType) {
	b.CellType(cellType)
}

// SetValueStyle assigns a semantic visual variant to one concrete cell value.
func (b *ColumnBuilder) SetValueStyle(value string, variant TableCellVariant) {
	b.ValueStyle(value, variant)
}

// SetFormat changes the column value format.
func (b *ColumnBuilder) SetFormat(format *TableColumnFormat) {
	if format == nil {
		b.column.Format = nil
		return
	}
	b.Format(*format)
}

// SetTitle sets the table title.
func (t *TableSchema) SetTitle(title string) {
	NewBuilder(t).Title(title)
}

// SetRequestURL sets the remote data request URL.
func (t *TableSchema) SetRequestURL(url string) {
	NewBuilder(t).RequestURL(url)
}

// SetRowIDKey sets the field used as a stable row identifier.
func (t *TableSchema) SetRowIDKey(key string) {
	NewBuilder(t).RowIDKey(key)
}

// SetEmptyMessage sets text displayed when the table has no rows.
func (t *TableSchema) SetEmptyMessage(message string) {
	NewBuilder(t).EmptyMessage(message)
}

// SetSubRows configures nested rows.
func (t *TableSchema) SetSubRows(key, requestURL string) {
	NewBuilder(t).SubRowsKey(key).SubRowsRequestURL(requestURL)
}

// Column appends an accessor column and returns its mutable DSL handle.
func (t *TableSchema) Column(id, header string) *TableColumnSchema {
	column := NewColumn(id).Header(header).Schema()
	t.Columns = append(t.Columns, column)
	return &t.Columns[len(t.Columns)-1]
}

// DisplayColumn appends a display-only column and returns its mutable DSL handle.
func (t *TableSchema) DisplayColumn(id, header string) *TableColumnSchema {
	column := NewColumn(id).Header(header).Kind(TableColumnKindDisplay).AccessorKey("").Schema()
	t.Columns = append(t.Columns, column)
	return &t.Columns[len(t.Columns)-1]
}

// SetColumns replaces the table columns.
func (t *TableSchema) SetColumns(columns []TableColumnSchema) {
	t.Columns = columns
}

// SetData configures inline table data.
func (t *TableSchema) SetData(data TableData) {
	NewBuilder(t).Data(data)
}

// SetRows configures inline table rows.
func (t *TableSchema) SetRows(rows []map[string]any) {
	NewBuilder(t).Data(TableData{
		Rows:  rows,
		Total: len(rows),
	})
}

// SetState replaces the initial table state.
func (t *TableSchema) SetState(state TableStateConfig) {
	NewBuilder(t).State(state)
}

// SetActions replaces all table action groups.
func (t *TableSchema) SetActions(actions TableActionGroups) {
	NewBuilder(t).Actions(actions)
}

// SetSelection replaces the selection config.
func (t *TableSchema) SetSelection(selection TableSelectionSchema) {
	NewBuilder(t).Selection(selection)
}

// FeaturesConfig returns the mutable table feature config.
func (t *TableSchema) FeaturesConfig() *TableFeatureConfig {
	if t.Features == nil {
		t.Features = &TableFeatureConfig{}
	}
	return t.Features
}

// ActionsConfig returns the mutable table action groups.
func (t *TableSchema) ActionsConfig() *TableActionGroups {
	if t.Actions == nil {
		t.Actions = &TableActionGroups{}
	}
	return t.Actions
}

// StateConfig returns the mutable initial table state.
func (t *TableSchema) StateConfig() *TableStateConfig {
	if t.State == nil {
		t.State = &TableStateConfig{}
	}
	return t.State
}

// SelectionConfig returns the mutable selection config.
func (t *TableSchema) SelectionConfig(mode TableSelectionMode) *TableSelectionSchema {
	if t.Selection == nil {
		t.Selection = &TableSelectionSchema{Mode: mode}
	}
	return t.Selection
}

// AddToolbarAction appends a toolbar action.
func (t *TableSchema) AddToolbarAction(action ActionSchema) {
	actions := t.ActionsConfig()
	actions.Toolbar = append(actions.Toolbar, action)
}

// AddRowAction appends a row action.
func (t *TableSchema) AddRowAction(action ActionSchema) {
	actions := t.ActionsConfig()
	actions.Row = append(actions.Row, action)
}

// AddSelectedAction appends an action for selected rows.
func (t *TableSchema) AddSelectedAction(action ActionSchema) {
	actions := t.ActionsConfig()
	actions.Selected = append(actions.Selected, action)
}

// SetHeader changes the column header.
func (c *TableColumnSchema) SetHeader(header string) {
	c.Header = header
}

// SetAccessorKey changes the row field used by this column.
func (c *TableColumnSchema) SetAccessorKey(key string) {
	c.AccessorKey = key
}

// SetKind changes the column kind.
func (c *TableColumnSchema) SetKind(kind TableColumnKind) {
	c.Kind = kind
}

// SetHidden sets whether the column is hidden by default.
func (c *TableColumnSchema) SetHidden(hidden bool) {
	c.Hidden = hidden
}

// SetSortable toggles column sorting.
func (c *TableColumnSchema) SetSortable(sortable bool) {
	c.Sortable = sortable
}

// SetFilterable toggles column filtering.
func (c *TableColumnSchema) SetFilterable(filterable bool) {
	c.Filterable = filterable
}

// SetSearchable toggles global search participation.
func (c *TableColumnSchema) SetSearchable(searchable bool) {
	c.Searchable = searchable
}

// SetHideable toggles column visibility control.
func (c *TableColumnSchema) SetHideable(hideable bool) {
	c.Hideable = hideable
}

// SetResizable toggles column resizing.
func (c *TableColumnSchema) SetResizable(resizable bool) {
	c.Resizable = resizable
}

// SetWidth sets the preferred column width.
func (c *TableColumnSchema) SetWidth(width int) {
	c.Width = width
}

// SetWidthBounds sets min and max column widths.
func (c *TableColumnSchema) SetWidthBounds(minWidth, maxWidth int) {
	c.MinWidth = minWidth
	c.MaxWidth = maxWidth
}

// SetAlign changes column alignment.
func (c *TableColumnSchema) SetAlign(align TableColumnAlign) {
	c.Align = align
}

// SetDataType changes the column data type.
func (c *TableColumnSchema) SetDataType(dataType TableColumnDataType) {
	c.DataType = dataType
}

// SetCellType changes the column cell renderer type.
func (c *TableColumnSchema) SetCellType(cellType TableColumnCellType) {
	c.CellType = cellType
}

// SetValueStyle assigns a semantic visual variant to one concrete cell value.
func (c *TableColumnSchema) SetValueStyle(value string, variant TableCellVariant) {
	if value == "" {
		return
	}
	if c.ValueStyles == nil {
		c.ValueStyles = map[string]TableValueStyle{}
	}
	c.ValueStyles[value] = TableValueStyle{Variant: variant}
}

// SetFormat changes the column value format.
func (c *TableColumnSchema) SetFormat(format *TableColumnFormat) {
	c.Format = format
}

func normalizeData(data any) *TableData {
	switch value := data.(type) {
	case nil:
		return nil
	case TableData:
		return &value
	case *TableData:
		return value
	case []map[string]any:
		return &TableData{
			Rows:  value,
			Total: len(value),
		}
	default:
		return &TableData{
			Rows: []map[string]any{
				{"value": value},
			},
			Total: 1,
		}
	}
}

func titleFromName(name string) string {
	words := strings.FieldsFunc(name, func(r rune) bool {
		return r == '-' || r == '_' || r == '.'
	})
	if len(words) == 0 {
		return name
	}
	for i, word := range words {
		words[i] = upperFirst(word)
	}
	return strings.Join(words, " ")
}

func upperFirst(value string) string {
	runes := []rune(value)
	if len(runes) == 0 {
		return value
	}
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func (b *Builder) on(event TableEventType, handler TableEventHandler) *Builder {
	if handler == nil || b.registrar == nil || b.tableID == "" {
		return b
	}
	if b.enabledEvents == nil {
		b.enabledEvents = map[TableEventType]bool{}
	}
	b.enabledEvents[event] = true
	b.registrar.RegisterTableHandler(b.tableID, event, handler)
	features := b.schema.Features
	if features == nil {
		features = &TableFeatureConfig{}
		b.schema.Features = features
	}
	b.applyEventFeatures(features)
	return b
}

func (b *Builder) applyEventFeatures(features *TableFeatureConfig) {
	if features == nil {
		return
	}
	if b.enabledEvents[TableEventReload] {
		features.Reload = true
	}
	if b.enabledEvents[TableEventFilter] {
		features.Filtering = true
	}
	if b.enabledEvents[TableEventPagination] {
		features.Pagination = true
	}
}
