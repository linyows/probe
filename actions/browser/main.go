package browser

import (
	"fmt"
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/linyows/probe"
	br "github.com/linyows/probe/browser"
)

type Action struct {
	log hclog.Logger
}

func (a *Action) Run(with map[string]any) (map[string]any, error) {
	probe.LogActionParams(a.log, "received browser action request", with)

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

	probe.LogActionOutcome(a.log, "browser request", ret, err)

	return ret, err
}

func Serve() {
	log := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Debug,
		Output:     os.Stderr,
		JSONFormat: true,
	})

	log.Debug("Starting browser action server")

	pl := &probe.ActionsPlugin{
		Impl: &Action{log: log},
	}

	log.Debug("Browser action plugin created")

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: probe.Handshake,
		Plugins:         map[string]plugin.Plugin{"actions": pl},
		GRPCServer:      plugin.DefaultGRPCServer,
	})
}
