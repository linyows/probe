package db

import (
	"fmt"
	"net"
	"net/url"
	"os"
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
//
// Under --read-only the query is also refused when it holds a semicolon
// anywhere but at its end, so that it cannot hold a second statement, such
// as one that ends the read-only transaction or turns query_only off.
func (r *Req) checkGuard(guard actionrpc.Guard) error {
	if len(guard.AllowHosts) > 0 {
		host, ok, err := r.host()
		if err != nil {
			return actionrpc.Refuse("%v", err)
		}
		if ok {
			if err := guard.CheckHost(host); err != nil {
				return err
			}
		}
	}
	if !guard.ReadOnly {
		return nil
	}
	query := strings.TrimRight(strings.TrimSpace(r.Query), "; \t\r\n")
	words := strings.Fields(query)
	if len(words) == 0 {
		return actionrpc.Refuse("the query holds no statement")
	}
	// A semicolon is refused wherever it is, in a string or a comment too:
	// what ends a string or a comment differs from one database to another,
	// and from one connection setting to another, so that a semicolon the
	// guard took for text could end a statement and start one that writes.
	if strings.Contains(query, ";") {
		return actionrpc.Refuse("the query holds a semicolon, which may end a statement and start another, and the run is read-only; give one statement that reads, without one")
	}
	word := strings.ToUpper(words[0])
	if !slices.Contains(readVerbs, word) {
		return actionrpc.Refuse("the statement %s may write, and the run is read-only; only %s are run", word, strings.Join(readVerbs, ", "))
	}
	return nil
}

// host returns the host and port of the server the DSN connects to, and
// false for a SQLite database. For PostgreSQL it is the server lib/pq
// connects to: the host and port parameters of the DSN override the ones
// before the path, and PGHOST and PGPORT fill in what the DSN leaves out.
// A DSN it cannot tell the server of is an error, which refuses it.
func (r *Req) host() (string, bool, error) {
	if r.Driver == "sqlite" {
		return "", false, nil
	}
	u, err := url.Parse(r.DSN)
	if err != nil {
		return "", true, err
	}
	host, port := u.Hostname(), u.Port()
	if r.Driver == "postgres" {
		q := u.Query()
		if q.Has("hostaddr") || strings.Contains(q.Get("host"), ",") {
			return "", true, fmt.Errorf("the DSN names its server by hostaddr or by several hosts, which the run cannot check against the hosts it allows")
		}
		if h := q.Get("host"); h != "" {
			host = h
		}
		if p := q.Get("port"); p != "" {
			port = p
		}
		if host == "" {
			host = os.Getenv("PGHOST")
		}
		if port == "" {
			port = os.Getenv("PGPORT")
		}
	}
	if host == "" {
		host = "localhost"
	}
	if port == "" {
		port = defaultPorts[r.Driver]
	}
	return net.JoinHostPort(host, port), true, nil
}
