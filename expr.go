package probe

import (
	"encoding/base64"
	"fmt"
	"math/rand/v2"
	"regexp"
	"strconv"
	"strings"
	"time"

	ex "github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

var (
	// Regular expression to find `{{ ... }}` patterns
	templateRegexp = regexp.MustCompile(`\{\{([^{}]+)\}\}`)
	templateStart  = "{{"
	templateEnd    = "}}"

	// Security: Maximum expression length and evaluation timeout
	maxExpressionLength = 1000000
	evaluationTimeout   = 5 * time.Second

	// Security: Maximum string length to prevent memory exhaustion
	maxStringLength = 1000000
)

type Expr struct{}

// Options builds the expr options used to compile every workflow expression.
//
// Two guards that used to live here have been removed because neither did
// anything. ex.DisableBuiltin had no effect on all / any / one / filter / map /
// count: expr's parser recognises those as predicate builtins before it looks
// at the disabled set (parser.go, the `predicates` table), so the calls were
// dead and the functions have always been reachable. A per-key blocklist that
// filtered the environment was equally inert, because the filtered copy only
// ever reached the type checker while ex.Run receives the caller's env, and
// enforcing it would have broken the documented way of reading credentials
// into `vars`.
//
// What still bounds an expression is the length check in validateExpression,
// the evaluation timeout in executeWithTimeout, and the argument limits on the
// functions registered below.
func (e *Expr) Options(env any) []ex.Option {
	return []ex.Option{
		ex.Env(env),
		// Workflow expressions routinely reference names that only exist at
		// run time, such as environment variables read into `vars`.
		ex.AllowUndefinedVariables(),

		// Functions probe adds on top of expr's own builtins.
		ex.Function(
			"match_json",
			func(params ...any) (any, error) {
				if len(params) != 2 {
					return false, fmt.Errorf("match_json requires exactly 2 parameters")
				}
				src, ok1 := params[0].(map[string]any)
				target, ok2 := params[1].(map[string]any)
				if !ok1 || !ok2 {
					return false, fmt.Errorf("match_json parameters must be objects")
				}
				return MatchJSON(src, target), nil
			},
		),
		ex.Function(
			"diff_json",
			func(params ...any) (any, error) {
				if len(params) != 2 {
					return nil, fmt.Errorf("diff_json requires exactly 2 parameters")
				}
				src, ok1 := params[0].(map[string]any)
				target, ok2 := params[1].(map[string]any)
				if !ok1 || !ok2 {
					return nil, fmt.Errorf("diff_json parameters must be objects")
				}
				return DiffJSON(src, target), nil
			},
		),
		ex.Function(
			"random_int",
			func(params ...any) (any, error) {
				if len(params) != 1 {
					return nil, fmt.Errorf("random_int requires exactly 1 parameter")
				}
				n, ok := params[0].(int)
				if !ok {
					// Try to convert float64 to int (common in JSON/expr)
					if f, ok := params[0].(float64); ok {
						n = int(f)
					} else {
						return nil, fmt.Errorf("random_int parameter must be an integer")
					}
				}
				if n <= 0 {
					return nil, fmt.Errorf("random_int parameter must be positive")
				}
				return rand.IntN(n), nil
			},
		),
		ex.Function(
			"random_str",
			func(params ...any) (any, error) {
				if len(params) != 1 {
					return nil, fmt.Errorf("random_str requires exactly 1 parameter")
				}
				length, ok := params[0].(int)
				if !ok {
					// Try to convert float64 to int (common in JSON/expr)
					if f, ok := params[0].(float64); ok {
						length = int(f)
					} else {
						return nil, fmt.Errorf("random_str parameter must be an integer")
					}
				}
				if length <= 0 {
					return nil, fmt.Errorf("random_str parameter must be positive")
				}
				if length > 1000000 {
					return nil, fmt.Errorf("random_str parameter must be <= 1000000")
				}

				const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
				b := make([]byte, length)
				for i := range b {
					b[i] = charset[rand.IntN(len(charset))]
				}

				return string(b), nil
			},
		),
		ex.Function(
			"unixtime",
			func(params ...any) (any, error) {
				if len(params) != 0 {
					return nil, fmt.Errorf("unixtime takes no parameters")
				}
				return time.Now().Unix(), nil
			},
		),
		ex.Function(
			"parse_float",
			func(params ...any) (any, error) {
				if len(params) != 1 {
					return nil, fmt.Errorf("parse_float requires exactly 1 parameter")
				}
				s, ok := params[0].(string)
				if !ok {
					return nil, fmt.Errorf("parse_float parameter must be a string")
				}
				f, err := strconv.ParseFloat(s, 64)
				if err != nil {
					return nil, fmt.Errorf("parse_float error: %w", err)
				}
				return f, nil
			},
		),
		ex.Function(
			"parse_int",
			func(params ...any) (any, error) {
				if len(params) != 1 {
					return nil, fmt.Errorf("parse_int requires exactly 1 parameter")
				}

				switch v := params[0].(type) {
				case string:
					i, err := strconv.ParseInt(v, 10, 64)
					if err != nil {
						return nil, fmt.Errorf("parse_int error: %w", err)
					}
					return i, nil
				case int:
					return int64(v), nil
				case int64:
					return v, nil
				case uint64:
					return int64(v), nil
				case float64:
					return int64(v), nil
				default:
					return nil, fmt.Errorf("parse_int parameter must be a string or number, got %T", v)
				}
			},
		),
		ex.Function(
			"encode_base64",
			func(params ...any) (any, error) {
				if len(params) != 1 {
					return nil, fmt.Errorf("encode_base64 requires exactly 1 parameter")
				}
				s, ok := params[0].(string)
				if !ok {
					return nil, fmt.Errorf("encode_base64 parameter must be a string")
				}
				if len(s) > maxStringLength {
					return nil, fmt.Errorf("encode_base64 parameter exceeds maximum length (%d chars)", maxStringLength)
				}
				return base64.StdEncoding.EncodeToString([]byte(s)), nil
			},
		),
		ex.Function(
			"parse_json",
			func(params ...any) (any, error) {
				if len(params) != 1 {
					return nil, fmt.Errorf("parse_json requires exactly 1 parameter")
				}
				s, ok := params[0].(string)
				if !ok {
					return nil, fmt.Errorf("parse_json parameter must be a string")
				}
				if len(s) > maxStringLength {
					return nil, fmt.Errorf("parse_json parameter exceeds maximum length (%d chars)", maxStringLength)
				}
				return ParseJSON(s)
			},
		),
		ex.Function(
			"decode_base64",
			func(params ...any) (any, error) {
				if len(params) != 1 {
					return nil, fmt.Errorf("decode_base64 requires exactly 1 parameter")
				}
				s, ok := params[0].(string)
				if !ok {
					return nil, fmt.Errorf("decode_base64 parameter must be a string")
				}
				if len(s) > maxStringLength {
					return nil, fmt.Errorf("decode_base64 parameter exceeds maximum length (%d chars)", maxStringLength)
				}
				decoded, err := base64.StdEncoding.DecodeString(s)
				if err != nil {
					return nil, fmt.Errorf("decode_base64 error: %w", err)
				}
				return string(decoded), nil
			},
		),
	}
}

// validateExpression bounds an expression before it is compiled.
//
// This used to also reject any expression whose text contained "env.secret",
// "env.path" and similar. That guarded a namespace expressions never had -
// the evaluation context exposes vars, res, req, rt, status, outputs and
// repeat_index, never env - while rejecting ordinary strings that happen to
// contain one of the patterns, such as the URL "https://env.pathfinder.example.com".
func (e *Expr) validateExpression(expression string) error {
	// Security: Check expression length
	if len(expression) > maxExpressionLength {
		return fmt.Errorf("SECURITY: expression exceeds maximum length (%d chars)", maxExpressionLength)
	}

	return nil
}

func (e *Expr) EvalOrEvalTemplate(input string, env any) (string, error) {
	// Security: Validate input expression
	if err := e.validateExpression(input); err != nil {
		return "", fmt.Errorf("expression validation failed: %w", err)
	}

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
	// Security: Validate expression before compilation
	if err := e.validateExpression(input); err != nil {
		return nil, fmt.Errorf("expression validation failed: %w", err)
	}

	program, err := ex.Compile(input, e.Options(env)...)
	if err != nil {
		return false, err
	}

	// Security: Execute with timeout to prevent infinite loops
	return e.executeWithTimeout(program, env)
}

// Security: Execute expression with timeout protection
func (e *Expr) executeWithTimeout(program *vm.Program, env any) (any, error) {
	type result struct {
		output any
		err    error
	}

	resultCh := make(chan result, 1)
	done := make(chan bool, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				select {
				case resultCh <- result{nil, fmt.Errorf("expression execution panicked: %v", r)}:
				default:
				}
				done <- true
			}
		}()

		output, err := ex.Run(program, env)
		select {
		case resultCh <- result{output, err}:
		default:
		}
		done <- true
	}()

	select {
	case res := <-resultCh:
		return res.output, res.err
	case <-time.After(evaluationTimeout):
		return nil, fmt.Errorf("SECURITY: expression evaluation timed out after %v", evaluationTimeout)
	}
}

