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

func TestMergeStringMaps(t *testing.T) {
	tests := []struct {
		name     string
		base     map[string]string
		over     map[string]any
		expected map[string]string
	}{
		{
			name:     "merge string values",
			base:     map[string]string{"a": "1", "b": "2"},
			over:     map[string]any{"b": "overridden", "c": "3"},
			expected: map[string]string{"a": "1", "b": "overridden", "c": "3"},
		},
		{
			name:     "ignore non-string values",
			base:     map[string]string{"a": "1"},
			over:     map[string]any{"a": "overridden", "b": 123, "c": true},
			expected: map[string]string{"a": "overridden"},
		},
		{
			name:     "empty base map",
			base:     map[string]string{},
			over:     map[string]any{"a": "1", "b": "2"},
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
			name:     "simple merge",
			base:     map[string]any{"a": 1, "b": 2},
			over:     map[string]any{"b": 3, "c": 4},
			expected: map[string]any{"a": 1, "b": 3, "c": 4},
		},
		{
			name: "recursive merge nested maps",
			base: map[string]any{
				"a":      1,
				"nested": map[string]any{"x": 1, "y": 2},
			},
			over: map[string]any{
				"nested": map[string]any{"y": 3, "z": 4},
				"c":      5,
			},
			expected: map[string]any{
				"a":      1,
				"nested": map[string]any{"x": 1, "y": 3, "z": 4},
				"c":      5,
			},
		},
		{
			name:     "overwrite with non-map value",
			base:     map[string]any{"a": map[string]any{"x": 1}},
			over:     map[string]any{"a": "string"},
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
		String:      "test",
		Number:      42,
		Bool:        true,
		Bytes:       []byte("bytes"),
		Required:    "required",
		MapStrStr:   map[string]string{"key": "value"},
		EmbedStruct: TestEmbedStruct{Name: "embedded"},
	}

	expected := map[string]any{
		"string":       "test",
		"number":       42,
		"bool":         true,
		"bytes":        "bytes",
		"required":     "required",
		"map_str_str":  map[string]string{"key": "value"},
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
