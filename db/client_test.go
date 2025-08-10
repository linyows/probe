package db

import (
	"testing"
)

func TestParseParams(t *testing.T) {
	tests := []struct {
		name    string
		input   map[string]string
		wantErr bool
	}{
		{
			name: "valid mysql dsn",
			input: map[string]string{
				"dsn":   "mysql://user:pass@localhost:3306/testdb",
				"query": "SELECT * FROM users",
			},
			wantErr: false,
		},
		{
			name: "valid postgres dsn",
			input: map[string]string{
				"dsn":   "postgres://user:pass@localhost:5432/testdb",
				"query": "SELECT * FROM users",
			},
			wantErr: false,
		},
		{
			name: "valid sqlite dsn",
			input: map[string]string{
				"dsn":   "test.db",
				"query": "SELECT * FROM users",
			},
			wantErr: false,
		},
		{
			name: "missing dsn",
			input: map[string]string{
				"query": "SELECT * FROM users",
			},
			wantErr: true,
		},
		{
			name: "missing query",
			input: map[string]string{
				"dsn": "mysql://user:pass@localhost:3306/testdb",
			},
			wantErr: true,
		},
		{
			name: "invalid dsn format",
			input: map[string]string{
				"dsn":   "invalid://dsn",
				"query": "SELECT * FROM users",
			},
			wantErr: true,
		},
		{
			name: "with timeout",
			input: map[string]string{
				"dsn":     "mysql://user:pass@localhost:3306/testdb",
				"query":   "SELECT * FROM users",
				"timeout": "30s",
			},
			wantErr: false,
		},
		{
			name: "with params",
			input: map[string]string{
				"dsn":    "mysql://user:pass@localhost:3306/testdb",
				"query":  "SELECT * FROM users WHERE id = ?",
				"param1": "123",
				"param2": "test",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _, _, err := ParseRequest(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && req == nil {
				t.Error("ParseRequest() returned nil request for valid input")
			}
		})
	}
}
