package table

type TableSchema struct {
	ID                string                `json:"id"`
	Title             string                `json:"title,omitempty"`
	RequestURL        string                `json:"requestUrl,omitempty"`
	Columns           []TableColumnSchema   `json:"columns"`
	Features          *TableFeatureConfig   `json:"features,omitempty"`
	Events            *TableEventRoutes     `json:"events,omitempty"`
	Actions           *TableActionGroups    `json:"actions,omitempty"`
	Selection         *TableSelectionSchema `json:"selection,omitempty"`
	RowIDKey          string                `json:"rowIdKey,omitempty"`
	State             *TableStateConfig     `json:"state,omitempty"`
	Data              *TableData            `json:"data,omitempty"`
	EmptyMessage      string                `json:"emptyMessage,omitempty"`
	SubRowsKey        string                `json:"subRowsKey,omitempty"`
	SubRowsRequestURL string                `json:"subRowsRequestUrl,omitempty"`
}

type TableEventRoutes struct {
	Reload     *TableEventRoute `json:"reload,omitempty"`
	Filter     *TableEventRoute `json:"filter,omitempty"`
	Pagination *TableEventRoute `json:"pagination,omitempty"`
}

type TableEventRoute struct {
	URL    string     `json:"url"`
	Method HTTPMethod `json:"method"`
}

type TableColumnSchema struct {
	ID          string          `json:"id"`
	Header      string          `json:"header"`
	Kind        TableColumnKind `json:"kind,omitempty"`
	AccessorKey string          `json:"accessorKey,omitempty"`
	Actions     []ActionSchema  `json:"actions,omitempty"`

	Hidden bool `json:"hidden,omitempty"`

	Sortable   bool `json:"sortable,omitempty"`
	Filterable bool `json:"filterable,omitempty"`
	Searchable bool `json:"searchable,omitempty"`
	Hideable   bool `json:"hideable,omitempty"`
	Resizable  bool `json:"resizable,omitempty"`

	Width    int `json:"width,omitempty"`
	MinWidth int `json:"minWidth,omitempty"`
	MaxWidth int `json:"maxWidth,omitempty"`

	Align    TableColumnAlign    `json:"align,omitempty"`
	DataType TableColumnDataType `json:"dataType,omitempty"`
	CellType TableColumnCellType `json:"cellType,omitempty"`

	Format *TableColumnFormat `json:"format,omitempty"`
}

type TableData struct {
	Rows      []map[string]any `json:"rows"`
	Total     int              `json:"total,omitempty"`
	PageIndex int              `json:"pageIndex,omitempty"`
	PageSize  int              `json:"pageSize,omitempty"`
}

type TableActionGroups struct {
	Toolbar  []ActionSchema `json:"toolbar,omitempty"`
	Row      []ActionSchema `json:"row,omitempty"`
	Selected []ActionSchema `json:"selected,omitempty"`
}

type ActionSchema struct {
	ID      string        `json:"id"`
	Label   string        `json:"label"`
	Icon    string        `json:"icon,omitempty"`
	Variant ActionVariant `json:"variant,omitempty"`
	URL     string        `json:"url,omitempty"`
	Method  HTTPMethod    `json:"method,omitempty"`
	Hotkey  string        `json:"hotkey,omitempty"`
}

type TableFeatureConfig struct {
	Reload        bool `json:"reload,omitempty"`
	Sorting       bool `json:"sorting,omitempty"`
	Filtering     bool `json:"filtering,omitempty"`
	GlobalSearch  bool `json:"globalSearch,omitempty"`
	Pagination    bool `json:"pagination,omitempty"`
	RowSelection  bool `json:"rowSelection,omitempty"`
	ColumnResize  bool `json:"columnResize,omitempty"`
	VirtualScroll bool `json:"virtualScroll,omitempty"`
	SubRows       bool `json:"subRows,omitempty"`
}

type TableSelectionSchema struct {
	Mode               TableSelectionMode `json:"mode"`
	Checkbox           bool               `json:"checkbox,omitempty"`
	PersistAcrossPages bool               `json:"persistAcrossPages,omitempty"`
	SelectOnRowClick   bool               `json:"selectOnRowClick,omitempty"`
	AllowDeselect      bool               `json:"allowDeselect,omitempty"`
}

type TableColumnFormat struct {
	Type                  TableColumnFormatType `json:"type"`
	Currency              string                `json:"currency,omitempty"`
	Locale                string                `json:"locale,omitempty"`
	MinimumFractionDigits int                   `json:"minimumFractionDigits,omitempty"`
	MaximumFractionDigits int                   `json:"maximumFractionDigits,omitempty"`
}

type TableStateConfig struct {
	PageIndex        int                `json:"pageIndex,omitempty"`
	PageSize         int                `json:"pageSize,omitempty"`
	GlobalFilter     string             `json:"globalFilter,omitempty"`
	Sorting          []TableSortingItem `json:"sorting,omitempty"`
	Filters          []TableFilterState `json:"filters,omitempty"`
	ColumnVisibility map[string]bool    `json:"columnVisibility,omitempty"`
	SelectedRows     []string           `json:"selectedRows,omitempty"`
	ColumnSizing     map[string]int     `json:"columnSizing,omitempty"`
}

type TableSortingItem struct {
	ID   string `json:"id"`
	Desc bool   `json:"desc,omitempty"`
}

type TableFilterState struct {
	ID       string              `json:"id"`
	Value    any                 `json:"value"`
	Operator TableFilterOperator `json:"operator,omitempty"`
}
