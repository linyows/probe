package browser

import (
	"fmt"

	"github.com/hashicorp/go-hclog"
	"github.com/linyows/probe/actionrpc"
	"github.com/linyows/probe/mapping"
)

type Action struct {
	log hclog.Logger
}

func (a *Action) Run(with map[string]any) (map[string]any, error) {
	actionrpc.LogParams(a.log, "received browser action request", with)

	within := WithInBrowser(func(s string, i ...any) {
		a.log.Debug("chromedp", "message", fmt.Sprintf(s, i...))
	})
	before := WithBefore(func(req *Req) {
		a.log.Debug("chromedp request prepared", "request", req)
	})
	after := WithAfter(func(res *Res) {
		a.log.Debug("chromedp response received", "response", res)
	})

	ret, err := Request(with, within, before, after)

	actionrpc.LogOutcome(a.log, "browser request", ret, err)

	return ret, err
}

func Serve() {
	actionrpc.Serve(func(log hclog.Logger) actionrpc.Action {
		log.Debug("Starting browser action server")
		return &Action{log: log}
	})
}

// Params returns the keys the browser action takes in with: those of Req,
// and timeout, which it reads apart from Req.
func Params() []string {
	return append(mapping.FieldTags(Req{}), "timeout")
}
