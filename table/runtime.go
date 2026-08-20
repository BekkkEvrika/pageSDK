package table

import (
	"fmt"

	"github.com/BekkkEvrika/pageSDK/engine"
)

// TableEventType identifies a supported table runtime event.
type TableEventType string

const (
	TableEventReload         TableEventType = "reload"
	TableEventFilter         TableEventType = "filter"
	TableEventPagination     TableEventType = "pagination"
	TableEventRowAction      TableEventType = "rowAction"
	TableEventToolbarAction  TableEventType = "toolbarAction"
	TableEventColumnAction   TableEventType = "columnAction"
	TableEventSelectedAction TableEventType = "selectedAction"
)

// TableEventHandler handles one table runtime event.
type TableEventHandler func(ctx *TableRuntimeContext)

// NavigationCallback handles a result returned from a page opened by a table event.
type NavigationCallback func(ctx *TableRuntimeContext)

// OpenOptions describes frontend-owned navigation state for opening another page.
type OpenOptions struct {
	Extra    map[string]any
	Callback NavigationCallback
}

// TableEventRegistrar is implemented by TableEngine.
type TableEventRegistrar interface {
	RegisterTableHandler(tableID string, event TableEventType, handler TableEventHandler)
	RegisterRowActionHandler(tableID, actionID string, handler TableEventHandler)
}

// TableToolbarActionRegistrar is implemented by runtimes that support toolbar actions.
type TableToolbarActionRegistrar interface {
	RegisterToolbarActionHandler(tableID, actionID string, handler TableEventHandler)
}

// TableColumnActionRegistrar is implemented by runtimes that support actions
// bound to one concrete table column.
type TableColumnActionRegistrar interface {
	RegisterColumnActionHandler(tableID, columnID, actionID string, handler TableEventHandler)
}

// TableSelectedActionRegistrar is implemented by runtimes that support selected-row actions.
type TableSelectedActionRegistrar interface {
	RegisterSelectedActionHandler(tableID, actionID string, handler TableEventHandler)
}

// TableEventRequest is the typed client payload for table runtime events.
type TableEventRequest struct {
	PageIndex *int               `json:"pageIndex,omitempty"`
	PageSize  *int               `json:"pageSize,omitempty"`
	Filters   []TableFilterState `json:"filters,omitempty"`
	Params    map[string]any     `json:"params,omitempty"`
	Extra     map[string]any     `json:"extra,omitempty"`
}

// TableRowActionRequest is the typed client payload for row actions.
type TableRowActionRequest struct {
	Row    map[string]any `json:"row"`
	Params map[string]any `json:"params,omitempty"`
	Extra  map[string]any `json:"extra,omitempty"`
}

// TableColumnActionRequest contains current values of one column keyed by row id.
type TableColumnActionRequest struct {
	Column map[string]any `json:"column"`
}

// TableSelectedActionRequest contains selected row-id values.
type TableSelectedActionRequest struct {
	SelectedRows []string `json:"selectedRows"`
}

// TableEventContext describes the table and state that triggered an event.
type TableEventContext struct {
	TableID      string             `json:"tableId"`
	Event        TableEventType     `json:"event"`
	ActionID     string             `json:"actionId,omitempty"`
	ColumnID     string             `json:"columnId,omitempty"`
	Row          map[string]any     `json:"row,omitempty"`
	Column       map[string]any     `json:"column,omitempty"`
	SelectedRows []string           `json:"selectedRows,omitempty"`
	PageIndex    int                `json:"pageIndex,omitempty"`
	PageSize     int                `json:"pageSize,omitempty"`
	Filters      []TableFilterState `json:"filters,omitempty"`
}

// TableRuntimeContext is used only by TableEngine handlers.
type TableRuntimeContext struct {
	State      TableStateConfig
	User       engine.User
	System     engine.SystemKeys
	Params     map[string]any
	Extra      map[string]any
	EventTable *TableEventContext
	Mutations  []engine.Mutation
	Navigation []engine.NavigationAction
	Err        error
	callback   func(NavigationCallback) string
}

