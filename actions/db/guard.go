package db

import (
	"fmt"
	"net"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/lib/pq"
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
		hosts, ok, err := r.hosts()
		if err != nil {
			return actionrpc.Refuse("%v", err)
		}
		if ok {
			for _, host := range hosts {
				if err := guard.CheckHost(host); err != nil {
					return err
				}
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

// hosts returns the host and port of each server the DSN may connect to,
// and false for a SQLite database, which is a local file. For PostgreSQL
// they are the servers lib/pq resolves the DSN to, the same DSN it then
// connects with: its host, hostaddr and port parameters, a service file, and
// PGHOST, PGHOSTADDR, PGPORT and the like count, and every host of a list.
// A DSN it cannot tell the servers of is an error, which refuses it.
func (r *Req) hosts() ([]string, bool, error) {
	switch r.Driver {
	case "sqlite":
		return nil, false, nil
	case "postgres":
		cfg, err := pq.NewConfig(r.DSN)
		if err != nil {
			return nil, true, fmt.Errorf("the DSN cannot be read for its server: %w", err)
		}
		servers := []pq.ConfigMultihost{{Host: cfg.Host, Hostaddr: cfg.Hostaddr, Port: cfg.Port}}
		servers = append(servers, cfg.Multi...)
		var out []string
		for _, s := range servers {
			// hostaddr is the address dialed, when it is given.
			host := s.Host
			if s.Hostaddr.IsValid() {
				host = s.Hostaddr.String()
			}
			if host == "" {
				host = "localhost"
			}
			port := strconv.Itoa(int(s.Port))
			if s.Port == 0 {
				port = defaultPorts[r.Driver]
			}
			out = append(out, net.JoinHostPort(host, port))
		}
		return out, true, nil
	}
	u, err := url.Parse(r.DSN)
	if err != nil {
		return nil, true, err
	}
	host, port := u.Hostname(), u.Port()
	if host == "" {
		host = "localhost"
	}
	if port == "" {
		port = defaultPorts[r.Driver]
	}
	return []string{net.JoinHostPort(host, port)}, true, nil
}
