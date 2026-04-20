package runtime

import (
	inputs "github.com/behzod/pageSDK/form"
	"github.com/behzod/pageSDK/internal/domain"
)

// PageLoader превращает forms.Form в domain.PageInstance.
type PageLoader struct{}

func NewPageLoader() *PageLoader { return &PageLoader{} }

// Load создаёт PageInstance из forms.Form, инициализируя видимость из defaultValue и rules.
func (l *PageLoader) Load(pageID string, form *inputs.Form, state *domain.UIState) *domain.PageInstance {
	var containers []*domain.ContainerInstance
	if form.Containers != nil {
		for i := range *form.Containers {
			ci := l.loadContainer(&(*form.Containers)[i], state)
			containers = append(containers, ci)
		}
	}
	return &domain.PageInstance{
		PageID:     pageID,
		Form:       *form,
		Containers: containers,
		State:      state,
	}
}

func (l *PageLoader) loadContainer(c *inputs.Container, state *domain.UIState) *domain.ContainerInstance {
	ci := &domain.ContainerInstance{
		ID:    c.Key,
		Props: *c,
	}

	// Видимость контейнера
	if c.VisibilityRule != nil {
		ci.Visible = EvalRule(*c.VisibilityRule, state.Values)
	} else {
		ci.Visible = true
	}

	// Поля
	for i := range c.Fields {
		comp := l.loadComponent(&c.Fields[i], state)
		ci.Components = append(ci.Components, comp)
	}

	// Дочерние контейнеры
	for i := range c.Containers {
		child := l.loadContainer(&c.Containers[i], state)
		ci.Children = append(ci.Children, child)
	}

	return ci
}

func (l *PageLoader) loadComponent(inp *inputs.Input, state *domain.UIState) *domain.ComponentInstance {
	visible := true
	if len(inp.VisibilityRules) > 0 {
		visible = EvalRules(inp.VisibilityRules, state.Values)
	}

	// Применяем сохранённое значение из state
	var value interface{}
	if v, ok := state.Values[inp.Id]; ok {
		value = v
	} else if inp.DefaultValue != "" {
		value = inp.DefaultValue
	}

	disabled := false
	if d, ok := state.Disabled[inp.Id]; ok {
		disabled = d
	}

	// Видимость из state имеет приоритет
	if sv, ok := state.Visibility[inp.Id]; ok {
		visible = sv
	}

	return &domain.ComponentInstance{
		ID:       inp.Id,
		Type:     inp.Type,
		Props:    *inp,
		Value:    value,
		Visible:  visible,
		Disabled: disabled,
		Errors:   state.Errors[inp.Id],
	}
}
