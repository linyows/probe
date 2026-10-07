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
		{name: "a comment", guard: readOnly, query: "SELECT 1 -- a comment\n, 2 /* another */"},
		{name: "a blank query", guard: readOnly, query: " ; ", refused: "the query holds no statement"},
		{name: "a semicolon in a string", guard: readOnly, query: "SELECT 'a;b'", refused: "the query holds a semicolon"},
		{name: "a semicolon in a comment", guard: readOnly, query: "SELECT 1 /* ; */", refused: "the query holds a semicolon"},
		// SQLite does not take a backslash to escape a quote, so the string
		// ends before the semicolon.
		{name: "a backslash before a quote", guard: readOnly, query: `SELECT '\'; PRAGMA query_only=OFF; DELETE FROM t`, refused: "the query holds a semicolon"},
		// MySQL takes -- for a comment only before a space, so 1--1 is 1 - -1.
		{name: "a double dash that is not a comment", guard: readOnly, query: "SELECT 1--1; COMMIT; DELETE FROM t", refused: "the query holds a semicolon"},
		{name: "INSERT", guard: readOnly, query: "INSERT INTO users VALUES (1)", refused: "the statement INSERT may write"},
		{name: "UPDATE", guard: readOnly, query: "update users set a = 1", refused: "the statement UPDATE may write"},
		{name: "two statements", guard: readOnly, query: "SELECT 1; DELETE FROM users", refused: "the query holds a semicolon"},
		{name: "INSERT without a guard", query: "INSERT INTO users VALUES (1)"},
		{name: "a server allowed", guard: actionrpc.Guard{AllowHosts: []string{"db.internal"}}, dsn: "mysql://u:p@db.internal:3306/app", query: "INSERT INTO t VALUES (1)"},
		{name: "the default port", guard: actionrpc.Guard{AllowHosts: []string{"db.internal:5432"}}, dsn: "postgres://u:p@db.internal/app", query: "SELECT 1"},
		{name: "localhost without a host", guard: actionrpc.Guard{AllowHosts: []string{"localhost:3306"}}, dsn: "mysql://u:p@/app", query: "SELECT 1"},
		{name: "a server not allowed", guard: actionrpc.Guard{AllowHosts: []string{"db.internal"}}, dsn: "postgres://u:p@prod.example.com:5432/app", query: "SELECT 1", refused: "the host prod.example.com:5432 is not one the run allows"},
		{name: "SQLite names no host", guard: actionrpc.Guard{AllowHosts: []string{"db.internal"}}, dsn: "file:./test.db", query: "SELECT 1"},
		// lib/pq connects to the host and port the parameters name.
		{name: "a PostgreSQL host parameter", guard: actionrpc.Guard{AllowHosts: []string{"allowed.example"}}, dsn: "postgres://u:p@allowed.example/app?host=prod.example&port=5433", query: "SELECT 1", refused: "the host prod.example:5433 is not one the run allows"},
		{name: "a PostgreSQL host parameter allowed", guard: actionrpc.Guard{AllowHosts: []string{"db.internal:6432"}}, dsn: "postgres://u:p@other/app?host=db.internal&port=6432", query: "SELECT 1"},
		{name: "a PostgreSQL hostaddr", guard: actionrpc.Guard{AllowHosts: []string{"allowed.example"}}, dsn: "postgres://u:p@allowed.example/app?hostaddr=10.0.0.9", query: "SELECT 1", refused: "hostaddr"},
		{name: "several PostgreSQL hosts", guard: actionrpc.Guard{AllowHosts: []string{"a.example"}}, dsn: "postgres://u:p@/app?host=a.example,b.example", query: "SELECT 1", refused: "several hosts"},
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

func TestReqHostTakesPGHOST(t *testing.T) {
	t.Setenv("PGHOST", "env.example")
	t.Setenv("PGPORT", "6543")
	r := &Req{Driver: "postgres", DSN: "postgres://u:p@/app", Query: "SELECT 1"}
	err := r.checkGuard(actionrpc.Guard{AllowHosts: []string{"localhost"}})
	if !actionrpc.IsRefused(err) || !strings.Contains(err.Error(), "the host env.example:6543") {
		t.Errorf("checkGuard = %v, want the host PGHOST and PGPORT name refused", err)
	}
	// A host the DSN names wins over PGHOST.
	r.DSN = "postgres://u:p@localhost/app"
	if err := r.checkGuard(actionrpc.Guard{AllowHosts: []string{"localhost"}}); err != nil {
		t.Errorf("checkGuard = %v, want nil", err)
	}
}
