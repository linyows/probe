// Package actions maps the action names usable in a step's `uses` to the
// built-in plugins that implement them.
//
// Each built-in lives in its own package below this one and serves itself over
// the go-plugin protocol, exactly as an out-of-tree plugin would. This package
// only records which names exist, so that cmd/probe does not have to.
package actions

import (
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

// Lookup returns the serve function of a built-in action.
func Lookup(name string) (func(), bool) {
	serve, ok := builtin[name]
	return serve, ok
}
