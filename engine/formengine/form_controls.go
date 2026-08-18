package formengine

import (
	"github.com/BekkkEvrika/pageSDK/access"
	inputs "github.com/BekkkEvrika/pageSDK/form"
)

type ClickListener func(ctx *RuntimeContext)
type ChangeListener func(ctx *RuntimeContext)

// FormControl is a typed runtime handle over an Input stored inside FormEngine DSL.
type FormControl struct {
	engine *FormEngine
	input  *inputs.Input
}

func newFormControl(engine *FormEngine, input *inputs.Input) FormControl {
	return FormControl{engine: engine, input: input}
}

func (c *FormControl) Input() *inputs.Input {
	return c.input
}

func (c *FormControl) SetName(name string) {
	c.input.Name = name
}

func (c *FormControl) SetLabel(label string) {
	c.input.Label = label
}

func (c *FormControl) SetActionID(actionID string) {
	c.input.ActionID = actionID
}

func (c *FormControl) SetVariant(variant string) {
	c.input.Variant = variant
}

func (c *FormControl) SetFromName(fromName string) {
	c.input.FromName = fromName
}

func (c *FormControl) SetReadOnly(readOnly bool) {
	c.input.ReadOnly = readOnly
}

func (c *FormControl) SetPlaceholder(placeholder string) {
	c.input.Placeholder = placeholder
}

func (c *FormControl) SetValidation(validation *inputs.FieldValidation) {
	c.input.Validation = validation
}

func (c *FormControl) SetMetaData(metaData string) {
	c.input.MetaData = metaData
}

func (c *FormControl) SetMetaKey(metaKey string) {
	c.input.MetaKey = metaKey
}

func (c *FormControl) SetFormat(format string) {
	c.input.Format = format
}

func (c *FormControl) SetOptions(options inputs.ComboItems) {
	c.input.Options = options
}

func (c *FormControl) SetVisibility(visibility bool) {
	c.input.Visibility = visibility
}

func (c *FormControl) SetFieldActions(actions []inputs.FieldAction) {
	c.input.FieldActions = actions
}

func (c *FormControl) SetFileConfig(config *inputs.FileConfig) {
	c.input.FileConfig = config
}

func (c *FormControl) SetColSpan(colSpan int) {
	c.input.ColSpan = colSpan
}

func (c *FormControl) SetHint(hint string) {
	c.input.Hint = hint
}

func (c *FormControl) SetSearchName(searchName string) {
	c.input.SearchName = searchName
}

func (c *FormControl) SetDefaultValue(defaultValue any) {
	c.input.DefaultValue = defaultValue
}

func (c *FormControl) SetSearch(search string) {
	c.input.Search = search
}

func (c *FormControl) SetDataType(dataType string) {
	c.input.DataType = dataType
}

func (c *FormControl) SetAccess(group access.AccessGroup, behavior access.NoAccessBehavior) {
	c.input.AccessGroupCode = group.Code
	if c.input.ElementCode == "" {
		c.input.ElementCode = c.input.Id
	}
	c.input.NoAccessBehavior = string(behavior)
}

// Name sets the submitted field name.
func (c *FormControl) Name(name string) *FormControl {
	c.SetName(name)
	return c
}

// Label sets the visible field label.
func (c *FormControl) Label(label string) *FormControl {
	c.SetLabel(label)
	return c
}

// ActionID sets the action identifier associated with the field.
func (c *FormControl) ActionID(actionID string) *FormControl {
	c.SetActionID(actionID)
	return c
}

// Variant sets the visual field variant.
func (c *FormControl) Variant(variant string) *FormControl {
	c.SetVariant(variant)
	return c
}

// FromName sets the source form name.
func (c *FormControl) FromName(fromName string) *FormControl {
	c.SetFromName(fromName)
	return c
}

// ReadOnly sets whether the field can be edited.
func (c *FormControl) ReadOnly(readOnly bool) *FormControl {
	c.SetReadOnly(readOnly)
	return c
}

// Placeholder sets the field placeholder.
func (c *FormControl) Placeholder(placeholder string) *FormControl {
	c.SetPlaceholder(placeholder)
	return c
}

// Validation sets field validation rules.
func (c *FormControl) Validation(validation *inputs.FieldValidation) *FormControl {
	c.SetValidation(validation)
	return c
}

// MetaData sets field metadata.
func (c *FormControl) MetaData(metaData string) *FormControl {
	c.SetMetaData(metaData)
	return c
}

// MetaKey sets the field metadata key.
func (c *FormControl) MetaKey(metaKey string) *FormControl {
	c.SetMetaKey(metaKey)
	return c
}

// Format sets the field display format.
func (c *FormControl) Format(format string) *FormControl {
	c.SetFormat(format)
	return c
}

// Options sets selectable field options.
func (c *FormControl) Options(options inputs.ComboItems) *FormControl {
	c.SetOptions(options)
	return c
}

// Visible sets field visibility.
func (c *FormControl) Visible(visible bool) *FormControl {
	c.SetVisibility(visible)
	return c
}

// FieldActions sets declarative field actions.
func (c *FormControl) FieldActions(actions []inputs.FieldAction) *FormControl {
	c.SetFieldActions(actions)
	return c
}

// FileConfig sets upload configuration.
func (c *FormControl) FileConfig(config *inputs.FileConfig) *FormControl {
	c.SetFileConfig(config)
	return c
}