// isWholeStringTemplate checks if the input string contains only a single template expression
func isWholeStringTemplate(input string) bool {
	trimmed := strings.TrimSpace(input)
	if !strings.HasPrefix(trimmed, templateStart) || !strings.HasSuffix(trimmed, templateEnd) {
		return false
	}

	// Count template markers to ensure there's exactly one pair
	startCount := strings.Count(trimmed, templateStart)
	endCount := strings.Count(trimmed, templateEnd)

	return startCount == 1 && endCount == 1
}

// extractTemplateExpression extracts the expression from a whole string template
func extractTemplateExpression(input string) string {
	trimmed := strings.TrimSpace(input)
	if !strings.HasPrefix(trimmed, templateStart) || !strings.HasSuffix(trimmed, templateEnd) {
		return ""
	}

	// Remove template markers and trim whitespace
	expression := trimmed[len(templateStart) : len(trimmed)-len(templateEnd)]
	return strings.TrimSpace(expression)
}

func (e *Expr) EvalTemplate(input string, env any) (string, error) {
	// Security: Validate template input
	if err := e.validateExpression(input); err != nil {
		return "", fmt.Errorf("template validation failed: %w", err)
	}

	re := templateRegexp
	var evalError error

	// Replace matches with evaluated results
	result := re.ReplaceAllFunc([]byte(input), func(match []byte) []byte {
		// Security: Check if we've already encountered an error
		if evalError != nil {
			return match
		}

		// Extract the expression inside `{{ ... }}` using submatch
		submatch := re.FindStringSubmatch(string(match))
		if len(submatch) < 2 {
			evalError = fmt.Errorf("invalid template expression: %s", string(match))
			return []byte("[TemplateError: invalid expression]")
		}
		expression := strings.TrimSpace(submatch[1])

		// Security: Validate individual expression
		if err := e.validateExpression(expression); err != nil {
			evalError = fmt.Errorf("template expression validation failed: %w", err)
			return fmt.Appendf(nil, "[SecurityError: %s]", err.Error())
		}

		// Evaluate the expression using expr
		program, err := ex.Compile(expression, e.Options(env)...)
		if err != nil {
			return fmt.Appendf(nil, "[CompileError: %s]", err.Error())
		}

		// Security: Execute with timeout protection
		output, err := e.executeWithTimeout(program, env)
		if err != nil {
			return fmt.Appendf(nil, "[RuntimeError: %s]", err.Error())
		}

		// Convert the output to string with size limit
		outputStr := fmt.Sprintf("%v", output)
		if len(outputStr) > maxStringLength {
			outputStr = outputStr[:maxStringLength] + GetTruncationMessage()
		}

		return []byte(outputStr)
	})

	if evalError != nil {
		return "", evalError
	}

	return string(result), nil
}

