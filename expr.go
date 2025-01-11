package probe

import (
	"fmt"
	"strings"

	ex "github.com/expr-lang/expr"
)

const (
	defaultExprStart = "{"
	defaultExprEnd   = "}"
)

type Expr struct {
	start string
	end   string
}

func NewExpr() *Expr {
	return &Expr{
		start: defaultExprStart,
		end:   defaultExprEnd,
	}
}

func (e *Expr) EvalTemplate(exprs map[string]any, env any) map[string]any {
	results := make(map[string]any)

	for key, val := range exprs {
		switch v := val.(type) {
		case string:
			output, err := e.EvalTemplateStr(v, env)
			if err != nil {
				results[key] = v
				continue
			}
			results[key] = output

		case map[string]any:
			results[key] = e.EvalTemplate(v, env)

		default:
			results[key] = v
		}
	}

	return results
}

func (e *Expr) EvalTemplateStr(s string, env any) (string, error) {
	start := strings.Index(s, e.start)
	end := strings.Index(s, e.end)
	if start == -1 || end == -1 || start > end {
		return s, nil
	}

	input := s[start+1 : end]

	output, err := EvalExpr(input, env)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s%v%s", s[:start], output, s[end+1:]), nil
}

func EvalExpr(input string, env any) (any, error) {
	return ex.Eval(input, env)
}
