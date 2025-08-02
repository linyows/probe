package probe

import (
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
