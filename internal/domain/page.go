package domain

import inputs "github.com/behzod/pageSDK/form"

// PageInstance — runtime-экземпляр страницы, основанной на forms.Form.
type PageInstance struct {
	PageID     string               `json:"pageId"`
	Form       inputs.Form          `json:"form"`
	Containers []*ContainerInstance `json:"containers"`
	State      *UIState             `json:"state"`
	Version    int64                `json:"version"`
}