// RuntimeTable is a mutation handle for one table.
type RuntimeTable struct {
	ctx     *TableRuntimeContext
	tableID string
}

// Table returns a runtime mutation handle for tableID.
func (ctx *TableRuntimeContext) Table(tableID string) *RuntimeTable {
	return &RuntimeTable{ctx: ctx, tableID: tableID}
}

// SetData records a table data update.
func (t *RuntimeTable) SetData(data TableData) {
	if t == nil || t.ctx == nil {
		return
	}
	if t.tableID == "" {
		t.ctx.SetError(fmt.Errorf("table runtime: table id is required"))
		return
	}
	t.ctx.Mutations = append(t.ctx.Mutations, engine.Mutation{
		Type:  engine.MutationUpdate,
		Path:  "tables." + t.tableID + ".data",
		Value: data,
	})
}

// SetError records the first handler error.
func (ctx *TableRuntimeContext) SetError(err error) {
	if err != nil && ctx.Err == nil {
		ctx.Err = err
	}
}

// Error returns the first handler error.
func (ctx *TableRuntimeContext) Error() error {
	if ctx == nil {
		return nil
	}
	return ctx.Err
}

// SetNavigationCallbackRegistrar configures callback URL registration for the
// runtime engine. Application code normally does not need to call this method.
func (ctx *TableRuntimeContext) SetNavigationCallbackRegistrar(registrar func(NavigationCallback) string) {
	ctx.callback = registrar
}

// OpenDialog records dialog navigation.
func (ctx *TableRuntimeContext) OpenDialog(page string, options ...any) {
	ctx.open(page, engine.NavigationModeDialog, options...)
}

// OpenTab records tab navigation.
func (ctx *TableRuntimeContext) OpenTab(page string, options ...any) {
	ctx.open(page, engine.NavigationModeTab, options...)
}

// Close records current page close.
func (ctx *TableRuntimeContext) Close() {
	ctx.Navigation = append(ctx.Navigation, engine.NavigationAction{Type: engine.NavigationClose})
}

// CloseWithResult records current page close with a result.
func (ctx *TableRuntimeContext) CloseWithResult(result any) {
	ctx.Navigation = append(ctx.Navigation, engine.NavigationAction{
		Type:   engine.NavigationClose,
		Result: result,
	})
}

func (ctx *TableRuntimeContext) open(page string, mode engine.NavigationMode, options ...any) {
	openOptions, err := normalizeOpenOptions(options...)
	if err != nil {
		ctx.SetError(err)
		return
	}
	action := engine.NavigationAction{
		Type:  engine.NavigationOpen,
		Mode:  mode,
		Page:  page,
		Extra: openOptions.Extra,
	}
	if openOptions.Callback != nil {
		if ctx.callback == nil {
			ctx.SetError(fmt.Errorf("table runtime: navigation callback registrar is not configured"))
			return
		}
		action.Callback = ctx.callback(openOptions.Callback)
	}
	ctx.Navigation = append(ctx.Navigation, action)
}

func normalizeOpenOptions(options ...any) (OpenOptions, error) {
	if len(options) == 0 || options[0] == nil {
		return OpenOptions{}, nil
	}
	if len(options) > 1 {
		return OpenOptions{}, fmt.Errorf("table runtime: expected one open options argument, got %d", len(options))
	}
	switch option := options[0].(type) {
	case OpenOptions:
		return option, nil
	case *OpenOptions:
		if option == nil {
			return OpenOptions{}, nil
		}
		return *option, nil
	case engine.Params:
		extra := make(map[string]any, len(option))
		for key, value := range option {
			extra[key] = value
		}
		return OpenOptions{Extra: extra}, nil
	case map[string]any:
		return OpenOptions{Extra: option}, nil
	default:
		return OpenOptions{}, fmt.Errorf("table runtime: unsupported open options type %T", option)
	}
}
