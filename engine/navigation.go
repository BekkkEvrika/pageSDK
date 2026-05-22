package engine

// NavigationType enumerates explicit runtime navigation actions.
type NavigationType string

const (
	NavigationOpenPage        NavigationType = "openPage"
	NavigationClosePage       NavigationType = "closePage"
	NavigationOpenDialog      NavigationType = "openDialog"
	NavigationOpenTab         NavigationType = "openTab"
	NavigationCloseWithResult NavigationType = "closeWithResult"
)

// NavigationItem is an explicit frontend navigation instruction.
type NavigationItem struct {
	Type   NavigationType `json:"type"`
	Page   string         `json:"page,omitempty"`
	Params Params         `json:"params,omitempty"`
	Result any            `json:"result,omitempty"`
}

// NavigationWriter collects explicit navigation actions.
type NavigationWriter struct {
	items []NavigationItem
}

// OpenPage records a page navigation.
func (w *NavigationWriter) OpenPage(page string, params Params) {
	w.items = append(w.items, NavigationItem{Type: NavigationOpenPage, Page: page, Params: params})
}

// ClosePage records current page close.
func (w *NavigationWriter) ClosePage() {
	w.items = append(w.items, NavigationItem{Type: NavigationClosePage})
}

// OpenDialog records dialog navigation.
func (w *NavigationWriter) OpenDialog(page string, params Params) {
	w.items = append(w.items, NavigationItem{Type: NavigationOpenDialog, Page: page, Params: params})
}

// OpenTab records tab navigation.
func (w *NavigationWriter) OpenTab(page string, params Params) {
	w.items = append(w.items, NavigationItem{Type: NavigationOpenTab, Page: page, Params: params})
}

// CloseWithResult records close action with callback result.
func (w *NavigationWriter) CloseWithResult(result any) {
	w.items = append(w.items, NavigationItem{Type: NavigationCloseWithResult, Result: result})
}

// Items returns collected navigation actions.
func (w *NavigationWriter) Items() []NavigationItem {
	return append([]NavigationItem(nil), w.items...)
}
