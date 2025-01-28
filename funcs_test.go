package probe

import (
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
