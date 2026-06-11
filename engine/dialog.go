package engine

// DialogLevel enumerates client-side dialog severity levels.
type DialogLevel string

const (
	DialogInfo    DialogLevel = "info"
	DialogWarning DialogLevel = "warning"
	DialogError   DialogLevel = "error"
	DialogSuccess DialogLevel = "success"
)

// DialogAction describes one action button in a client-side dialog.
type DialogAction struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	URL    string `json:"url,omitempty"`
	Method string `json:"method,omitempty"`
}

// Dialog describes a client-side message dialog requested by runtime code.
type Dialog struct {
	Title       string         `json:"title"`
	Description string         `json:"description,omitempty"`
	Level       DialogLevel    `json:"level"`
	Actions     []DialogAction `json:"actions,omitempty"`
}
