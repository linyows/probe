// Package actions maps the action names usable in a step's `uses` to the
// built-in plugins that implement them.
//
// Each built-in lives in its own package below this one and serves itself over
// the go-plugin protocol, exactly as an out-of-tree plugin would. This package
// only records which names exist, so that cmd/probe does not have to.
package actions

import (
	"sort"

	"github.com/linyows/probe/actions/browser"
	"github.com/linyows/probe/actions/db"
	"github.com/linyows/probe/actions/embedded"
	"github.com/linyows/probe/actions/grpc"
	"github.com/linyows/probe/actions/hello"
	"github.com/linyows/probe/actions/http"
	"github.com/linyows/probe/actions/imap"
	maillatency "github.com/linyows/probe/actions/mail-latency"
	"github.com/linyows/probe/actions/shell"
	"github.com/linyows/probe/actions/smtp"
	"github.com/linyows/probe/actions/ssh"
)

// builtin maps an action name to the function that serves it. Each value
// blocks until the workflow runner closes the plugin connection.
var builtin = map[string]func(){
	"browser":      browser.Serve,
	"db":           db.Serve,
	"embedded":     embedded.Serve,
	"grpc":         grpc.Serve,
	"hello":        hello.Serve,
	"http":         http.Serve,
	"imap":         imap.Serve,
	"mail-latency": maillatency.Serve,
	"shell":        shell.Serve,
	"smtp":         smtp.Serve,
	"ssh":          ssh.Serve,
}

// keeping are the built-in actions that keep to the guard of a run
// themselves: http, db and grpc refuse what it does not allow, embedded runs
// its job under it, and hello reaches nothing.
var keeping = []string{"db", "embedded", "grpc", "hello", "http"}

// Keeping returns the names of the built-in actions that keep to the guard
// of a run, such as --read-only, in alphabetical order. Any other action is
// refused under a guard unless it is allowed by name.
func Keeping() []string {
	return append([]string(nil), keeping...)
}

// params are the keys each built-in action takes in with. hello, which
// takes any, is left out.
var params = map[string]func() []string{
	"browser":      browser.Params,
	"db":           db.Params,
	"embedded":     embedded.Params,
	"grpc":         grpc.Params,
	"http":         http.Params,
	"imap":         imap.Params,
	"mail-latency": maillatency.Params,
	"shell":        shell.Params,
	"smtp":         smtp.Params,
	"ssh":          ssh.Params,
}

// Params returns the keys the built-in action name takes in with, and false
// for an action that takes any key, or that is not built in.
func Params(name string) ([]string, bool) {
	f, ok := params[name]
	if !ok {
		return nil, false
	}
	return f(), true
}

// AllParams returns the keys each built-in action takes in with, by its
// name, leaving out those that take any key.
func AllParams() map[string][]string {
	out := make(map[string][]string, len(params))
	for name, f := range params {
		out[name] = f()
	}
	return out
}

// Lookup returns the serve function of a built-in action.
func Lookup(name string) (func(), bool) {
	serve, ok := builtin[name]
	return serve, ok
}

// Names returns the names of the built-in actions in alphabetical order.
func Names() []string {
	names := make([]string, 0, len(builtin))
	for name := range builtin {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
