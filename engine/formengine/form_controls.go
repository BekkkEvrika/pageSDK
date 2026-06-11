package formengine

import (
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
	c.engine.registerFormEvent(c.input.Type, c.input.Id, inputs.Change, c.input, func(ctx *RuntimeContext) { listener(ctx) })
}

func (c *Date) SetOnChange(listener ChangeListener) {
	c.engine.registerFormEvent(c.input.Type, c.input.Id, inputs.Change, c.input, func(ctx *RuntimeContext) { listener(ctx) })
}

func (c *Datetime) SetOnChange(listener ChangeListener) {
	c.engine.registerFormEvent(c.input.Type, c.input.Id, inputs.Change, c.input, func(ctx *RuntimeContext) { listener(ctx) })
}

func (c *Text) SetOnChange(listener ChangeListener) {
	c.engine.registerFormEvent(c.input.Type, c.input.Id, inputs.Change, c.input, func(ctx *RuntimeContext) { listener(ctx) })
}

func (c *Number) SetOnChange(listener ChangeListener) {
	c.engine.registerFormEvent(c.input.Type, c.input.Id, inputs.Change, c.input, func(ctx *RuntimeContext) { listener(ctx) })
}

func (c *Checkbox) SetOnChange(listener ChangeListener) {
	c.engine.registerFormEvent(c.input.Type, c.input.Id, inputs.Change, c.input, func(ctx *RuntimeContext) { listener(ctx) })
}

func (c *Search) SetOnChange(listener ChangeListener) {
	c.engine.registerFormEvent(c.input.Type, c.input.Id, inputs.Change, c.input, func(ctx *RuntimeContext) { listener(ctx) })
}

func (c *Textarea) SetOnChange(listener ChangeListener) {
	c.engine.registerFormEvent(c.input.Type, c.input.Id, inputs.Change, c.input, func(ctx *RuntimeContext) { listener(ctx) })
}

func (c *File) SetOnChange(listener ChangeListener) {
	c.engine.registerFormEvent(c.input.Type, c.input.Id, inputs.Change, c.input, func(ctx *RuntimeContext) { listener(ctx) })
}

func (c *Button) SetOnClick(listener ClickListener) {
	c.engine.registerFormEvent(c.input.Type, c.input.Id, inputs.Click, c.input, func(ctx *RuntimeContext) { listener(ctx) })
}

func (c *Label) Input() *inputs.Input {
	return c.control.Input()
}

func (c *Label) SetLabel(label string) {
	c.control.SetLabel(label)
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
