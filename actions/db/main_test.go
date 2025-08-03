package db

import (
	"io"
	"testing"

	"github.com/hashicorp/go-hclog"
)

func TestParseDSN(t *testing.T) {
	tests := []struct {
		name        string
		dsn         string
		wantDriver  string
		wantDSN     string
		wantErr     bool
	}{
		{
			name:        "MySQL DSN",
			dsn:         "mysql://user:pass@localhost:3306/testdb",
			wantDriver:  "mysql",
			wantDSN:     "user:pass@tcp(localhost:3306)/testdb",
			wantErr:     false,
		},
		{
			name:        "MySQL DSN with query params",
			dsn:         "mysql://user:pass@localhost:3306/testdb?charset=utf8",
			wantDriver:  "mysql",
			wantDSN:     "user:pass@tcp(localhost:3306)/testdb?charset=utf8",
			wantErr:     false,
		},
		{
			name:        "PostgreSQL DSN",
			dsn:         "postgres://user:pass@localhost:5432/testdb",
			wantDriver:  "postgres",
			wantDSN:     "postgres://user:pass@localhost:5432/testdb",
			wantErr:     false,
		},
		{
			name:        "PostgreSQL DSN with SSL",
			dsn:         "postgres://user:pass@localhost:5432/testdb?sslmode=disable",
			wantDriver:  "postgres",
			wantDSN:     "postgres://user:pass@localhost:5432/testdb?sslmode=disable",
			wantErr:     false,
		},
		{
			name:        "SQLite absolute path",
			dsn:         "sqlite:///tmp/test.db",
			wantDriver:  "sqlite3",
			wantDSN:     "/tmp/test.db",
			wantErr:     false,
		},
		{
			name:        "SQLite relative path",
			dsn:         "sqlite://./test.db",
			wantDriver:  "sqlite3",
			wantDSN:     "./test.db",
			wantErr:     false,
		},
		{
			name:        "Unsupported scheme",
			dsn:         "mongodb://localhost:27017/test",
			wantDriver:  "",
			wantDSN:     "",
			wantErr:     true,
		},
		{
			name:        "Invalid DSN",
			dsn:         "not-a-valid-dsn",
			wantDriver:  "",
			wantDSN:     "",
			wantErr:     true,
		},
		{
			name:        "MySQL without host",
			dsn:         "mysql:///testdb",
			wantDriver:  "",
			wantDSN:     "",
			wantErr:     true,
		},
		{
			name:        "SQLite with host (invalid)",
			dsn:         "sqlite://localhost/test.db",
			wantDriver:  "",
			wantDSN:     "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDriver, gotDSN, err := parseDSN(tt.dsn)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDSN() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotDriver != tt.wantDriver {
				t.Errorf("parseDSN() gotDriver = %v, want %v", gotDriver, tt.wantDriver)
			}
			if gotDSN != tt.wantDSN {
				t.Errorf("parseDSN() gotDSN = %v, want %v", gotDSN, tt.wantDSN)
			}
		})
	}
}

func TestMaskDSN(t *testing.T) {
	tests := []struct {
		name string
		dsn  string
		want string
	}{
		{
			name: "MySQL with password",
			dsn:  "mysql://user:secretpass@localhost:3306/testdb",
			want: "mysql://user:%2A%2A%2A%2A@localhost:3306/testdb", // URL encoded ****
		},
		{
			name: "PostgreSQL with password",
			dsn:  "postgres://admin:mypassword@localhost:5432/testdb?sslmode=disable",
			want: "postgres://admin:%2A%2A%2A%2A@localhost:5432/testdb?sslmode=disable", // URL encoded ****
		},
		{
			name: "SQLite (no masking needed)",
			dsn:  "sqlite:///tmp/test.db",
			want: "sqlite:///tmp/test.db",
		},
		{
			name: "MySQL without password",
			dsn:  "mysql://user@localhost:3306/testdb",
			want: "mysql://user@localhost:3306/testdb",
		},
		{
			name: "Invalid DSN",
			dsn:  "not-a-valid-dsn",
			want: "not-a-valid-dsn", // SQLite file path returned as-is
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := maskDSN(tt.dsn); got != tt.want {
				t.Errorf("maskDSN() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseParams(t *testing.T) {
	tests := []struct {
		name    string
		with    map[string]string
		wantErr bool
	}{
		{
			name: "Valid MySQL DSN",
			with: map[string]string{
				"dsn":   "mysql://user:pass@localhost:3306/testdb",
				"query": "SELECT * FROM users",
			},
			wantErr: false,
		},
		{
			name: "Valid PostgreSQL DSN with timeout",
			with: map[string]string{
				"dsn":     "postgres://user:pass@localhost:5432/testdb",
				"query":   "SELECT * FROM users",
				"timeout": "60s",
			},
			wantErr: false,
		},
		{
			name: "Valid SQLite DSN with params",
			with: map[string]string{
				"dsn":       "sqlite:///tmp/test.db",
				"query":     "SELECT * FROM users WHERE id = ? AND name = ?",
				"params__0": "123",
				"params__1": "john",
			},
			wantErr: false,
		},
		{
			name: "Missing DSN",
			with: map[string]string{
				"query": "SELECT * FROM users",
			},
			wantErr: true,
		},
		{
			name: "Missing query",
			with: map[string]string{
				"dsn": "mysql://user:pass@localhost:3306/testdb",
			},
			wantErr: true,
		},
		{
			name: "Invalid timeout",
			with: map[string]string{
				"dsn":     "mysql://user:pass@localhost:3306/testdb",
				"query":   "SELECT * FROM users",
				"timeout": "invalid",
			},
			wantErr: true,
		},
		{
			name: "Invalid DSN scheme",
			with: map[string]string{
				"dsn":   "mongodb://localhost:27017/test",
				"query": "SELECT * FROM users",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseParams(tt.with)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseParams() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestActionRun_ValidationErrors(t *testing.T) {
	action := &Action{
		log: hclog.New(&hclog.LoggerOptions{
			Output: io.Discard,
			Level:  hclog.NoLevel,
		}),
	}

	tests := []struct {
		name    string
		with    map[string]string
		wantErr bool
	}{
		{
			name:    "Empty parameters",
			with:    map[string]string{},
			wantErr: true,
		},
		{
			name: "Missing DSN",
			with: map[string]string{
				"query": "SELECT * FROM users",
			},
			wantErr: true,
		},
		{
			name: "Missing query",
			with: map[string]string{
				"dsn": "sqlite:///tmp/test.db",
			},
			wantErr: true,
		},
		{
			name: "Empty DSN",
			with: map[string]string{
				"dsn":   "",
				"query": "SELECT * FROM users",
			},
			wantErr: true,
		},
		{
			name: "Empty query",
			with: map[string]string{
				"dsn":   "sqlite:///tmp/test.db",
				"query": "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Run([]string{}, tt.with)
			if (err != nil) != tt.wantErr {
				t.Errorf("Action.Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}