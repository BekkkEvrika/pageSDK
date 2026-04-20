package application

import (
	"context"
	"fmt"
	"time"

	"github.com/behzod/pageSDK/internal/domain"
	"github.com/behzod/pageSDK/internal/ports"
	"github.com/behzod/pageSDK/internal/runtime"
)

// StateManager управляет жизненным циклом UIState.
type StateManager struct {
	store ports.StateStore
}

func NewStateManager(store ports.StateStore) *StateManager {
	return &StateManager{store: store}
}

func (m *StateManager) GetOrCreate(ctx context.Context, sessionID, pageID string) (*domain.UIState, error) {
	state, err := m.store.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if state == nil {
		state = domain.NewUIState(sessionID, pageID)
		if err := m.store.Save(ctx, sessionID, state); err != nil {
			return nil, err
		}
	}
	return state, nil
}

func (m *StateManager) Save(ctx context.Context, state *domain.UIState) error {
	state.Version++
	state.UpdatedAt = time.Now()
	return m.store.Save(ctx, state.SessionID, state)
}

// ApplyFieldActions обрабатывает forms.FieldAction для компонента после изменения значения.
func (m *StateManager) ApplyFieldActions(state *domain.UIState, page *domain.PageInstance, componentID string) []domain.StateDiff {
	var diffs []domain.StateDiff

	// Найдём компонент в page
	comp := findComponentInPage(page, componentID)
	if comp == nil {
		return nil
	}

	for _, fa := range comp.Props.FieldActions {
		if !runtime.EvalRule(fa.When, state.Values) {
			continue
		}
		for _, targetID := range fa.TargetFields {
			diff := domain.StateDiff{ComponentID: targetID, Changes: make(map[string]interface{})}
			switch fa.Action {
			case "clear":
				delete(state.Values, targetID)
				diff.Changes["value"] = nil
			case "setRequired":
				// metadata — можно расширить
				diff.Changes["required"] = true
			case "setOptional":
				diff.Changes["required"] = false
			case "show":
				state.Visibility[targetID] = true
				diff.Changes["visible"] = true
			case "hide":
				state.Visibility[targetID] = false
				diff.Changes["visible"] = false
			case "setValue":
				val := fa.Value
				if fa.ValueRef != "" {
					val = state.Values[fa.ValueRef]
				}
				state.Values[targetID] = val
				diff.Changes["value"] = val
			}
			diffs = append(diffs, diff)
		}
	}
	return diffs
}

func findComponentInPage(page *domain.PageInstance, id string) *domain.ComponentInstance {
	for _, c := range page.Containers {
		if comp := findComponentInContainer(c, id); comp != nil {
			return comp
		}
	}
	return nil
}

func findComponentInContainer(c *domain.ContainerInstance, id string) *domain.ComponentInstance {
	for _, comp := range c.Components {
		if comp.ID == id {
			return comp
		}
	}
	for _, child := range c.Children {
		if comp := findComponentInContainer(child, id); comp != nil {
			return comp
		}
	}
	return nil
}

// ValidateComponent валидирует значение компонента по правилам forms.FieldValidation.
func ValidateComponent(comp *domain.ComponentInstance, value interface{}) []string {
	if comp.Props.Validation == nil {
		return nil
	}
	v := comp.Props.Validation
	var errs []string
	sVal := fmt.Sprintf("%v", value)
	if v.MinLength != nil && len(sVal) < *v.MinLength {
		msg := fmt.Sprintf("минимальная длина %d символов", *v.MinLength)
		if v.Message != "" {
			msg = v.Message
		}
		errs = append(errs, msg)
	}
	if v.MaxLength != nil && len(sVal) > *v.MaxLength {
		msg := fmt.Sprintf("максимальная длина %d символов", *v.MaxLength)
		if v.Message != "" {
			msg = v.Message
		}
		errs = append(errs, msg)
	}
	return errs
}
