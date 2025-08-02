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

func TestMapToStructByTags(t *testing.T) {
	got := TestStruct{
		String: "hello, world!",
		MapStrStr: map[string]string{
			"foo":   "bar",
			"hello": "world",
		},
	}

	params := map[string]any{
		"string":   "s-t-r-i-n-g",
		"number":   123,
		"bool":     false,
		"bytes":    "b-y-t-e-s",
		"required": "required!",
		"map_str_str": map[string]any{
			"foo": "f-o-o",
			"bar": "b-a-r",
			"baz": "b-a-z",
		},
		"embed_struct": map[string]any{
			"name": "probe",
		},
	}

	expects := TestStruct{
		String:   "s-t-r-i-n-g",
		Number:   123,
		Bool:     false,
		Bytes:    []byte("b-y-t-e-s"),
		Required: "required!",
		MapStrStr: map[string]string{
			"foo":   "f-o-o",
			"bar":   "b-a-r",
			"baz":   "b-a-z",
			"hello": "world",
		},
		EmbedStruct: TestEmbedStruct{
			Name: "probe",
		},
	}

	if err := MapToStructByTags(params, &got); err != nil {
		t.Errorf("MapToStructByTags error %s", err)
	}

	if !reflect.DeepEqual(got, expects) {
		t.Errorf("\nExpected:\n%#v\nGot:\n%#v", expects, got)
	}
}

func TestMapToStructByTags_Required(t *testing.T) {
	got := TestStruct{}
	params := map[string]any{"string": "yo"}
	err := MapToStructByTags(params, &got)

	if err.Error() != "required field 'required' is missing" {
		t.Errorf("MapToStructByTags error is wrong: %s", err)
	}
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

func TestMergeStringMaps(t *testing.T) {
	tests := []struct {
		name     string
		base     map[string]string
		over     map[string]any
		expected map[string]string
	}{
		{
			name: "merge string values",
			base: map[string]string{"a": "1", "b": "2"},
			over: map[string]any{"b": "overridden", "c": "3"},
			expected: map[string]string{"a": "1", "b": "overridden", "c": "3"},
		},
		{
			name: "ignore non-string values",
			base: map[string]string{"a": "1"},
			over: map[string]any{"a": "overridden", "b": 123, "c": true},
			expected: map[string]string{"a": "overridden"},
		},
		{
			name: "empty base map",
			base: map[string]string{},
			over: map[string]any{"a": "1", "b": "2"},
			expected: map[string]string{"a": "1", "b": "2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MergeStringMaps(tt.base, tt.over)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("MergeStringMaps() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestMergeMaps(t *testing.T) {
	tests := []struct {
		name     string
		base     map[string]any
		over     map[string]any
		expected map[string]any
	}{
		{
			name: "simple merge",
			base: map[string]any{"a": 1, "b": 2},
			over: map[string]any{"b": 3, "c": 4},
			expected: map[string]any{"a": 1, "b": 3, "c": 4},
		},
		{
			name: "recursive merge nested maps",
			base: map[string]any{
				"a": 1,
				"nested": map[string]any{"x": 1, "y": 2},
			},
			over: map[string]any{
				"nested": map[string]any{"y": 3, "z": 4},
				"c": 5,
			},
			expected: map[string]any{
				"a": 1,
				"nested": map[string]any{"x": 1, "y": 3, "z": 4},
				"c": 5,
			},
		},
		{
			name: "overwrite with non-map value",
			base: map[string]any{"a": map[string]any{"x": 1}},
			over: map[string]any{"a": "string"},
			expected: map[string]any{"a": "string"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MergeMaps(tt.base, tt.over)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("MergeMaps() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestStructToMapByTags(t *testing.T) {
	input := TestStruct{
		String:   "test",
		Number:   42,
		Bool:     true,
		Bytes:    []byte("bytes"),
		Required: "required",
		MapStrStr: map[string]string{"key": "value"},
		EmbedStruct: TestEmbedStruct{Name: "embedded"},
	}

	expected := map[string]any{
		"string":   "test",
		"number":   42,
		"bool":     true,
		"bytes":    "bytes",
		"required": "required",
		"map_str_str": map[string]string{"key": "value"},
		"embed_struct": map[string]any{"name": "embedded"},
	}

	result, err := StructToMapByTags(input)
	if err != nil {
		t.Errorf("StructToMapByTags() error = %v", err)
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("StructToMapByTags() = %v, want %v", result, expected)
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

func TestAnyToString(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
		ok       bool
	}{
		{"string", "hello", "hello", true},
		{"bool true", true, "true", true},
		{"bool false", false, "false", true},
		{"int", 42, "42", true},
		{"int64", int64(42), "42", true},
		{"float64", 3.14, "3.14", true},
		{"[]byte", []byte("bytes"), "bytes", true},
		{"nil", nil, "nil", true},
		{"unsupported", []int{1, 2, 3}, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := AnyToString(tt.input)
			if ok != tt.ok {
				t.Errorf("AnyToString() ok = %v, want %v", ok, tt.ok)
			}
			if result != tt.expected {
				t.Errorf("AnyToString() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestTitleCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		char     string
		expected string
	}{
		{"hyphen separated", "content-type", "-", "Content-Type"},
		{"underscore separated", "user_name", "_", "User_Name"},
		{"single word", "hello", "-", "Hello"},
		{"empty string", "", "-", ""},
		{"multiple separators", "a-b-c-d", "-", "A-B-C-D"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TitleCase(tt.input, tt.char)
			if result != tt.expected {
				t.Errorf("TitleCase() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestStrmapToAnymap(t *testing.T) {
	input := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}
	expected := map[string]any{
		"key1": "value1",
		"key2": "value2",
	}

	result := StrmapToAnymap(input)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("StrmapToAnymap() = %v, want %v", result, expected)
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
