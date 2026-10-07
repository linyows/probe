package db

import (
	"net"
	"net/url"
	"slices"
	"strings"

	"github.com/linyows/probe/actionrpc"
)

// defaultPorts are the ports of the servers a DSN without one connects to.
var defaultPorts = map[string]string{"mysql": "3306", "postgres": "5432"}

// readVerbs are the words a statement allowed under a read-only guard
// starts with.
var readVerbs = []string{"SELECT", "SHOW", "DESCRIBE", "DESC", "EXPLAIN", "WITH"}

// checkGuard returns a Refused error when the guard of the run does not
// allow the query: a server it does not allow, or, under --read-only, a
// statement that is not one that reads, or more than one statement. A
// SQLite database is a local file, which no host names.
//
// A statement that reads by its first word may still write, as a WITH that
// deletes does, so that under --read-only the query also runs in a
// read-only transaction, where the database refuses a write itself.
func (r *Req) checkGuard(guard actionrpc.Guard) error {
	if host, ok := r.host(); ok {
		if err := guard.CheckHost(host); err != nil {
			return err
		}
	}
	if !guard.ReadOnly {
		return nil
	}
	query := strings.TrimRight(strings.TrimSpace(r.Query), "; \t\r\n")
	if hasStatementSeparator(query) {
		return actionrpc.Refuse("the query holds more than one statement, and the run is read-only; give one that reads")
	}
	word := strings.ToUpper(strings.Fields(query + " ")[0])
	if !slices.Contains(readVerbs, word) {
		return actionrpc.Refuse("the statement %s may write, and the run is read-only; only %s are run", word, strings.Join(readVerbs, ", "))
	}
	return nil
}

// host returns the host and port of the server the DSN connects to, and
// false for a SQLite database.
func (r *Req) host() (string, bool) {
	if r.Driver == "sqlite" {
		return "", false
	}
	u, err := url.Parse(r.DSN)
	if err != nil {
		return "", false
	}
	host := u.Hostname()
	if host == "" {
		host = "localhost"
	}
	port := u.Port()
	if port == "" {
		port = defaultPorts[r.Driver]
	}
	return net.JoinHostPort(host, port), true
}

// hasStatementSeparator reports whether query holds a semicolon that ends a
// statement: one outside quotes, quoted identifiers and comments.
func hasStatementSeparator(query string) bool {
	for i := 0; i < len(query); i++ {
		switch c := query[i]; {
		case c == '\'' || c == '"' || c == '`':
			for i++; i < len(query) && query[i] != c; i++ {
				if query[i] == '\\' {
					i++
				}
			}
		case c == '-' && i+1 < len(query) && query[i+1] == '-':
			for i < len(query) && query[i] != '\n' {
				i++
			}
		case c == '/' && i+1 < len(query) && query[i+1] == '*':
			end := strings.Index(query[i+2:], "*/")
			if end < 0 {
				return false
			}
			i += end + 3
		case c == ';':
			return true
		}
	}
	return false
}
