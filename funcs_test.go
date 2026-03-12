package probe

import (
	"reflect"
	"testing"
)

func TestMatchJSON(t *testing.T) {
	tests := []struct {
		name   string
		src    map[string]any
		target map[string]any
		want   bool
	}{
		{
			name: "Exact match with simple fields",
			src: map[string]any{
				"id":   123,
				"name": "Alice",
			},
			target: map[string]any{
				"id":   123,
				"name": "Alice",
			},
			want: true,
		},
		{
			name: "Extra field in src",
			src: map[string]any{
				"id":   123,
				"name": "Alice",
				"age":  30,
			},
			target: map[string]any{
				"id":   123,
				"name": "Alice",
			},
			want: false,
		},
		{
			name: "Missing field in src",
			src: map[string]any{
				"id": 123,
			},
			target: map[string]any{
				"id":   123,
				"name": "Alice",
			},
			want: false,
		},
		{
			name: "Field value mismatch",
			src: map[string]any{
				"id":   123,
				"name": "Bob",
			},
			target: map[string]any{
				"id":   123,
				"name": "Alice",
			},
			want: false,
		},
		{
			name: "Nested map exact match",
			src: map[string]any{
				"id":   123,
				"name": "Alice",
				"meta": map[string]any{
					"role": "admin",
				},
			},
			target: map[string]any{
				"id":   123,
				"name": "Alice",
				"meta": map[string]any{
					"role": "admin",
				},
			},
			want: true,
		},
		{
			name: "Nested map mismatch",
			src: map[string]any{
				"id":   123,
				"name": "Alice",
				"meta": map[string]any{
					"role": "user",
				},
			},
			target: map[string]any{
				"id":   123,
				"name": "Alice",
				"meta": map[string]any{
					"role": "admin",
				},
			},
			want: false,
		},
		{
			name: "Regex in bool src matches",
			src: map[string]any{
				"bot": true,
			},
			target: map[string]any{
				"bot": "/^(true|false)$/",
			},
			want: true,
		},
		{
			name: "Different number type as float64 and uint64",
			src: map[string]any{
				"id": float64(123),
			},
			target: map[string]any{
				"id": uint64(123),
			},
			want: true,
		},
		{
			name: "Regex in float64 src matches",
			src: map[string]any{
				"id": float64(12345),
			},
			target: map[string]any{
				"id": "/^\\d{5}$/",
			},
			want: true,
		},
		{
			name: "Array exact match",
			src: map[string]any{
				"id":    123,
				"roles": []any{"admin", "editor"},
			},
			target: map[string]any{
				"id":    123,
				"roles": []any{"admin", "editor"},
			},
			want: true,
		},
		{
			name: "Array mismatch",
			src: map[string]any{
				"id":    123,
				"roles": []any{"admin", "viewer"},
			},
			target: map[string]any{
				"id":    123,
				"roles": []any{"admin", "editor"},
			},
			want: false,
		},
		{
			name: "Regex in target does not match",
			src: map[string]any{
				"id":   123,
				"name": "Alice",
			},
			target: map[string]any{
				"id":   123,
				"name": "/^Bob$/", // Expecting "Bob", but "Alice" is provided
			},
			want: false,
		},
		{
			name: "Regex in target matches",
			src: map[string]any{
				"id":   123,
				"name": "Alice",
			},
			target: map[string]any{
				"id":   123,
				"name": "/^\\w{5}$/", // Matches any name match 5 ascii
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchJSON(tt.src, tt.target)
			if got != tt.want {
				t.Errorf("deepMatch() = %v, want %v (src: %v, target: %v)", got, tt.want, tt.src, tt.target)
			}
		})
	}
}

func TestParseJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected any
		wantErr  bool
	}{
		{
			name:  "simple object",
			input: `{"id": 123, "name": "test"}`,
			expected: map[string]any{
				"id":   float64(123),
				"name": "test",
			},
			wantErr: false,
		},
		{
			name:  "nested object",
			input: `{"data": {"id": 456, "tags": ["a", "b"]}}`,
			expected: map[string]any{
				"data": map[string]any{
					"id":   float64(456),
					"tags": []any{"a", "b"},
				},
			},
			wantErr: false,
		},
		{
			name:     "array",
			input:    `[1, 2, 3]`,
			expected: []any{float64(1), float64(2), float64(3)},
			wantErr:  false,
		},
		{
			name:     "array of objects",
			input:    `[{"id": 1}, {"id": 2}]`,
			expected: []any{map[string]any{"id": float64(1)}, map[string]any{"id": float64(2)}},
			wantErr:  false,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid json",
			input:   "not json",
			wantErr: true,
		},
		{
			name:    "plain string",
			input:   `"hello"`,
			wantErr: true,
		},
		{
			name:    "plain number",
			input:   "123",
			wantErr: true,
		},
		{
			name:  "object with whitespace",
			input: "  \n  {\"key\": \"value\"}  \n  ",
			expected: map[string]any{
				"key": "value",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseJSON(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseJSON() expected error but got none, result: %v", got)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseJSON() unexpected error: %v", err)
				return
			}

			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("ParseJSON() = %v, want %v", got, tt.expected)
			}
		})
	}
}
