package engine

// NavigationType enumerates explicit runtime navigation actions.
type NavigationType string

const (
	NavigationOpen            NavigationType = "open"
	NavigationClose           NavigationType = "close"
	NavigationOpenPage        NavigationType = "open"
	NavigationClosePage       NavigationType = "close"
	NavigationOpenDialog      NavigationType = "open"
	NavigationOpenTab         NavigationType = "open"
	NavigationCloseWithResult NavigationType = "close"
)

// NavigationMode describes how an opened page should be presented.
type NavigationMode string

const (
	NavigationModePage   NavigationMode = "page"
	NavigationModeDialog NavigationMode = "dialog"
	NavigationModeTab    NavigationMode = "tab"
)

// NavigationAction is an explicit frontend navigation instruction.
type NavigationAction struct {
	Type     NavigationType `json:"type"`
	Mode     NavigationMode `json:"mode,omitempty"`
	Page     string         `json:"page,omitempty"`
	Extra    map[string]any `json:"extra,omitempty"`
	Callback string         `json:"callback,omitempty"`
	Result   any            `json:"result,omitempty"`
}

// NavigationItem is kept as a source-compatible alias.
type NavigationItem = NavigationAction

// NavigationWriter collects explicit navigation actions.
type NavigationWriter struct {
	items []NavigationAction
}

// OpenPage records a page navigation.
func (w *NavigationWriter) OpenPage(page string, params Params) {
	w.items = append(w.items, NavigationAction{Type: NavigationOpen, Mode: NavigationModePage, Page: page, Extra: paramsToExtra(params)})
}

// ClosePage records current page close.
func (w *NavigationWriter) ClosePage() {
	w.items = append(w.items, NavigationAction{Type: NavigationClose})
}

// OpenDialog records dialog navigation.
func (w *NavigationWriter) OpenDialog(page string, params Params) {
	w.items = append(w.items, NavigationAction{Type: NavigationOpen, Mode: NavigationModeDialog, Page: page, Extra: paramsToExtra(params)})
}

// OpenTab records tab navigation.
func (w *NavigationWriter) OpenTab(page string, params Params) {
	w.items = append(w.items, NavigationAction{Type: NavigationOpen, Mode: NavigationModeTab, Page: page, Extra: paramsToExtra(params)})
}

// CloseWithResult records close action with callback result.
func (w *NavigationWriter) CloseWithResult(result any) {
	w.items = append(w.items, NavigationAction{Type: NavigationClose, Result: result})
}

// Items returns collected navigation actions.
func (w *NavigationWriter) Items() []NavigationAction {
	return append([]NavigationAction(nil), w.items...)
}

func paramsToExtra(params Params) map[string]any {
	if len(params) == 0 {
		return nil
	}
	extra := make(map[string]any, len(params))
	for key, value := range params {
		extra[key] = value
	}
	return extra
}