func (e *Expr) EvalTemplateWithTypePreservation(input string, env any) (any, error) {
	// Security: Validate template input
	if err := e.validateExpression(input); err != nil {
		return "", fmt.Errorf("template validation failed: %w", err)
	}

	// Check if this is a whole string template for type preservation
	if isWholeStringTemplate(input) {
		expression := extractTemplateExpression(input)
		if expression == "" {
			return "", fmt.Errorf("failed to extract expression from template: %s", input)
		}

		// Use Eval directly to preserve type
		return e.Eval(expression, env)
	}

	// For partial templates, fall back to string processing
	return e.EvalTemplate(input, env)
}

func (e *Expr) EvalTemplateMap(input map[string]any, env any) map[string]any {
	results := make(map[string]any)

	for key, val := range input {
		// Security: Limit the number of processed keys to prevent DoS
		if len(results) > 1000 {
			results["_truncated"] = "Map processing truncated due to size limits"
			break
		}

		switch v := val.(type) {
		case string:
			output, err := e.EvalTemplateWithTypePreservation(v, env)
			if err != nil {
				// Security: Don't expose internal errors, use sanitized error
				results[key] = "[EvaluationError]"
				continue
			}
			results[key] = output

		case map[string]any:
			results[key] = e.EvalTemplateMap(v, env)

		case []any:
			results[key] = e.evalTemplateArray(v, env)

		default:
			results[key] = v
		}
	}

	return results
}

// evalTemplateArray evaluates templates in array elements
func (e *Expr) evalTemplateArray(input []any, env any) []any {
	results := make([]any, len(input))

	for i, val := range input {
		// Security: Limit the number of processed elements to prevent DoS
		if i > 1000 {
			results = append(results, "_truncated: Array processing truncated due to size limits")
			break
		}

		switch v := val.(type) {
		case string:
			output, err := e.EvalTemplateWithTypePreservation(v, env)
			if err != nil {
				// Security: Don't expose internal errors, use sanitized error
				results[i] = "[EvaluationError]"
				continue
			}
			results[i] = output

		case map[string]any:
			results[i] = e.EvalTemplateMap(v, env)

		case []any:
			results[i] = e.evalTemplateArray(v, env)

		default:
			results[i] = v
		}
	}

	return results
}
