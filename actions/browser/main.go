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

func (a *Action) Run(args []string, with map[string]string) (map[string]string, error) {
	truncateLength := probe.MaxLogStringLength
	truncatedParams := probe.TruncateMapStringString(with, truncateLength)
	a.log.Debug("received browser action request", "params", truncatedParams)

	within := br.WithInBrowser(func(s string, i ...interface{}) {
		a.log.Debug("chromedp", "message", fmt.Sprintf(s, i...))
	})
	before := br.WithBefore(func(req *br.Req) {
		a.log.Debug("chromedp request prepared", "request", req)
	})
	after := br.WithAfter(func(res *br.Res) {
		a.log.Debug("chromedp response received", "response", res)
	})

	ret, err := br.Request(with, within, before, after)

	if err != nil {
		a.log.Error("browser request failed", "error", err)
	} else {
		truncatedResult := probe.TruncateMapStringString(ret, truncateLength)
		a.log.Debug("browser request completed successfully", "result", truncatedResult)
	}

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
