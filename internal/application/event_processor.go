package application

import (
	"context"
	"fmt"

	"github.com/behzod/pageSDK/internal/domain"
	"github.com/behzod/pageSDK/internal/ports"
	"github.com/behzod/pageSDK/internal/runtime"
)

// EventProcessor — ядро обработки событий.
type EventProcessor struct {
	stateManager *StateManager
	pageRegistry ports.PageRegistry
	actionReg    *runtime.ActionRegistry
	pageLoader   *runtime.PageLoader
	// bindings хранит event bindings, добавленные программно
	bindings map[string][]domain.EventBinding // ключ: pageID
}

func NewEventProcessor(
	sm *StateManager,
	reg ports.PageRegistry,
	ar *runtime.ActionRegistry,
	loader *runtime.PageLoader,
) *EventProcessor {
	return &EventProcessor{
		stateManager: sm,
		pageRegistry: reg,
		actionReg:    ar,
		pageLoader:   loader,
		bindings:     make(map[string][]domain.EventBinding),
	}
}

// RegisterBinding добавляет runtime event binding для страницы.
func (p *EventProcessor) RegisterBinding(pageID string, binding domain.EventBinding) {
	p.bindings[pageID] = append(p.bindings[pageID], binding)
}

// Process — главный метод обработки события.
// Возвращает список diff-ов, возможную навигацию и список ошибок.
func (p *EventProcessor) Process(ctx context.Context, event domain.UIEvent) (*ProcessResult, error) {
	// 1. Загрузка формы
	form, err := p.pageRegistry.Get(event.PageID)
	if err != nil {
		return nil, fmt.Errorf("PAGE_NOT_FOUND: %w", err)
	}

	// 2. Загрузка / создание state
	state, err := p.stateManager.GetOrCreate(ctx, event.SessionID, event.PageID)
	if err != nil {
		return nil, err
	}

	// 3. Загрузка PageInstance
	page := p.pageLoader.Load(event.PageID, form, state)

	// 4. Применяем входящее значение из payload (если есть)
	if val, ok := event.Payload["value"]; ok {
		state.Values[event.ComponentID] = val

		// Валидация компонента
		comp := findComponentInPage(page, event.ComponentID)
		if comp != nil {
			if errs := ValidateComponent(comp, val); len(errs) > 0 {
				state.Errors[event.ComponentID] = errs
			} else {
				delete(state.Errors, event.ComponentID)
			}
		}
	}

	var allDiffs []domain.StateDiff
	var navigateTo string
	var appErrors []domain.AppError

	// 5. Применяем FieldActions из DSL
	// Перезагружаем page с обновлённым state
	page = p.pageLoader.Load(event.PageID, form, state)
	diffs := p.stateManager.ApplyFieldActions(state, page, event.ComponentID)
	allDiffs = append(allDiffs, diffs...)

	// 6. Ищем и выполняем runtime bindings
	for _, b := range p.bindings[event.PageID] {
		if b.ComponentID == event.ComponentID && b.EventType == event.EventType {
			for _, action := range b.Actions {
				result, err := p.actionReg.Execute(ctx, action, state, event)
				if err != nil {
					appErrors = append(appErrors, domain.AppError{Code: "ACTION_ERROR", Message: err.Error()})
					continue
				}
				allDiffs = append(allDiffs, result.Diffs...)
				if result.NavigateTo != "" {
					navigateTo = result.NavigateTo
				}
				appErrors = append(appErrors, result.Errors...)
			}
		}
	}

	// 7. Ищем FormActions из DSL (init/change/click)
	allDiffs = append(allDiffs, p.processDSLFormActions(ctx, form, event, state)...)

	// 8. Пересчёт visibility всех компонентов
	visibilityDiffs := p.recalcVisibility(form, state)
	allDiffs = append(allDiffs, visibilityDiffs...)

	// 9. Сохраняем state
	if err := p.stateManager.Save(ctx, state); err != nil {
		return nil, err
	}

	return &ProcessResult{
		Diffs:        allDiffs,
		NavigateTo:   navigateTo,
		Errors:       appErrors,
		StateVersion: state.Version,
	}, nil
}

// processDSLFormActions обрабатывает FormAction из DSL (например, click → apiCall).
// Реальный HTTP-вызов здесь не делается — это задача UseCaseRunner.
func (p *EventProcessor) processDSLFormActions(ctx context.Context, form interface{}, event domain.UIEvent, state *domain.UIState) []domain.StateDiff {
	// FormActions из forms.Form пока обрабатываются через runtime.Bindings.
	// Данный метод — точка расширения.
	return nil
}

// recalcVisibility пересчитывает visibility всех компонентов на основе обновлённого state.
func (p *EventProcessor) recalcVisibility(form interface{}, state *domain.UIState) []domain.StateDiff {
	// Используем page loader для пересчёта (lazy — только если rules есть)
	// Возвращаем diffs только для изменившихся компонентов
	return nil
}

// ProcessResult — итог обработки события.
type ProcessResult struct {
	Diffs        []domain.StateDiff `json:"diffs"`
	NavigateTo   string             `json:"navigateTo,omitempty"`
	Errors       []domain.AppError  `json:"errors,omitempty"`
	StateVersion int64              `json:"stateVersion"`
}
