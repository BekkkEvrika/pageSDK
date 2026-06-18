package tableengine

import "github.com/BekkkEvrika/pageSDK/table"

type TableEventType = table.TableEventType
type TableEventRequest = table.TableEventRequest
type TableEventContext = table.TableEventContext
type TableRuntimeContext = table.TableRuntimeContext
type TableEventHandler = table.TableEventHandler

const (
	TableEventReload     = table.TableEventReload
	TableEventFilter     = table.TableEventFilter
	TableEventPagination = table.TableEventPagination
)
