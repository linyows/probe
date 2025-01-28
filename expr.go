package probe

import (
	"fmt"
	"regexp"
	"strings"

	ex "github.com/expr-lang/expr"
)

var (
	// Regular expression to find `{ ... }` patterns
	templateRegexp = regexp.MustCompile(`\{([^{}]+)\}`)
	templateStart  = "{"
	templateEnd    = "}"
)

type Expr struct{}

func (e *Expr) Options(env any) []ex.Option {
	return []ex.Option{
		ex.Env(env),
		ex.AllowUndefinedVariables(),
		ex.Function(
			"match_json",
			func(params ...any) (any, error) {
				src := params[0].(map[string]any)
				target := params[1].(map[string]any)
				return MatchJSON(src, target), nil
			},
		),
		ex.Function(
			"diff_json",
			func(params ...any) (any, error) {
				src := params[0].(map[string]any)
				target := params[1].(map[string]any)
				return DiffJSON(src, target), nil
			},
		),
	}
}

func (e *Expr) EvalOrEvalTemplate(input string, env any) (string, error) {
	if strings.Contains(input, templateStart) && strings.Contains(input, templateEnd) {
		return e.EvalTemplate(input, env)
	}
	output, err := e.Eval(input, env)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%v", output), nil
}

func (e *Expr) Eval(input string, env any) (any, error) {
	program, err := ex.Compile(input, e.Options(env)...)
	if err != nil {
		return false, err
	}
	return ex.Run(program, env)
}

func (e *Expr) EvalTemplate(input string, env any) (string, error) {
	re := templateRegexp

	// Replace matches with evaluated results
	result := re.ReplaceAllFunc([]byte(input), func(match []byte) []byte {
		// Extract the expression inside `{ ... }`
		expression := strings.TrimSpace(string(match[1 : len(match)-1]))

		// Evaluate the expression using expr
		program, err := ex.Compile(expression, e.Options(env)...)
		if err != nil {
			return []byte(fmt.Sprintf("[Error: %s]", err.Error()))
		}

		output, err := ex.Run(program, env)
		if err != nil {
			return []byte(fmt.Sprintf("[Error: %s]", err.Error()))
		}

		// Convert the output to string
		return []byte(fmt.Sprintf("%v", output))
	})

	return string(result), nil
}

func (e *Expr) EvalTemplateMap(input map[string]any, env any) map[string]any {
	results := make(map[string]any)

	for key, val := range input {
		switch v := val.(type) {
		case string:
			output, err := e.EvalTemplate(v, env)
			if err != nil {
				results[key] = v
				continue
			}
			results[key] = output

		case map[string]any:
			results[key] = e.EvalTemplateMap(v, env)

		default:
			results[key] = v
		}
	}

	return results
}
