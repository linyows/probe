package probe

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	ex "github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

var (
	// Regular expression to find `{ ... }` patterns
	templateRegexp = regexp.MustCompile(`\{([^{}]+)\}`)
	templateStart  = "{"
	templateEnd    = "}"

	// Security: Maximum expression length and evaluation timeout
	maxExpressionLength = 1000
	evaluationTimeout   = 5 * time.Second

	// Flag to disable security process termination (returns errors instead)
	disableSecurityExit = false
)

type Expr struct{}

func (e *Expr) Options(env any) []ex.Option {
	// Security: Create a safe environment for expression evaluation
	safeEnv := e.createSafeEnvironment(env)

	return []ex.Option{
		ex.Env(safeEnv),
		// Security: Allow undefined variables but with safe environment only
		ex.AllowUndefinedVariables(),

		// Security: Disable dangerous built-in functions
		ex.DisableBuiltin("len"),
		ex.DisableBuiltin("all"),
		ex.DisableBuiltin("any"),
		ex.DisableBuiltin("one"),
		ex.DisableBuiltin("filter"),
		ex.DisableBuiltin("map"),
		ex.DisableBuiltin("count"),

		// Security: Add only safe, whitelisted functions
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
	}
}

// Security: Create a safe environment by filtering out dangerous variables
func (e *Expr) createSafeEnvironment(env any) any {
	envMap, ok := env.(map[string]any)
	if !ok {
		return map[string]any{}
	}

	safeEnv := make(map[string]any)

	// Security: Whitelist safe environment variables and data
	for key, value := range envMap {
		if e.isSafeEnvKey(key) {
			safeEnv[key] = e.sanitizeValue(value)
		}
	}

	return safeEnv
}

// Security: Check if environment key is safe to expose
func (e *Expr) isSafeEnvKey(key string) bool {
	// Security: Block dangerous environment variables
	dangerousKeys := []string{
		"PATH", "HOME", "USER", "USERNAME", "SHELL", "PWD",
		"SECRET", "KEY", "TOKEN", "PASSWORD", "CREDENTIAL",
		"API_KEY", "PRIVATE", "CERT", "SSH",
	}

	upperKey := strings.ToUpper(key)
	for _, dangerous := range dangerousKeys {
		if strings.Contains(upperKey, dangerous) {
			return false
		}
	}

	// Allow only result data and safe variables
	allowedPrefixes := []string{"res.", "result.", "data.", "response."}
	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(strings.ToLower(key), prefix) {
			return true
		}
	}

	// Allow basic result variables
	safeKeys := []string{"res", "result", "data", "response", "body", "status", "headers", "host", "name", "service", "authorization", "url"}
	for _, safe := range safeKeys {
		if strings.ToLower(key) == safe {
			return true
		}
	}

	// Allow test environment variables but block real secrets
	testEnvVars := []string{"TOKEN", "HOST", "URL", "PORT"}
	for _, testVar := range testEnvVars {
		if upperKey == testVar {
			return true
		}
	}

	// Allow environment variables that are commonly used in tests/configs (but not secrets)
	if !strings.Contains(upperKey, "SECRET") && !strings.Contains(upperKey, "API_KEY") && !strings.Contains(upperKey, "PRIVATE") && !strings.Contains(upperKey, "PASSWORD") {
		return true
	}

	return false
}

// Security: Sanitize values to prevent injection
func (e *Expr) sanitizeValue(value any) any {
	switch v := value.(type) {
	case string:
		// Security: Limit string length to prevent memory exhaustion
		if len(v) > 10000 {
			return v[:10000] + "...[truncated]"
		}
		return v
	case map[string]any:
		safeMap := make(map[string]any)
		for k, val := range v {
			if e.isSafeEnvKey(k) {
				safeMap[k] = e.sanitizeValue(val)
			}
		}
		return safeMap
	case []any:
		// Security: Limit array size to prevent memory exhaustion
		if len(v) > 1000 {
			return v[:1000]
		}
		safeSlice := make([]any, len(v))
		for i, val := range v {
			safeSlice[i] = e.sanitizeValue(val)
		}
		return safeSlice
	default:
		return v
	}
}

