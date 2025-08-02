package probe

import (
	"reflect"
	"testing"
)

type TestStruct struct {
	String      string            `map:"string"`
	Number      int               `map:"number"`
	Bool        bool              `map:"bool"`
	Bytes       []byte            `map:"bytes"`
	Required    string            `map:"required" validate:"required"`
	MapStrStr   map[string]string `map:"map_str_str"`
	EmbedStruct TestEmbedStruct   `map:"embed_struct"`
}

type TestEmbedStruct struct {
	Name string `map:"name"`
}


func TestFlattenInterface(t *testing.T) {
	tests := []struct {
		name    string
		expects map[string]string
		data    map[string]any
	}{
		{
			name: "two consecutive underscores are used to express nesting",
			expects: map[string]string{
				"map_str_str__foo": "f-o-o",
				"map_str_str__bar": "b-a-r",
				"string":           "s-t-r-i-n-g",
			},
			data: map[string]any{
				"map_str_str": map[string]any{
					"foo": "f-o-o",
					"bar": "b-a-r",
				},
				"string": "s-t-r-i-n-g",
			},
		},
		{
			name: "it is an empty string when the value is nil",
			expects: map[string]string{
				"map_str_str__foo": "f-o-o",
				"map_str_str__bar": "",
				"string":           "s-t-r-i-n-g",
			},
			data: map[string]any{
				"map_str_str": map[string]any{
					"foo": "f-o-o",
					"bar": nil,
				},
				"string": "s-t-r-i-n-g",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FlattenInterface(tt.data)

			if !reflect.DeepEqual(got, tt.expects) {
				t.Errorf("\nExpected:\n%#v\nGot:\n%#v", tt.expects, got)
			}
		})
	}
}

func TestUnflattenInterface(t *testing.T) {
	tests := []struct {
		name    string
		expects map[string]any
		data    map[string]string
	}{
		{
			name: "nest maps if there are two consecutive underscore keys",
			expects: map[string]any{
				"map_str_str": map[string]any{
					"foo": "f-o-o",
					"bar": "b-a-r",
				},
				"string": "s-t-r-i-n-g",
			},
			data: map[string]string{
				"map_str_str__foo": "f-o-o",
				"map_str_str__bar": "b-a-r",
				"string":           "s-t-r-i-n-g",
			},
		},
		{
			name: "nesting it is the same when the field is an empty string",
			expects: map[string]any{
				"map_str_str": map[string]any{
					"foo": "f-o-o",
					"bar": "",
				},
				"string": "s-t-r-i-n-g",
			},
			data: map[string]string{
				"map_str_str__foo": "f-o-o",
				"map_str_str__bar": "",
				"string":           "s-t-r-i-n-g",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UnflattenInterface(tt.data)

			if !reflect.DeepEqual(got, tt.expects) {
				t.Errorf("\nExpected:\n%#v\nGot:\n%#v", tt.expects, got)
			}
		})
	}
}




func TestShouldConvertToArray(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected bool
	}{
		{
			name:     "empty map",
			input:    map[string]any{},
			expected: false,
		},
		{
			name:     "sequential numeric keys from 0",
			input:    map[string]any{"0": "a", "1": "b", "2": "c"},
			expected: true,
		},
		{
			name:     "non-sequential numeric keys",
			input:    map[string]any{"0": "a", "2": "b", "3": "c"},
			expected: false,
		},
		{
			name:     "non-numeric keys",
			input:    map[string]any{"a": "a", "b": "b"},
			expected: false,
		},
		{
			name:     "mixed keys",
			input:    map[string]any{"0": "a", "name": "b"},
			expected: false,
		},
		{
			name:     "single element",
			input:    map[string]any{"0": "single"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldConvertToArray(tt.input)
			if result != tt.expected {
				t.Errorf("shouldConvertToArray() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsNumericSequence(t *testing.T) {
	tests := []struct {
		name     string
		keys     []string
		expected bool
	}{
		{
			name:     "empty slice",
			keys:     []string{},
			expected: false,
		},
		{
			name:     "sequential from 0",
			keys:     []string{"0", "1", "2"},
			expected: true,
		},
		{
			name:     "non-sequential",
			keys:     []string{"0", "2", "3"},
			expected: false,
		},
		{
			name:     "non-numeric",
			keys:     []string{"a", "b", "c"},
			expected: false,
		},
		{
			name:     "single element",
			keys:     []string{"0"},
			expected: true,
		},
		{
			name:     "unordered but sequential",
			keys:     []string{"2", "0", "1"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNumericSequence(tt.keys)
			if result != tt.expected {
				t.Errorf("isNumericSequence() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConvertMapToArrayWithNumericConversion(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected []any
	}{
		{
			name:     "simple array conversion",
			input:    map[string]any{"0": "1", "1": "2", "2": "3"},
			expected: []any{1, 2, 3},
		},
		{
			name: "nested arrays and maps",
			input: map[string]any{
				"0": map[string]any{"name": "item1", "count": "5"},
				"1": map[string]any{"0": "nested1", "1": "nested2"},
			},
			expected: []any{
				map[string]any{"name": "item1", "count": 5},
				[]any{"nested1", "nested2"},
			},
		},
		{
			name:     "mixed data types",
			input:    map[string]any{"0": "10", "1": "3.14", "2": "text"},
			expected: []any{10, 3.14, "text"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertMapToArrayWithNumericConversion(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("convertMapToArrayWithNumericConversion() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestUnflattenInterface_ArrayConversion(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]string
		expected map[string]any
	}{
		{
			name: "root level array",
			input: map[string]string{
				"0__name": "item1",
				"1__name": "item2",
			},
			expected: map[string]any{
				"__array_root": []any{
					map[string]any{"name": "item1"},
					map[string]any{"name": "item2"},
				},
			},
		},
		{
			name: "nested array in map",
			input: map[string]string{
				"items__0__name": "nested1",
				"items__1__name": "nested2",
				"title":          "test",
			},
			expected: map[string]any{
				"items": []any{
					map[string]any{"name": "nested1"},
					map[string]any{"name": "nested2"},
				},
				"title": "test",
			},
		},
		{
			name: "numeric conversion",
			input: map[string]string{
				"count": "42",
				"price": "19.99",
				"name":  "product",
			},
			expected: map[string]any{
				"count": 42,
				"price": 19.99,
				"name":  "product",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UnflattenInterface(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("UnflattenInterface() = %v, want %v", result, tt.expected)
			}
		})
	}
}




func TestIsJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid object", `{"key": "value"}`, true},
		{"valid array", `["item1", "item2"]`, true},
		{"invalid json", `not json`, false},
		{"empty string", "", false},
		{"single char", "a", false},
		{"object with spaces", ` {"key": "value"} `, true},
		{"object-like string", `{key: value}`, true}, // isJSON only checks { } brackets, not valid JSON syntax
		{"non-json string", `hello world`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isJSON(tt.input)
			if result != tt.expected {
				t.Errorf("isJSON() = %v, want %v", result, tt.expected)
			}
		})
	}
}
