package ports

import (
	"context"

	inputs "github.com/behzod/pageSDK/form"
	"github.com/behzod/pageSDK/internal/domain"
)

// StateStore — порт для хранения UI-состояния (Redis / in-memory).
type StateStore interface {
	Get(ctx context.Context, sessionID string) (*domain.UIState, error)
	Save(ctx context.Context, sessionID string, state *domain.UIState) error
	Delete(ctx context.Context, sessionID string) error
}

// PageRegistry — порт реестра страниц.
type PageRegistry interface {
	Get(pageID string) (*inputs.Form, error)
	Register(pageID string, form *inputs.Form) error
}

// ActionExecutor — порт выполнения конкретного action-а.
type ActionExecutor interface {
	// Type возвращает тип action-а, который обрабатывает этот executor.
	Type() domain.ActionType
	Execute(ctx context.Context, action domain.Action, state *domain.UIState, event domain.UIEvent) (*domain.ActionResult, error)
}

// UseCaseRunner — порт вызова бизнес-логики из action-а.
type UseCaseRunner interface {
	Run(ctx context.Context, name string, params map[string]interface{}, state *domain.UIState) (map[string]interface{}, error)
}
