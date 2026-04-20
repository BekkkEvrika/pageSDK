package domain

import inputs "github.com/behzod/pageSDK/form"

// ComponentInstance — runtime-экземпляр компонента (поверх inputs.Input).
// Хранит DSL-пропсы из forms и runtime-состояние.
type ComponentInstance struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Props    inputs.Input `json:"props"`
	Value    interface{}  `json:"value,omitempty"`
	Visible  bool         `json:"visible"`
	Disabled bool         `json:"disabled"`
	Errors   []string     `json:"errors,omitempty"`
}

// ContainerInstance — runtime-обёртка над inputs.Container.
type ContainerInstance struct {
	ID         string               `json:"id"`
	Props      inputs.Container     `json:"props"`
	Visible    bool                 `json:"visible"`
	Components []*ComponentInstance `json:"components,omitempty"`
	Children   []*ContainerInstance `json:"children,omitempty"`
}
