package probe

import (
	"fmt"
	"strings"

	"github.com/expr-lang/expr"
)

const (
	exprStart = "{"
	exprEnd   = "}"
)

func EvaluateExprs(exprs map[string]any, env any) map[string]any {
	results := make(map[string]any)

	for key, val := range exprs {
		switch v := val.(type) {
		case string:
			output, err := evaluateExpr(v, env)
			if err != nil {
				results[key] = v
				continue
			}
			results[key] = output

		case map[string]any:
			results[key] = EvaluateExprs(v, env)

		default:
			results[key] = v
		}
	}

	return results
}

func evaluateExpr(ex string, env any) (string, error) {
	start := strings.Index(ex, exprStart)
	end := strings.Index(ex, exprEnd)
	if start == -1 || end == -1 || start > end {
		return ex, nil
	}

	expression := ex[start+1 : end]

	output, err := expr.Eval(expression, env)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s%v%s", ex[:start], output, ex[end+1:]), nil
}
