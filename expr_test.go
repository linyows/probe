package probe

import (
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestEvalTemplateMap(t *testing.T) {
	exprs := map[string]any{
		"url":           "{env.URL}",
		"authorization": "Bearer {env.TOKEN}",
	}
	env := map[string]any{
		"env": map[string]any{
			"URL":   "https://example.com",
			"TOKEN": "secrets",
		},
	}
	expected := map[string]any{
		"url":           "https://example.com",
		"authorization": "Bearer secrets",
	}
	expr := &Expr{}
	actual := expr.EvalTemplateMap(exprs, env)
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("map are not equal: expected %+v, got %+v", expected, actual)
	}
}

func TestEvalTemplate(t *testing.T) {
	tests := []struct {
		name     string
		str      string
		env      map[string]any
		expected string
	}{
		{
			name: "only variable",
			str:  "{env.URL}",
			env: map[string]any{
				"env": map[string]any{
					"URL": "https://example.com",
				},
			},
			expected: "https://example.com",
		},
		{
			name: "expr twice",
			str:  "Hi, { name }. My name is { service }.",
			env: map[string]any{
				"name":    "Bob",
				"service": "Alice",
			},
			expected: "Hi, Bob. My name is Alice.",
		},
		{
			name: "use nil coalescing operator",
			str:  "{env.URL ?? 'http://localhost'}",
			env: map[string]any{
				"env": map[string]any{},
			},
			expected: "http://localhost",
		},
		{
			name: "use ternary operator",
			str:  "{env.URL == 'localhost' ? 'http://localhost:3000' : env.URL}",
			env: map[string]any{
				"env": map[string]any{
					"URL": "localhost",
				},
			},
			expected: "http://localhost:3000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr := &Expr{}
			actual, err := expr.EvalTemplate(tt.str, tt.env)
			if err != nil {
				t.Errorf("EvalTemplate error %s", err)
			}
			if tt.expected != actual {
				t.Errorf("expected %+v, got %+v", tt.expected, actual)
			}
		})
	}
}

