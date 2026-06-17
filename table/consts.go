package table

type TableColumnKind string

const (
	TableColumnKindAccessor TableColumnKind = "accessor"
	TableColumnKindDisplay  TableColumnKind = "display"
)

type TableColumnAlign string

const (
	TableColumnAlignLeft   TableColumnAlign = "left"
	TableColumnAlignCenter TableColumnAlign = "center"
	TableColumnAlignRight  TableColumnAlign = "right"
)

type TableColumnDataType string

const (
	TableColumnDataTypeString   TableColumnDataType = "string"
	TableColumnDataTypeNumber   TableColumnDataType = "number"
	TableColumnDataTypeBoolean  TableColumnDataType = "boolean"
	TableColumnDataTypeDate     TableColumnDataType = "date"
	TableColumnDataTypeDatetime TableColumnDataType = "datetime"
	TableColumnDataTypeCurrency TableColumnDataType = "currency"
	TableColumnDataTypeImage    TableColumnDataType = "image"
	TableColumnDataTypeFile     TableColumnDataType = "file"
	TableColumnDataTypeStatus   TableColumnDataType = "status"
	TableColumnDataTypeCustom   TableColumnDataType = "custom"
)

type TableColumnCellType string

const (
	TableColumnCellTypeText     TableColumnCellType = "text"
	TableColumnCellTypeNumber   TableColumnCellType = "number"
	TableColumnCellTypeBoolean  TableColumnCellType = "boolean"
	TableColumnCellTypeDate     TableColumnCellType = "date"
	TableColumnCellTypeDatetime TableColumnCellType = "datetime"
	TableColumnCellTypeCurrency TableColumnCellType = "currency"
	TableColumnCellTypeBadge    TableColumnCellType = "badge"
	TableColumnCellTypeStatus   TableColumnCellType = "status"
	TableColumnCellTypeImage    TableColumnCellType = "image"
	TableColumnCellTypeFile     TableColumnCellType = "file"
	TableColumnCellTypeLink     TableColumnCellType = "link"
	TableColumnCellTypeActions  TableColumnCellType = "actions"
	TableColumnCellTypeCustom   TableColumnCellType = "custom"
)

type TableColumnFormatType string

const (
	TableColumnFormatTypeCurrency TableColumnFormatType = "currency"
	TableColumnFormatTypeNumber   TableColumnFormatType = "number"
	TableColumnFormatTypeDate     TableColumnFormatType = "date"
	TableColumnFormatTypeDatetime TableColumnFormatType = "datetime"
)

type ActionVariant string

const (
	ActionVariantPrimary   ActionVariant = "primary"
	ActionVariantSecondary ActionVariant = "secondary"
	ActionVariantSuccess   ActionVariant = "success"
	ActionVariantWarning   ActionVariant = "warning"
	ActionVariantDanger    ActionVariant = "danger"
)

type HTTPMethod string

const (
	HTTPMethodGET    HTTPMethod = "GET"
	HTTPMethodPOST   HTTPMethod = "POST"
	HTTPMethodPUT    HTTPMethod = "PUT"
	HTTPMethodPATCH  HTTPMethod = "PATCH"
	HTTPMethodDELETE HTTPMethod = "DELETE"
)

type TableSelectionMode string

const (
	TableSelectionModeSingle   TableSelectionMode = "single"
	TableSelectionModeMultiple TableSelectionMode = "multiple"
)

type TableHotkeyScope string

const (
	TableHotkeyScopeTable     TableHotkeyScope = "table"
	TableHotkeyScopeRow       TableHotkeyScope = "row"
	TableHotkeyScopeSelection TableHotkeyScope = "selection"
	TableHotkeyScopeGlobal    TableHotkeyScope = "global"
)

type TableFilterOperator string

const (
	TableFilterOperatorEq         TableFilterOperator = "eq"
	TableFilterOperatorNeq        TableFilterOperator = "neq"
	TableFilterOperatorContains   TableFilterOperator = "contains"
	TableFilterOperatorStartsWith TableFilterOperator = "startsWith"
	TableFilterOperatorEndsWith   TableFilterOperator = "endsWith"
	TableFilterOperatorGt         TableFilterOperator = "gt"
	TableFilterOperatorGte        TableFilterOperator = "gte"
	TableFilterOperatorLt         TableFilterOperator = "lt"
	TableFilterOperatorLte        TableFilterOperator = "lte"
	TableFilterOperatorBetween    TableFilterOperator = "between"
	TableFilterOperatorIn         TableFilterOperator = "in"
	TableFilterOperatorNotIn      TableFilterOperator = "notIn"
)
