package table

import (
	"fmt"

	"github.com/BekkkEvrika/pageSDK/engine"
)

// TableEventType identifies a supported table runtime event.
type TableEventType string

const (
	TableEventReload     TableEventType = "reload"
	TableEventFilter     TableEventType = "filter"
	TableEventPagination TableEventType = "pagination"
)

// TableEventHandler handles one table runtime event.
type TableEventHandler func(ctx *TableRuntimeContext)

// TableEventRegistrar is implemented by TableEngine.
type TableEventRegistrar interface {
	RegisterTableHandler(tableID string, event TableEventType, handler TableEventHandler)
}

// TableEventRequest is the typed client payload for table runtime events.
type TableEventRequest struct {
	PageIndex *int               `json:"pageIndex,omitempty"`
	PageSize  *int               `json:"pageSize,omitempty"`
	Filters   []TableFilterState `json:"filters,omitempty"`
	Params    map[string]any     `json:"params,omitempty"`
	Extra     map[string]any     `json:"extra,omitempty"`
}

// TableEventContext describes the table and state that triggered an event.
type TableEventContext struct {
	TableID   string             `json:"tableId"`
	Event     TableEventType     `json:"event"`
	PageIndex int                `json:"pageIndex,omitempty"`
	PageSize  int                `json:"pageSize,omitempty"`
	Filters   []TableFilterState `json:"filters,omitempty"`
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

// OpenDialog records dialog navigation.
func (ctx *TableRuntimeContext) OpenDialog(page string, params ...engine.Params) {
	ctx.Navigation = append(ctx.Navigation, engine.NavigationAction{
		Type:  engine.NavigationOpen,
		Mode:  engine.NavigationModeDialog,
		Page:  page,
		Extra: optionalExtra(params),
	})
}

// OpenTab records tab navigation.
func (ctx *TableRuntimeContext) OpenTab(page string, params ...engine.Params) {
	ctx.Navigation = append(ctx.Navigation, engine.NavigationAction{
		Type:  engine.NavigationOpen,
		Mode:  engine.NavigationModeTab,
		Page:  page,
		Extra: optionalExtra(params),
	})
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

func optionalExtra(params []engine.Params) map[string]any {
	if len(params) == 0 {
		return nil
	}
	extra := make(map[string]any, len(params[0]))
	for key, value := range params[0] {
		extra[key] = value
	}
	return extra
}
