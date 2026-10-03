package probe

import (
	"reflect"
	"testing"
)

func TestFindTemplates(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"no template", "plain text", nil},
		{"one", "a {{ x }} b", []string{"{{ x }}"}},
		{"two", "{{a}}-{{b}}", []string{"{{a}}", "{{b}}"}},
		{"map literal", "{{ {'a': {'b': 1}}['a']['b'] }}", []string{"{{ {'a': {'b': 1}}['a']['b'] }}"}},
		{"closing braces in a double-quoted string", `{{ "}}" + x }} tail`, []string{`{{ "}}" + x }}`}},
		{"closing braces in a single-quoted string", `{{ '}}' }}`, []string{`{{ '}}' }}`}},
		{"closing braces in a raw string", "{{ `}}` }}", []string{"{{ `}}` }}"}},
		{"escaped quote in a string", `{{ "\"}}" }}`, []string{`{{ "\"}}" }}`}},
		{"opening braces in a string", `{{ "{{" }}`, []string{`{{ "{{" }}`}},
		{"triple braces keep the outer ones as text", "{{{x}}}", []string{"{{x}}"}},
		{"unterminated", "{{ x", nil},
		{"unterminated after a template", "{{ x }} {{ y", []string{"{{ x }}"}},
		{"unterminated string", `{{ "}} }}`, nil},
		{"single braces", "{x}", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []string
			for _, span := range findTemplates(tt.input) {
				got = append(got, tt.input[span.start:span.end])
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("findTemplates(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestEvalTemplateWithBraces(t *testing.T) {
	env := map[string]any{
		"vars": map[string]any{"name": "probe", "key": "id"},
	}
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"map literal in a sentence", "value: {{ {'a': {'b': 1}}['a']['b'] }}!", "value: 1!"},
		{"JSON with nested objects in a string", `{{ parse_json('{"' + vars.key + '": {"name": "' + vars.name + '"}}')[vars.key].name }}`, "probe"},
		{"closing braces in a string", `[{{ "}}" + vars.name }}]`, "[}}probe]"},
		{"triple braces", "{{{vars.name}}}", "{probe}"},
		{"unterminated stays as text", "{{ vars.name", "{{ vars.name"},
		{"text after an unterminated template", "{{ vars.name }} and {{ vars", "probe and {{ vars"},
	}

	e := &Expr{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := e.EvalTemplate(tt.input, env)
			if err != nil {
				t.Fatalf("EvalTemplate(%q) error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("EvalTemplate(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestEvalTemplateWithTypePreservationMapLiteral(t *testing.T) {
	e := &Expr{}
	got, err := e.EvalTemplateWithTypePreservation("{{ {'a': {'b': 1}} }}", nil)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"a": map[string]any{"b": 1}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}

	// Two templates are not one whole-string template, even though the
	// string starts with "{{" and ends with "}}".
	got, err = e.EvalTemplateWithTypePreservation("{{ 1 }}{{ 2 }}", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != "12" {
		t.Errorf("got %#v, want %q", got, "12")
	}
}
