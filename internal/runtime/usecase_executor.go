package runtime

import (
	"context"
	"fmt"

	"github.com/behzod/pageSDK/internal/domain"
	"github.com/behzod/pageSDK/internal/ports"
)

// CallUseCaseExecutor вызывает бизнес-логику через UseCaseRunner.
type CallUseCaseExecutor struct {
	runner ports.UseCaseRunner
}

func NewCallUseCaseExecutor(runner ports.UseCaseRunner) *CallUseCaseExecutor {
	return &CallUseCaseExecutor{runner: runner}
}

func (e *CallUseCaseExecutor) Type() domain.ActionType { return domain.ActionCallUseCase }

func (e *CallUseCaseExecutor) Execute(ctx context.Context, action domain.Action, state *domain.UIState, _ domain.UIEvent) (*domain.ActionResult, error) {
	if action.UseCaseName == "" {
		return nil, fmt.Errorf("useCaseName is required for callUseCase action")
	}
	result, err := e.runner.Run(ctx, action.UseCaseName, action.Params, state)
	if err != nil {
		return &domain.ActionResult{
			Errors: []domain.AppError{{Code: "USE_CASE_ERROR", Message: err.Error()}},
		}, nil
	}

	// Результат use case мержим в state.Values
	var diffs []domain.StateDiff
	for k, v := range result {
		state.Values[k] = v
		diffs = append(diffs, domain.StateDiff{
			ComponentID: k,
			Changes:     map[string]interface{}{"value": v},
		})
	}
	return &domain.ActionResult{Diffs: diffs}, nil
}