// ColSpan sets the field grid span.
func (c *FormControl) ColSpan(colSpan int) *FormControl {
	c.SetColSpan(colSpan)
	return c
}

// Hint sets helper text for the field.
func (c *FormControl) Hint(hint string) *FormControl {
	c.SetHint(hint)
	return c
}

// SearchName sets the search object name.
func (c *FormControl) SearchName(searchName string) *FormControl {
	c.SetSearchName(searchName)
	return c
}

// DefaultValue sets the initial field value.
func (c *FormControl) DefaultValue(defaultValue any) *FormControl {
	c.SetDefaultValue(defaultValue)
	return c
}

// Search sets the field search source.
func (c *FormControl) Search(search string) *FormControl {
	c.SetSearch(search)
	return c
}

// DataType sets the field data type.
func (c *FormControl) DataType(dataType string) *FormControl {
	c.SetDataType(dataType)
	return c
}

// Access binds this UI element to a registered SFP access group.
// The group must exist in the Application access registry; access generate
// fails for unknown group codes instead of creating new groups from typos.
func (c *FormControl) Access(group access.AccessGroup, behavior access.NoAccessBehavior) *FormControl {
	c.SetAccess(group, behavior)
	return c
}

// OnChange registers a change listener for this field.
func (c *FormControl) OnChange(listener ChangeListener) *FormControl {
	if listener != nil {
		c.engine.registerFormEvent(c.input.Type, c.input.Id, inputs.Change, c.input, func(ctx *RuntimeContext) {
			listener(ctx)
		})
	}
	return c
}

// OnClick registers a click listener for this field.
func (c *FormControl) OnClick(listener ClickListener) *FormControl {
	if listener != nil {
		c.engine.registerFormEvent(c.input.Type, c.input.Id, inputs.Click, c.input, func(ctx *RuntimeContext) {
			listener(ctx)
		})
	}
	return c
}

type Select struct{ FormControl }
type Date struct{ FormControl }
type Datetime struct{ FormControl }
type Text struct{ FormControl }
type Number struct{ FormControl }
type Checkbox struct{ FormControl }
type Label struct {
	control FormControl
}
type Search struct{ FormControl }
type Textarea struct{ FormControl }
type Hidden struct{ FormControl }
type File struct{ FormControl }
type Button struct{ FormControl }

func (c *Select) SetOnChange(listener ChangeListener) {
	c.FormControl.OnChange(listener)
}

func (c *Date) SetOnChange(listener ChangeListener) {
	c.FormControl.OnChange(listener)
}

func (c *Datetime) SetOnChange(listener ChangeListener) {
	c.FormControl.OnChange(listener)
}

func (c *Text) SetOnChange(listener ChangeListener) {
	c.FormControl.OnChange(listener)
}

func (c *Number) SetOnChange(listener ChangeListener) {
	c.FormControl.OnChange(listener)
}

func (c *Checkbox) SetOnChange(listener ChangeListener) {
	c.FormControl.OnChange(listener)
}

func (c *Search) SetOnChange(listener ChangeListener) {
	c.FormControl.OnChange(listener)
}

func (c *Textarea) SetOnChange(listener ChangeListener) {
	c.FormControl.OnChange(listener)
}

func (c *File) SetOnChange(listener ChangeListener) {
	c.FormControl.OnChange(listener)
}

func (c *Button) SetOnClick(listener ClickListener) {
	c.FormControl.OnClick(listener)
}

func (c *Label) Input() *inputs.Input {
	return c.control.Input()
}

func (c *Label) SetLabel(label string) {
	c.control.SetLabel(label)
}

// Label sets the visible label text.
func (c *Label) Label(label string) *Label {
	c.SetLabel(label)
	return c
}

func newSelect(engine *FormEngine, input *inputs.Input) *Select {
	return &Select{FormControl: newFormControl(engine, input)}
}

func newDate(engine *FormEngine, input *inputs.Input) *Date {
	return &Date{FormControl: newFormControl(engine, input)}
}

func newDatetime(engine *FormEngine, input *inputs.Input) *Datetime {
	return &Datetime{FormControl: newFormControl(engine, input)}
}

func newText(engine *FormEngine, input *inputs.Input) *Text {
	return &Text{FormControl: newFormControl(engine, input)}
}

func newNumber(engine *FormEngine, input *inputs.Input) *Number {
	return &Number{FormControl: newFormControl(engine, input)}
}

func newCheckbox(engine *FormEngine, input *inputs.Input) *Checkbox {
	return &Checkbox{FormControl: newFormControl(engine, input)}
}

func newLabel(engine *FormEngine, input *inputs.Input) *Label {
	return &Label{control: newFormControl(engine, input)}
}

func newSearch(engine *FormEngine, input *inputs.Input) *Search {
	return &Search{FormControl: newFormControl(engine, input)}
}

func newTextarea(engine *FormEngine, input *inputs.Input) *Textarea {
	return &Textarea{FormControl: newFormControl(engine, input)}
}

func newHidden(engine *FormEngine, input *inputs.Input) *Hidden {
	return &Hidden{FormControl: newFormControl(engine, input)}
}

func newFile(engine *FormEngine, input *inputs.Input) *File {
	return &File{FormControl: newFormControl(engine, input)}
}

func newButton(engine *FormEngine, input *inputs.Input) *Button {
	return &Button{FormControl: newFormControl(engine, input)}
}