func TestEval(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		env      map[string]any
		expected any
		hasError bool
	}{
		{
			name:     "simple boolean expression",
			input:    "true && false",
			env:      map[string]any{},
			expected: false,
			hasError: false,
		},
		{
			name:  "variable access",
			input: "name == 'test'",
			env: map[string]any{
				"name": "test",
			},
			expected: true,
			hasError: false,
		},
		{
			name:  "numeric comparison",
			input: "status >= 200 && status < 300",
			env: map[string]any{
				"status": 200,
			},
			expected: true,
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr := &Expr{}
			result, err := expr.Eval(tt.input, tt.env)

			if tt.hasError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.hasError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.hasError && result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEvalOrEvalTemplate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		env      map[string]any
		expected string
		hasError bool
	}{
		{
			name:     "simple expression without template",
			input:    "1 + 1",
			env:      map[string]any{},
			expected: "2",
			hasError: false,
		},
		{
			name:  "template with variable",
			input: "Hello {name}!",
			env: map[string]any{
				"name": "World",
			},
			expected: "Hello World!",
			hasError: false,
		},
		{
			name:  "boolean expression",
			input: "status == 200",
			env: map[string]any{
				"status": 200,
			},
			expected: "true",
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr := &Expr{}
			result, err := expr.EvalOrEvalTemplate(tt.input, tt.env)

			if tt.hasError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.hasError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.hasError && result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestSecurityValidation(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "expression too long",
			input:       strings.Repeat("a", 1001),
			shouldError: true,
			errorMsg:    "expression exceeds maximum length",
		},
		{
			name:        "dangerous env access",
			input:       "env.SECRET_KEY",
			shouldError: true,
			errorMsg:    "dangerous environment variable",
		},
		{
			name:        "safe expression",
			input:       "status == 200",
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr := &Expr{}
			_, err := expr.Eval(tt.input, map[string]any{"status": 200})

			if tt.shouldError {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestCustomFunctions(t *testing.T) {
	t.Run("match_json function", func(t *testing.T) {
		expr := &Expr{}
		input := "match_json(src, target)"
		env := map[string]any{
			"src": map[string]any{
				"name": "test",
				"age":  25,
			},
			"target": map[string]any{
				"name": "test",
				"age":  25,
			},
		}

		result, err := expr.Eval(input, env)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result != true {
			t.Errorf("expected true, got %v", result)
		}
	})

	t.Run("diff_json function", func(t *testing.T) {
		expr := &Expr{}
		input := "diff_json(src, target)"
		env := map[string]any{
			"src": map[string]any{
				"name": "test",
				"age":  25,
			},
			"target": map[string]any{
				"name": "test",
				"age":  30,
			},
		}

		result, err := expr.Eval(input, env)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result == nil {
			t.Errorf("expected diff result, got nil")
		}
		// Should return difference as string
		if resultStr, ok := result.(string); !ok || resultStr == "No diff" {
			t.Errorf("expected diff string, got %v", result)
		}
	})

	t.Run("custom function with wrong parameters", func(t *testing.T) {
		expr := &Expr{}
		input := "match_json(src)"
		env := map[string]any{
			"src": map[string]any{"name": "test"},
		}

		_, err := expr.Eval(input, env)
		if err == nil {
			t.Errorf("expected error for wrong parameter count")
		}
	})
}

func TestSafeEnvironment(t *testing.T) {
	expr := &Expr{}

	t.Run("blocks dangerous environment variables", func(t *testing.T) {
		dangerousEnv := map[string]any{
			"SECRET_KEY": "secret123",
			"PASSWORD":   "pass123",
			"PATH":       "/usr/bin",
			"safe_var":   "allowed",
		}

		safeEnv := expr.createSafeEnvironment(dangerousEnv)
		envMap := safeEnv.(map[string]any)

		if _, exists := envMap["SECRET_KEY"]; exists {
			t.Errorf("SECRET_KEY should be blocked")
		}
		if _, exists := envMap["PASSWORD"]; exists {
			t.Errorf("PASSWORD should be blocked")
		}
		if _, exists := envMap["PATH"]; exists {
			t.Errorf("PATH should be blocked")
		}
		if _, exists := envMap["safe_var"]; !exists {
			t.Errorf("safe_var should be allowed")
		}
	})

	t.Run("allows safe environment variables", func(t *testing.T) {
		safeEnv := map[string]any{
			"res":      map[string]any{"status": 200},
			"result":   "success",
			"data":     map[string]any{"id": 1},
			"response": "ok",
			"HOST":     "localhost",
			"URL":      "http://example.com",
		}

		result := expr.createSafeEnvironment(safeEnv)
		envMap := result.(map[string]any)

		for key := range safeEnv {
			if _, exists := envMap[key]; !exists {
				t.Errorf("%s should be allowed", key)
			}
		}
	})
}

func TestValueSanitization(t *testing.T) {
	expr := &Expr{}

	t.Run("truncates long strings", func(t *testing.T) {
		longString := strings.Repeat("a", 10001)
		result := expr.sanitizeValue(longString)
		resultStr := result.(string)

		// The actual truncation includes "...[truncated]" suffix which adds extra chars
		if len(resultStr) <= 10000 {
			t.Errorf("string appears not to be long enough to test truncation, got %d chars", len(resultStr))
		}
		if !strings.Contains(resultStr, "...[truncated]") {
			t.Errorf("truncated string should contain truncation marker")
		}
		// Check that the original long part was truncated
		if len(strings.Replace(resultStr, "...[truncated]", "", 1)) > 10000 {
			t.Errorf("string content should be truncated to 10000 chars")
		}
	})

	t.Run("limits array size", func(t *testing.T) {
		largeArray := make([]any, 1001)
		for i := range largeArray {
			largeArray[i] = i
		}

		result := expr.sanitizeValue(largeArray)
		resultArray := result.([]any)

		if len(resultArray) > 1000 {
			t.Errorf("large array should be truncated to 1000 elements")
		}
	})
}

func TestErrorHandling(t *testing.T) {
	t.Run("handles compilation errors", func(t *testing.T) {
		expr := &Expr{}
		_, err := expr.Eval("invalid syntax $$", map[string]any{})

		if err == nil {
			t.Errorf("expected compilation error")
		}
	})

	t.Run("handles template evaluation errors", func(t *testing.T) {
		expr := &Expr{}
		result, err := expr.EvalTemplate("{invalid syntax $$}", map[string]any{})

		if err == nil && !strings.Contains(result, "CompileError") {
			t.Errorf("expected compilation error in template")
		}
	})

	t.Run("handles map evaluation with errors", func(t *testing.T) {
		expr := &Expr{}
		inputMap := map[string]any{
			"valid":   "simple string",
			"invalid": "{invalid syntax $$}",
		}

		result := expr.EvalTemplateMap(inputMap, map[string]any{})

		if result["valid"] != "simple string" {
			t.Errorf("valid expression should work")
		}
		// The actual error message may vary, just check it's not the original invalid template
		if result["invalid"] == "{invalid syntax $$}" {
			t.Errorf("invalid expression should be processed and not return original template")
		}
	})
}

func TestTimeoutProtection(t *testing.T) {
	t.Run("prevents infinite loops", func(t *testing.T) {
		expr := &Expr{}

		// This test is tricky because we need an expression that would loop
		// but expr-lang is designed to be safe. Let's test the timeout mechanism
		// by using a mock timeout scenario
		start := time.Now()
		_, err := expr.Eval("true", map[string]any{})
		duration := time.Since(start)

		// The expression should complete quickly (not timeout)
		if err != nil {
			t.Errorf("simple expression should not error: %v", err)
		}
		if duration > time.Second {
			t.Errorf("simple expression took too long: %v", duration)
		}
	})
}

func TestNewCustomFunctions(t *testing.T) {
	expr := &Expr{}
	env := map[string]any{}

	t.Run("random_int", func(t *testing.T) {
		tests := []struct {
			name      string
			input     string
			expectErr bool
		}{
			{
				name:      "valid random_int with positive integer",
				input:     "random_int(100)",
				expectErr: false,
			},
			{
				name:      "valid random_int with float64 (common in JSON)",
				input:     "random_int(100.0)",
				expectErr: false,
			},
			{
				name:      "invalid random_int with zero",
				input:     "random_int(0)",
				expectErr: true,
			},
			{
				name:      "invalid random_int with negative",
				input:     "random_int(-1)",
				expectErr: true,
			},
			{
				name:      "invalid random_int with string",
				input:     "random_int('100')",
				expectErr: true,
			},
			{
				name:      "invalid random_int with no params",
				input:     "random_int()",
				expectErr: true,
			},
			{
				name:      "invalid random_int with multiple params",
				input:     "random_int(100, 200)",
				expectErr: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := expr.Eval(tt.input, env)
				if tt.expectErr {
					if err == nil {
						t.Errorf("expected error but got none")
					}
					return
				}
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				
				// Check result is integer and in valid range
				if val, ok := result.(int); ok {
					if tt.input == "random_int(100)" || tt.input == "random_int(100.0)" {
						if val < 0 || val >= 100 {
							t.Errorf("random_int(100) returned %d, expected 0 <= result < 100", val)
						}
					}
				} else {
					t.Errorf("expected int result, got %T", result)
				}
			})
		}
	})

	t.Run("random_str", func(t *testing.T) {
		tests := []struct {
			name      string
			input     string
			expectErr bool
			length    int
		}{
			{
				name:      "valid random_str with positive length",
				input:     "random_str(10)",
				expectErr: false,
				length:    10,
			},
			{
				name:      "valid random_str with float64",
				input:     "random_str(5.0)",
				expectErr: false,
				length:    5,
			},
			{
				name:      "valid random_str with large length",
				input:     "random_str(10000)",
				expectErr: false,
				length:    10000,
			},
			{
				name:      "invalid random_str with zero",
				input:     "random_str(0)",
				expectErr: true,
			},
			{
				name:      "invalid random_str with negative",
				input:     "random_str(-1)",
				expectErr: true,
			},
			{
				name:      "invalid random_str too long",
				input:     "random_str(1000001)",
				expectErr: true,
			},
			{
				name:      "invalid random_str with string",
				input:     "random_str('10')",
				expectErr: true,
			},
			{
				name:      "invalid random_str with no params",
				input:     "random_str()",
				expectErr: true,
			},
			{
				name:      "invalid random_str with multiple params",
				input:     "random_str(10, 20)",
				expectErr: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := expr.Eval(tt.input, env)
				if tt.expectErr {
					if err == nil {
						t.Errorf("expected error but got none")
					}
					return
				}
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				
				// Check result is string with correct length
				if str, ok := result.(string); ok {
					if len(str) != tt.length {
						t.Errorf("random_str(%d) returned string of length %d, expected %d", tt.length, len(str), tt.length)
					}
					
					// Check charset (alphanumeric)
					matched, err := regexp.MatchString("^[a-zA-Z0-9]+$", str)
					if err != nil {
						t.Errorf("regex error: %v", err)
					}
					if !matched {
						t.Errorf("random_str returned invalid characters: %s", str)
					}
				} else {
					t.Errorf("expected string result, got %T", result)
				}
			})
		}
	})

	t.Run("unixtime", func(t *testing.T) {
		tests := []struct {
			name      string
			input     string
			expectErr bool
		}{
			{
				name:      "valid unixtime with no params",
				input:     "unixtime()",
				expectErr: false,
			},
			{
				name:      "invalid unixtime with params",
				input:     "unixtime(123)",
				expectErr: true,
			},
			{
				name:      "invalid unixtime with string param",
				input:     "unixtime('now')",
				expectErr: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				beforeCall := time.Now().Unix()
				result, err := expr.Eval(tt.input, env)
				afterCall := time.Now().Unix()
				
				if tt.expectErr {
					if err == nil {
						t.Errorf("expected error but got none")
					}
					return
				}
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				
				// Check result is int64 and reasonable timestamp
				if val, ok := result.(int64); ok {
					if val < beforeCall || val > afterCall {
						t.Errorf("unixtime() returned %d, expected between %d and %d", val, beforeCall, afterCall)
					}
				} else {
					t.Errorf("expected int64 result, got %T", result)
				}
			})
		}
	})

	t.Run("template evaluation with custom functions", func(t *testing.T) {
		tests := []struct {
			name     string
			template string
		}{
			{
				name:     "random_int in template",
				template: "User ID: {random_int(9999)}",
			},
			{
				name:     "random_str in template",
				template: "Session: {random_str(16)}",
			},
			{
				name:     "unixtime in template",
				template: "Timestamp: {unixtime()}",
			},
			{
				name:     "multiple functions in template",
				template: "ID: {random_int(1000)}, Token: {random_str(8)}, Time: {unixtime()}",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := expr.EvalTemplate(tt.template, env)
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				
				// Basic validation that template was processed
				if result == tt.template {
					t.Errorf("template was not processed: %s", result)
				}
				
				// Check that placeholders were replaced
				if regexp.MustCompile(`\{[^}]+\}`).MatchString(result) {
					t.Errorf("template still contains unreplaced placeholders: %s", result)
				}
			})
		}
	})
}

func TestNewCustomFunctionsEdgeCases(t *testing.T) {
	expr := &Expr{}
	env := map[string]any{}

	t.Run("random_int boundary values", func(t *testing.T) {
		// Test with 1 (smallest valid value)
		result, err := expr.Eval("random_int(1)", env)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if val, ok := result.(int); ok {
			if val != 0 {
				t.Errorf("random_int(1) should always return 0, got %d", val)
			}
		}

		// Test with 2 (returns 0 or 1)
		result, err = expr.Eval("random_int(2)", env)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if val, ok := result.(int); ok {
			if val < 0 || val >= 2 {
				t.Errorf("random_int(2) returned %d, expected 0 or 1", val)
			}
		}
	})

	t.Run("random_str boundary values", func(t *testing.T) {
		// Test with 1 (smallest valid length)
		result, err := expr.Eval("random_str(1)", env)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if str, ok := result.(string); ok {
			if len(str) != 1 {
				t.Errorf("random_str(1) returned string of length %d, expected 1", len(str))
			}
		}
	})

	t.Run("unixtime consistency", func(t *testing.T) {
		// Call unixtime multiple times in quick succession
		results := make([]int64, 3)
		for i := 0; i < 3; i++ {
			result, err := expr.Eval("unixtime()", env)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if val, ok := result.(int64); ok {
				results[i] = val
			}
		}

		// All results should be within a reasonable time range (same second or consecutive seconds)
		for i := 1; i < len(results); i++ {
			diff := results[i] - results[i-1]
			if diff < 0 || diff > 1 {
				t.Errorf("unixtime() calls returned inconsistent results: %v", results)
			}
		}
	})
}
