package page

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/BekkkEvrika/pageSDK/access"
	"github.com/BekkkEvrika/pageSDK/engine"
	"github.com/BekkkEvrika/pageSDK/engine/formengine"
	inputs "github.com/BekkkEvrika/pageSDK/form"
)

// CalculatorPage demonstrates a classic calculator with one display.
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
				Gap:       12,
				Card:      true,
				Title:     "Калькулятор",
				Fields: []inputs.Input{
					{
						Id:               "result",
						Type:             inputs.InputTypeText,
						Label:            "Результат",
						ReadOnly:         true,
						DefaultValue:     "0",
						ElementCode:      "calculator.result",
						AccessGroupCode:  CalculatorUsage.Code,
						NoAccessBehavior: string(access.NoAccessReadonly),
					},
				},
				Containers: []inputs.Container{
					calculatorButtonRow("row1",
						calculatorButton("clear", "C", "destructive"),
						calculatorButton("divide", "÷", "secondary"),
						calculatorButton("multiply", "×", "secondary"),
						calculatorButton("backspace", "⌫", "secondary"),
					),
					calculatorButtonRow("row2",
						calculatorButton("digit7", "7", ""),
						calculatorButton("digit8", "8", ""),
						calculatorButton("digit9", "9", ""),
						calculatorButton("subtract", "−", "secondary"),
					),
					calculatorButtonRow("row3",
						calculatorButton("digit4", "4", ""),
						calculatorButton("digit5", "5", ""),
						calculatorButton("digit6", "6", ""),
						calculatorButton("add", "+", "secondary"),
					),
					calculatorButtonRow("row4",
						calculatorButton("digit1", "1", ""),
						calculatorButton("digit2", "2", ""),
						calculatorButton("digit3", "3", ""),
						calculatorButton("equals", "=", "primary"),
					),
					calculatorButtonRow("row5",
						calculatorButton("digit0", "0", ""),
						calculatorButton("decimal", ".", ""),
					),
				},
			},
		},
	})

	for digit := 0; digit <= 9; digit++ {
		id := "digit" + strconv.Itoa(digit)
		value := strconv.Itoa(digit)
		if err := p.bindCalculatorButton(id, func(ctx *formengine.RuntimeContext) {
			appendCalculatorDigit(ctx, value)
		}); err != nil {
			return err
		}
	}

	handlers := []struct {
		id      string
		handler formengine.ClickListener
	}{
		{id: "add", handler: calculatorOperator("+")},
		{id: "subtract", handler: calculatorOperator("-")},
		{id: "multiply", handler: calculatorOperator("*")},
		{id: "divide", handler: calculatorOperator("/")},
		{id: "decimal", handler: appendCalculatorDecimal},
		{id: "equals", handler: calculateResult},
		{id: "clear", handler: clearCalculator},
		{id: "backspace", handler: backspaceCalculator},
	}
	for _, item := range handlers {
		if err := p.bindCalculatorButton(item.id, item.handler); err != nil {
			return err
		}
	}

	return nil
}

func calculatorButtonRow(id string, buttons ...inputs.Input) inputs.Container {
	return inputs.Container{
		Key:         id,
		Direction:   "horizontal",
		Gap:         8,
		GridColumns: 4,
		Fields:      buttons,
	}
}

func calculatorButton(id, label, variant string) inputs.Input {
	return inputs.Input{
		Id:               id,
		Type:             inputs.InputTypeButton,
		Label:            label,
		Variant:          variant,
		ElementCode:      "calculator.button." + id,
		AccessGroupCode:  CalculatorUsage.Code,
		NoAccessBehavior: string(access.NoAccessHidden),
	}
}

func (p *CalculatorPage) bindCalculatorButton(id string, handler formengine.ClickListener) error {
	button, err := p.GetButtonById(id)
	if err != nil {
		return err
	}
	button.SetOnClick(handler)
	return nil
}

func appendCalculatorDigit(ctx *formengine.RuntimeContext, digit string) {
	updateCalculatorDisplay(ctx, func(expression string) (string, error) {
		if expression == "0" {
			return digit, nil
		}
		return expression + digit, nil
	})
}

func appendCalculatorDecimal(ctx *formengine.RuntimeContext) {
	updateCalculatorDisplay(ctx, func(expression string) (string, error) {
		parts := strings.Fields(expression)
		if len(parts)%2 == 0 {
			return expression + "0.", nil
		}
		current := parts[len(parts)-1]
		if strings.Contains(current, ".") {
			return expression, nil
		}
		return expression + ".", nil
	})
}

func calculatorOperator(operator string) formengine.ClickListener {
	return func(ctx *formengine.RuntimeContext) {
		updateCalculatorDisplay(ctx, func(expression string) (string, error) {
			parts := strings.Fields(expression)
			if len(parts)%2 == 0 {
				parts[len(parts)-1] = operator
				return strings.Join(parts, " "), nil
			}
			return expression + " " + operator + " ", nil
		})
	}
}

func calculateResult(ctx *formengine.RuntimeContext) {
	updateCalculatorDisplay(ctx, evaluateCalculatorExpression)
}

func clearCalculator(ctx *formengine.RuntimeContext) {
	setCalculatorDisplay(ctx, "0")
}

func backspaceCalculator(ctx *formengine.RuntimeContext) {
	updateCalculatorDisplay(ctx, func(expression string) (string, error) {
		expression = strings.TrimSpace(expression)
		if expression == "" || expression == "0" {
			return "0", nil
		}
		expression = strings.TrimSpace(expression[:len(expression)-1])
		if expression == "" {
			return "0", nil
		}
		return expression, nil
	})
}

func updateCalculatorDisplay(
	ctx *formengine.RuntimeContext,
	update func(string) (string, error),
) {
	display, err := ctx.GetTextById("result")
	if err != nil {
		return
	}

	expression := fmt.Sprint(display.Element().Value)
	if expression == "" || expression == "<nil>" {
		expression = "0"
	}
	value, err := update(expression)
	if err != nil {
		ctx.SetError(err)
		return
	}
	display.SetValue(value)
}

func setCalculatorDisplay(ctx *formengine.RuntimeContext, value string) {
	display, err := ctx.GetTextById("result")
	if err != nil {
		return
	}
	display.SetValue(value)
}

// evaluateCalculatorExpression evaluates operations from left to right,
// matching the behaviour of a simple pocket calculator.
func evaluateCalculatorExpression(expression string) (string, error) {
	parts := strings.Fields(expression)
	if len(parts) == 0 {
		return "0", nil
	}
	if len(parts)%2 == 0 {
		return "", errors.New("выражение не закончено")
	}

	result, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return "", errors.New("некорректное число")
	}

	for index := 1; index < len(parts); index += 2 {
		right, parseErr := strconv.ParseFloat(parts[index+1], 64)
		if parseErr != nil {
			return "", errors.New("некорректное число")
		}

		switch parts[index] {
		case "+":
			result += right
		case "-":
			result -= right
		case "*":
			result *= right
		case "/":
			if right == 0 {
				return "", errors.New("деление на ноль невозможно")
			}
			result /= right
		default:
			return "", errors.New("неизвестная операция")
		}
	}

	return strconv.FormatFloat(result, 'f', -1, 64), nil
}
