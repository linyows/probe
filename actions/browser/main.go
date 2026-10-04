package browser

import (
	"fmt"

	"github.com/hashicorp/go-hclog"
	"github.com/linyows/probe/actionrpc"
	br "github.com/linyows/probe/browser"
)

type Action struct {
	log hclog.Logger
}

func (a *Action) Run(with map[string]any) (map[string]any, error) {
	actionrpc.LogParams(a.log, "received browser action request", with)

	within := br.WithInBrowser(func(s string, i ...any) {
		a.log.Debug("chromedp", "message", fmt.Sprintf(s, i...))
	})
	before := br.WithBefore(func(req *br.Req) {
		a.log.Debug("chromedp request prepared", "request", req)
	})
	after := br.WithAfter(func(res *br.Res) {
		a.log.Debug("chromedp response received", "response", res)
	})

	ret, err := br.Request(with, within, before, after)

	actionrpc.LogOutcome(a.log, "browser request", ret, err)

	return ret, err
}

func Serve() {
	actionrpc.Serve(func(log hclog.Logger) actionrpc.Action {
		log.Debug("Starting browser action server")
		return &Action{log: log}
	})
}
