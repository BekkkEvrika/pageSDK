package page

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/BekkkEvrika/pageSDK/engine"
	"github.com/BekkkEvrika/pageSDK/engine/formengine"
	inputs "github.com/BekkkEvrika/pageSDK/form"
)

// CalculatorPage is a minimal form page demonstrating nested containers
// and runtime updates without changing the form structure.
type CalculatorPage struct {
	*formengine.FormEngine
}

func NewCalculatorPage() engine.Page {
	return &CalculatorPage{
		FormEngine: &formengine.FormEngine{},
	}
}

func (p *CalculatorPage) Init(_ *engine.BuildContext) error {
	p.CreateForm(inputs.Form{
		Containers: &[]inputs.Container{
			{
				Key:       "calculator",
				Direction: "vertical",
				Gap:       16,
				Card:      true,
				Title:     "Простой калькулятор",
				Containers: []inputs.Container{
					{
						Key:         "operands",
						Direction:   "horizontal",
						Gap:         12,
						GridColumns: 2,
						Fields: []inputs.Input{
							{
								Id:           "left",
								Type:         inputs.InputTypeNumber,
								Label:        "Первое число",
								DefaultValue: 0,
							},
							{
								Id:           "right",
								Type:         inputs.InputTypeNumber,
								Label:        "Второе число",
								DefaultValue: 0,
							},
						},
					},
					{
						Key:       "operations",
						Direction: "horizontal",
						Gap:       8,
						Fields: []inputs.Input{
							{Id: "add", Type: inputs.InputTypeButton, Label: "+", Variant: "primary"},
							{Id: "subtract", Type: inputs.InputTypeButton, Label: "−"},
							{Id: "multiply", Type: inputs.InputTypeButton, Label: "×"},
							{Id: "divide", Type: inputs.InputTypeButton, Label: "÷"},
						},
					},
				},
				Fields: []inputs.Input{
					{
						Id:           "result",
						Type:         inputs.InputTypeNumber,
						Label:        "Результат",
						ReadOnly:     true,
						DefaultValue: 0,
					},
				},
			},
		},
	})

	if err := p.bindCalculatorButton("add", calculatorAdd); err != nil {
		return err
	}
	if err := p.bindCalculatorButton("subtract", calculatorSubtract); err != nil {
		return err
	}
	if err := p.bindCalculatorButton("multiply", calculatorMultiply); err != nil {
		return err
	}
	return p.bindCalculatorButton("divide", calculatorDivide)
}

func (p *CalculatorPage) bindCalculatorButton(id string, handler formengine.ClickListener) error {
	button, err := p.GetButtonById(id)
	if err != nil {
		return err
	}
	button.SetOnClick(handler)
	return nil
}

func calculatorAdd(ctx *formengine.RuntimeContext) {
	calculate(ctx, func(left, right float64) (float64, error) {
		return left + right, nil
	})
}

func calculatorSubtract(ctx *formengine.RuntimeContext) {
	calculate(ctx, func(left, right float64) (float64, error) {
		return left - right, nil
	})
}

func calculatorMultiply(ctx *formengine.RuntimeContext) {
	calculate(ctx, func(left, right float64) (float64, error) {
		return left * right, nil
	})
}

func calculatorDivide(ctx *formengine.RuntimeContext) {
	calculate(ctx, func(left, right float64) (float64, error) {
		if right == 0 {
			return 0, errors.New("деление на ноль невозможно")
		}
		return left / right, nil
	})
}

func calculate(ctx *formengine.RuntimeContext, operation func(float64, float64) (float64, error)) {
	left, err := calculatorValue(ctx, "left")
	if err != nil {
		ctx.SetError(err)
		return
	}
	right, err := calculatorValue(ctx, "right")
	if err != nil {
		ctx.SetError(err)
		return
	}

	value, err := operation(left, right)
	if err != nil {
		ctx.SetError(err)
		return
	}

	result, err := ctx.GetNumberById("result")
	if err != nil {
		return
	}
	result.SetValue(value)
}

func calculatorValue(ctx *formengine.RuntimeContext, id string) (float64, error) {
	control, err := ctx.GetNumberById(id)
	if err != nil {
		return 0, err
	}

	value := control.Element().Value
	switch number := value.(type) {
	case float64:
		return number, nil
	case float32:
		return float64(number), nil
	case int:
		return float64(number), nil
	case int64:
		return float64(number), nil
	case json.Number:
		return number.Float64()
	case string:
		parsed, parseErr := strconv.ParseFloat(strings.TrimSpace(number), 64)
		if parseErr != nil {
			return 0, fmt.Errorf("поле %q должно содержать число", id)
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("поле %q должно содержать число", id)
	}
}
