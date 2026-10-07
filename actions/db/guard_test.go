package db

import (
	"strings"
	"testing"

	"github.com/linyows/probe/actionrpc"
)

func TestReqCheckGuard(t *testing.T) {
	readOnly := actionrpc.Guard{ReadOnly: true}
	tests := []struct {
		name    string
		guard   actionrpc.Guard
		dsn     string
		query   string
		refused string // a part of the reason; empty when the query runs
	}{
		{name: "SELECT", guard: readOnly, query: "SELECT * FROM users"},
		{name: "lower case, with a trailing semicolon", guard: readOnly, query: "  select 1;  "},
		{name: "SHOW", guard: readOnly, query: "SHOW TABLES"},
		{name: "EXPLAIN", guard: readOnly, query: "EXPLAIN SELECT 1"},
		{name: "WITH", guard: readOnly, query: "WITH x AS (SELECT 1) SELECT * FROM x"},
		{name: "a semicolon in a string", guard: readOnly, query: "SELECT 'a;b', \"c;d\", `e;f`"},
		{name: "a semicolon in a comment", guard: readOnly, query: "SELECT 1 -- a; b\n, 2 /* c; d */"},
		{name: "INSERT", guard: readOnly, query: "INSERT INTO users VALUES (1)", refused: "the statement INSERT may write"},
		{name: "UPDATE", guard: readOnly, query: "update users set a = 1", refused: "the statement UPDATE may write"},
		{name: "two statements", guard: readOnly, query: "SELECT 1; DELETE FROM users", refused: "more than one statement"},
		{name: "a statement after a string", guard: readOnly, query: "SELECT 'it''s'; DROP TABLE users", refused: "more than one statement"},
		{name: "INSERT without a guard", query: "INSERT INTO users VALUES (1)"},
		{name: "a server allowed", guard: actionrpc.Guard{AllowHosts: []string{"db.internal"}}, dsn: "mysql://u:p@db.internal:3306/app", query: "INSERT INTO t VALUES (1)"},
		{name: "the default port", guard: actionrpc.Guard{AllowHosts: []string{"db.internal:5432"}}, dsn: "postgres://u:p@db.internal/app", query: "SELECT 1"},
		{name: "localhost without a host", guard: actionrpc.Guard{AllowHosts: []string{"localhost:3306"}}, dsn: "mysql://u:p@/app", query: "SELECT 1"},
		{name: "a server not allowed", guard: actionrpc.Guard{AllowHosts: []string{"db.internal"}}, dsn: "postgres://u:p@prod.example.com:5432/app", query: "SELECT 1", refused: "the host prod.example.com:5432 is not one the run allows"},
		{name: "SQLite names no host", guard: actionrpc.Guard{AllowHosts: []string{"db.internal"}}, dsn: "file:./test.db", query: "SELECT 1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dsn := tt.dsn
			if dsn == "" {
				dsn = "file::memory:"
			}
			driver, _, err := parseDSN(dsn)
			if err != nil {
				t.Fatal(err)
			}
			r := &Req{Driver: driver, DSN: dsn, Query: tt.query}
			err = r.checkGuard(tt.guard)
			if tt.refused == "" {
				if err != nil {
					t.Errorf("checkGuard = %v, want nil", err)
				}
				return
			}
			if !actionrpc.IsRefused(err) || !strings.Contains(err.Error(), tt.refused) {
				t.Errorf("checkGuard = %v, want a refusal saying %q", err, tt.refused)
			}
		})
	}
}

func TestHasStatementSeparator(t *testing.T) {
	tests := map[string]bool{
		"SELECT 1":                     false,
		"SELECT 1; SELECT 2":           true,
		"SELECT ';'":                   false,
		`SELECT "a;b"`:                 false,
		"SELECT `a;b`":                 false,
		`SELECT 'a\';' ; DELETE`:       true,
		"SELECT 1 -- ;\n":              false,
		"SELECT 1 /* ; */ , 2":         false,
		"SELECT 1 /* unclosed ;":       false,
		"SELECT 1 -- x\n; DELETE FROM": true,
	}
	for query, want := range tests {
		if got := hasStatementSeparator(query); got != want {
			t.Errorf("hasStatementSeparator(%q) = %v, want %v", query, got, want)
		}
	}
}
