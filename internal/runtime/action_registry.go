package runtime

import (
	"context"
	"fmt"

	"github.com/behzod/pageSDK/internal/domain"
	"github.com/behzod/pageSDK/internal/ports"
)

// ActionRegistry хранит все зарегистрированные ActionExecutor по типу.
type ActionRegistry struct {
	executors map[domain.ActionType]ports.ActionExecutor
}

func NewActionRegistry() *ActionRegistry {
	return &ActionRegistry{executors: make(map[domain.ActionType]ports.ActionExecutor)}
}

// Register добавляет executor в реестр.
func (r *ActionRegistry) Register(e ports.ActionExecutor) {
	r.executors[e.Type()] = e
}

// Execute находит нужный executor и вызывает его.
func (r *ActionRegistry) Execute(ctx context.Context, action domain.Action, state *domain.UIState, event domain.UIEvent) (*domain.ActionResult, error) {
	e, ok := r.executors[action.Type]
	if !ok {
		return nil, fmt.Errorf("no executor for action type: %s", action.Type)
	}
	return e.Execute(ctx, action, state, event)
}
