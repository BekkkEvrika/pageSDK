package tableengine

import "github.com/BekkkEvrika/pageSDK/table"

type TableEventType = table.TableEventType
type TableEventRequest = table.TableEventRequest
type TableRowActionRequest = table.TableRowActionRequest
type TableColumnActionRequest = table.TableColumnActionRequest
type TableSelectedActionRequest = table.TableSelectedActionRequest
type TableEventContext = table.TableEventContext
type TableRuntimeContext = table.TableRuntimeContext
type TableEventHandler = table.TableEventHandler

const (
	TableEventReload         = table.TableEventReload
	TableEventFilter         = table.TableEventFilter
	TableEventPagination     = table.TableEventPagination
	TableEventRowAction      = table.TableEventRowAction
	TableEventToolbarAction  = table.TableEventToolbarAction
	TableEventColumnAction   = table.TableEventColumnAction
	TableEventSelectedAction = table.TableEventSelectedAction
)
