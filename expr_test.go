package probe

import (
	"reflect"
	"testing"
)

func TestNewExpr(t *testing.T) {
	expected := &Expr{
		start: "{",
		end:   "}",
	}
	actual := NewExpr()
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Structs are not equal: expected %+v, got %+v", expected, actual)
	}
}

func TestEvalTemplate(t *testing.T) {
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
	actual := NewExpr().EvalTemplate(exprs, env)
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("map are not equal: expected %+v, got %+v", expected, actual)
	}
}

func TestEvalTemplateStr(t *testing.T) {
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
			expr := NewExpr()
			actual, err := expr.EvalTemplateStr(tt.str, tt.env)
			if err != nil {
				t.Errorf("EvalTemplateStr error %s", err)
			}
			if tt.expected != actual {
				t.Errorf("expected %+v, got %+v", tt.expected, actual)
			}
		})
	}
}
