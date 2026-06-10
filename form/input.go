package inputs

const (
	InputTypeSelect   = "select"
	InputTypeDate     = "date"
	InputTypeDatetime = "datetime"
	InputTypeText     = "text"
	InputTypeNumber   = "number"
	InputTypeCheckbox = "checkbox"
	InputTypeLabel    = "label"
	InputTypeSearch   = "search"
	InputTypeTextarea = "textarea"
	InputTypeHidden   = "hidden"
	InputTypeFile     = "file"
	InputTypeButton   = "button"
)

type Input struct {
	Id           string           `json:"id"`
	Type         string           `json:"type"`
	Name         string           `json:"name,omitempty"`
	Label        string           `json:"label,omitempty"`
	ActionID     string           `json:"actionId,omitempty"`
	Variant      string           `json:"variant,omitempty"` // button uchun: 'primary' | 'secondary' | 'destructive' | 'outline' | 'ghost' | 'link'
	FromName     string           `json:"fromName,omitempty"`
	ReadOnly     bool             `json:"readOnly,omitempty"`
	Placeholder  string           `json:"placeholder,omitempty"`
	Validation   *FieldValidation `json:"validation,omitempty"`
	MetaData     string           `json:"metaData,omitempty"` //search-view uchun malumot boradi qimat
	MetaKey      string           `json:"metaKey,omitempty"`  //search-view uchun malumot boradi kalit
	Format       string           `json:"format,omitempty"`
	Options      ComboItems       `json:"options,omitempty"`
	Visibility   bool             `json:"visibility,omitempty"`
	FieldActions []FieldAction    `json:"fieldActions,omitempty"`
	FileConfig   *FileConfig      `json:"fileConfig,omitempty"`
	ColSpan      int              `json:"colSpan,omitempty"`
	Hint         string           `json:"hint,omitempty"`
	SearchName   string           `json:"searchObject,omitempty"`
	DefaultValue string           `json:"defaultValue,omitempty"`
	Search       string           `json:"searchSource,omitempty"`
	DataType     string           `json:"dataType,omitempty"` //number,string,bool : default string
}

type ComboItem struct {
	ID   interface{} `json:"value"`
	Text interface{} `json:"label"`
}

type ComboItems []ComboItem
