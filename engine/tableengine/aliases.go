package tableengine

import "github.com/BekkkEvrika/pageSDK/table"

type TableSchema = table.TableSchema
type TableColumnSchema = table.TableColumnSchema
type TableData = table.TableData
type TableActionGroups = table.TableActionGroups
type ActionSchema = table.ActionSchema
type TableFeatureConfig = table.TableFeatureConfig
type TableSelectionSchema = table.TableSelectionSchema
type TableHotkeySchema = table.TableHotkeySchema
type TableColumnFormat = table.TableColumnFormat
type TableStateConfig = table.TableStateConfig
type TableSortingItem = table.TableSortingItem
type TableFilterState = table.TableFilterState

type TableColumnKind = table.TableColumnKind
type TableColumnAlign = table.TableColumnAlign
type TableColumnDataType = table.TableColumnDataType
type TableColumnCellType = table.TableColumnCellType
type TableColumnFormatType = table.TableColumnFormatType
type ActionVariant = table.ActionVariant
type HTTPMethod = table.HTTPMethod
type TableSelectionMode = table.TableSelectionMode
type TableHotkeyScope = table.TableHotkeyScope
type TableFilterOperator = table.TableFilterOperator

const (
	TableColumnKindAccessor = table.TableColumnKindAccessor
	TableColumnKindDisplay  = table.TableColumnKindDisplay

	TableColumnAlignLeft   = table.TableColumnAlignLeft
	TableColumnAlignCenter = table.TableColumnAlignCenter
	TableColumnAlignRight  = table.TableColumnAlignRight

	TableColumnDataTypeString   = table.TableColumnDataTypeString
	TableColumnDataTypeNumber   = table.TableColumnDataTypeNumber
	TableColumnDataTypeBoolean  = table.TableColumnDataTypeBoolean
	TableColumnDataTypeDate     = table.TableColumnDataTypeDate
	TableColumnDataTypeDatetime = table.TableColumnDataTypeDatetime
	TableColumnDataTypeCurrency = table.TableColumnDataTypeCurrency
	TableColumnDataTypeImage    = table.TableColumnDataTypeImage
	TableColumnDataTypeFile     = table.TableColumnDataTypeFile
	TableColumnDataTypeStatus   = table.TableColumnDataTypeStatus
	TableColumnDataTypeCustom   = table.TableColumnDataTypeCustom

	TableColumnCellTypeText     = table.TableColumnCellTypeText
	TableColumnCellTypeNumber   = table.TableColumnCellTypeNumber
	TableColumnCellTypeBoolean  = table.TableColumnCellTypeBoolean
	TableColumnCellTypeDate     = table.TableColumnCellTypeDate
	TableColumnCellTypeDatetime = table.TableColumnCellTypeDatetime
	TableColumnCellTypeCurrency = table.TableColumnCellTypeCurrency
	TableColumnCellTypeBadge    = table.TableColumnCellTypeBadge
	TableColumnCellTypeStatus   = table.TableColumnCellTypeStatus
	TableColumnCellTypeImage    = table.TableColumnCellTypeImage
	TableColumnCellTypeFile     = table.TableColumnCellTypeFile
	TableColumnCellTypeLink     = table.TableColumnCellTypeLink
	TableColumnCellTypeActions  = table.TableColumnCellTypeActions
	TableColumnCellTypeCustom   = table.TableColumnCellTypeCustom

	TableColumnFormatTypeCurrency = table.TableColumnFormatTypeCurrency
	TableColumnFormatTypeNumber   = table.TableColumnFormatTypeNumber
	TableColumnFormatTypeDate     = table.TableColumnFormatTypeDate
	TableColumnFormatTypeDatetime = table.TableColumnFormatTypeDatetime

	ActionVariantPrimary   = table.ActionVariantPrimary
	ActionVariantSecondary = table.ActionVariantSecondary
	ActionVariantSuccess   = table.ActionVariantSuccess
	ActionVariantWarning   = table.ActionVariantWarning
	ActionVariantDanger    = table.ActionVariantDanger

	HTTPMethodGET    = table.HTTPMethodGET
	HTTPMethodPOST   = table.HTTPMethodPOST
	HTTPMethodPUT    = table.HTTPMethodPUT
	HTTPMethodPATCH  = table.HTTPMethodPATCH
	HTTPMethodDELETE = table.HTTPMethodDELETE

	TableSelectionModeSingle   = table.TableSelectionModeSingle
	TableSelectionModeMultiple = table.TableSelectionModeMultiple

	TableHotkeyScopeTable     = table.TableHotkeyScopeTable
	TableHotkeyScopeRow       = table.TableHotkeyScopeRow
	TableHotkeyScopeSelection = table.TableHotkeyScopeSelection
	TableHotkeyScopeGlobal    = table.TableHotkeyScopeGlobal

	TableFilterOperatorEq         = table.TableFilterOperatorEq
	TableFilterOperatorNeq        = table.TableFilterOperatorNeq
	TableFilterOperatorContains   = table.TableFilterOperatorContains
	TableFilterOperatorStartsWith = table.TableFilterOperatorStartsWith
	TableFilterOperatorEndsWith   = table.TableFilterOperatorEndsWith
	TableFilterOperatorGt         = table.TableFilterOperatorGt
	TableFilterOperatorGte        = table.TableFilterOperatorGte
	TableFilterOperatorLt         = table.TableFilterOperatorLt
	TableFilterOperatorLte        = table.TableFilterOperatorLte
	TableFilterOperatorBetween    = table.TableFilterOperatorBetween
	TableFilterOperatorIn         = table.TableFilterOperatorIn
	TableFilterOperatorNotIn      = table.TableFilterOperatorNotIn
)
