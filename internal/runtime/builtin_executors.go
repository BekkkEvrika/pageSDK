package runtime

import (
	"context"

	"github.com/behzod/pageSDK/internal/domain"
)

// ---- UpdateState ----

// UpdateStateExecutor обновляет произвольные поля состояния.
type UpdateStateExecutor struct{}

func (e *UpdateStateExecutor) Type() domain.ActionType { return domain.ActionUpdateState }

func (e *UpdateStateExecutor) Execute(_ context.Context, action domain.Action, state *domain.UIState, _ domain.UIEvent) (*domain.ActionResult, error) {
	var diffs []domain.StateDiff
	for k, v := range action.Params {
		state.Values[k] = v
		diffs = append(diffs, domain.StateDiff{
			ComponentID: k,
			Changes:     map[string]interface{}{"value": v},
		})
	}
	return &domain.ActionResult{Diffs: diffs}, nil
}

// ---- SetValue ----

type SetValueExecutor struct{}

func (e *SetValueExecutor) Type() domain.ActionType { return domain.ActionSetValue }

func (e *SetValueExecutor) Execute(_ context.Context, action domain.Action, state *domain.UIState, _ domain.UIEvent) (*domain.ActionResult, error) {
	id := action.TargetID
	val := action.Params["value"]
	state.Values[id] = val
	return &domain.ActionResult{
		Diffs: []domain.StateDiff{{ComponentID: id, Changes: map[string]interface{}{"value": val}}},
	}, nil
}

// ---- SetVisible ----

type SetVisibleExecutor struct{}

func (e *SetVisibleExecutor) Type() domain.ActionType { return domain.ActionSetVisible }

func (e *SetVisibleExecutor) Execute(_ context.Context, action domain.Action, state *domain.UIState, _ domain.UIEvent) (*domain.ActionResult, error) {
	id := action.TargetID
	visible := true
	if v, ok := action.Params["visible"]; ok {
		if b, ok2 := v.(bool); ok2 {
			visible = b
		}
	}
	state.Visibility[id] = visible
	return &domain.ActionResult{
		Diffs: []domain.StateDiff{{ComponentID: id, Changes: map[string]interface{}{"visible": visible}}},
	}, nil
}

// ---- SetDisabled ----

type SetDisabledExecutor struct{}

func (e *SetDisabledExecutor) Type() domain.ActionType { return domain.ActionSetDisabled }

func (e *SetDisabledExecutor) Execute(_ context.Context, action domain.Action, state *domain.UIState, _ domain.UIEvent) (*domain.ActionResult, error) {
	id := action.TargetID
	disabled := true
	if v, ok := action.Params["disabled"]; ok {
		if b, ok2 := v.(bool); ok2 {
			disabled = b
		}
	}
	state.Disabled[id] = disabled
	return &domain.ActionResult{
		Diffs: []domain.StateDiff{{ComponentID: id, Changes: map[string]interface{}{"disabled": disabled}}},
	}, nil
}

// ---- ClearField ----

type ClearFieldExecutor struct{}

func (e *ClearFieldExecutor) Type() domain.ActionType { return domain.ActionClearField }

func (e *ClearFieldExecutor) Execute(_ context.Context, action domain.Action, state *domain.UIState, _ domain.UIEvent) (*domain.ActionResult, error) {
	id := action.TargetID
	delete(state.Values, id)
	delete(state.Errors, id)
	return &domain.ActionResult{
		Diffs: []domain.StateDiff{{ComponentID: id, Changes: map[string]interface{}{"value": nil, "errors": nil}}},
	}, nil
}

// ---- Navigate ----

type NavigateExecutor struct{}

func (e *NavigateExecutor) Type() domain.ActionType { return domain.ActionNavigate }

func (e *NavigateExecutor) Execute(_ context.Context, action domain.Action, _ *domain.UIState, _ domain.UIEvent) (*domain.ActionResult, error) {
	return &domain.ActionResult{NavigateTo: action.NavigateTo}, nil
}
