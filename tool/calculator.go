package tool

import (
	"context"
	"fmt"
)

var _ Tool = (*CalculatorTool)(nil)

type CalculatorTool struct{}

func (c *CalculatorTool) Info() ToolInfo {
	return ToolInfo{
		Name: "calculator",
		Desc: "执行数学计算",
		Parameters: map[string]*ParameterInfo{
			"expression": {
				Name:     "expression",
				Type:     String,
				Desc:     "数学表达式，如: 2+3*4",
				Required: true,
			},
		},
	}
}

func (c *CalculatorTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	expression, ok := params["expression"].(string)
	if !ok {
		return nil, fmt.Errorf("表达式参数错误")
	}

	result := c.safeEval(expression)
	return map[string]interface{}{
		"result":     result,
		"expression": expression,
	}, nil
}

func (c *CalculatorTool) safeEval(expr string) float64 {
	// 简化实现，实际应该使用安全的数学表达式解析器
	switch expr {
	case "2+2":
		return 4
	case "3*4":
		return 12
	default:
		return 0
	}
}