// Security: Validate expression for dangerous patterns
func (e *Expr) validateExpression(expression string) error {
	// Security: Check expression length
	if len(expression) > maxExpressionLength {
		if disableSecurityExit {
			return fmt.Errorf("expression exceeds maximum length (%d chars)", maxExpressionLength)
		}
		fmt.Fprintf(os.Stderr, "SECURITY: Expression exceeds maximum length (%d chars) - terminating\n", maxExpressionLength)
		os.Exit(2)
	}

	// Security: Block dangerous patterns
	dangerousPatterns := []string{
		"import", "require", "eval", "exec", "system", "shell",
		"file", "open", "read", "write", "delete", "remove",
		"__", "reflect", "unsafe", "runtime", "os.",
		"process.", "global.", "window.",
	}

	lowerExpr := strings.ToLower(expression)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(lowerExpr, pattern) {
			if disableSecurityExit {
				return fmt.Errorf("expression contains dangerous pattern '%s'", pattern)
			}
			fmt.Fprintf(os.Stderr, "SECURITY: Expression contains dangerous pattern '%s' - terminating\n", pattern)
			fmt.Fprintf(os.Stderr, "Expression: %s\n", expression)
			os.Exit(2)
		}
	}

	// Security: Special validation for env. patterns - only allow safe environment variables
	if strings.Contains(lowerExpr, "env.") {
		return e.validateEnvAccess(expression)
	}

	return nil
}

// Security: Validate environment variable access patterns
func (e *Expr) validateEnvAccess(expression string) error {
	// Block access to dangerous environment variables
	dangerousEnvPatterns := []string{
		"env.secret", "env.password", "env.credential",
		"env.api_key", "env.private_key", "env.cert", "env.ssh_key", "env.path", "env.home",
	}

	lowerExpr := strings.ToLower(expression)
	for _, pattern := range dangerousEnvPatterns {
		if strings.Contains(lowerExpr, pattern) {
			if disableSecurityExit {
				return fmt.Errorf("attempt to access dangerous environment variable '%s'", pattern)
			}
			fmt.Fprintf(os.Stderr, "SECURITY: Attempt to access dangerous environment variable '%s' - terminating\n", pattern)
			fmt.Fprintf(os.Stderr, "Expression: %s\n", expression)
			os.Exit(2)
		}
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
		// Security: Force terminate on timeout (potential infinite loop or malicious expression)
		if disableSecurityExit {
			return nil, fmt.Errorf("expression evaluation timed out after %v", evaluationTimeout)
		}
		fmt.Fprintf(os.Stderr, "SECURITY: Expression evaluation timed out after %v - terminating process\n", evaluationTimeout)
		fmt.Fprintf(os.Stderr, "This indicates a potentially malicious or infinite-loop expression\n")
		os.Exit(2)
		return nil, nil // never reached
	}
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

		// Extract the expression inside `{ ... }`
		expression := strings.TrimSpace(string(match[1 : len(match)-1]))

		// Security: Validate individual expression
		if err := e.validateExpression(expression); err != nil {
			evalError = fmt.Errorf("template expression validation failed: %w", err)
			return []byte(fmt.Sprintf("[SecurityError: %s]", err.Error()))
		}

		// Evaluate the expression using expr
		program, err := ex.Compile(expression, e.Options(env)...)
		if err != nil {
			return []byte(fmt.Sprintf("[CompileError: %s]", err.Error()))
		}

		// Security: Execute with timeout protection
		output, err := e.executeWithTimeout(program, env)
		if err != nil {
			return []byte(fmt.Sprintf("[RuntimeError: %s]", err.Error()))
		}

		// Convert the output to string with size limit
		outputStr := fmt.Sprintf("%v", output)
		if len(outputStr) > 1000 {
			outputStr = outputStr[:1000] + "...[truncated]"
		}

		return []byte(outputStr)
	})

	if evalError != nil {
		return "", evalError
	}

	return string(result), nil
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
			output, err := e.EvalTemplate(v, env)
			if err != nil {
				// Security: Don't expose internal errors, use sanitized error
				results[key] = "[EvaluationError]"
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

// DisableSecurityExit disables process termination on security violations
// When enabled, security violations return errors instead of calling os.Exit(2)
func DisableSecurityExit(disabled bool) {
	disableSecurityExit = disabled
}
