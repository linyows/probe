package imap

import (
	"errors"
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/linyows/probe"
	"github.com/linyows/probe/imap"
)

type Action struct {
	log hclog.Logger
}

func (a *Action) Run(args []string, with map[string]string) (map[string]string, error) {
	// Validate that required parameters are provided
	if len(with) == 0 {
		return map[string]string{}, errors.New("imap action requires parameters in 'with' section. Please specify connection details like host, username, password")
	}

	// Use default truncate length, can be overridden by caller
	truncateLength := probe.MaxLogStringLength

	// Truncate long parameters for logging to prevent log bloat
	truncatedParams := probe.TruncateMapStringString(with, truncateLength)
	a.log.Debug("received imap request parameters", "params", truncatedParams)

	before := imap.WithBefore(func(req *imap.Req) {
		a.log.Debug("imap request prepared", "request", req)
	})
	after := imap.WithAfter(func(res *imap.Res) {
		a.log.Debug("imap response received", "response", res)
	})
	ret, err := imap.Request(with, before, after)

	if err != nil {
		a.log.Error("imap request failed", "error", err)
	} else {
		// Truncate result for logging to prevent log bloat
		truncatedResult := probe.TruncateMapStringString(ret, truncateLength)
		a.log.Debug("imap request completed successfully", "result", truncatedResult)
	}

	return ret, err
}

func Serve() {
	log := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Debug,
		Output:     os.Stderr,
		JSONFormat: true,
	})

	pl := &probe.ActionsPlugin{
		Impl: &Action{log: log},
	}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: probe.Handshake,
		Plugins:         map[string]plugin.Plugin{"actions": pl},
		GRPCServer:      plugin.DefaultGRPCServer,
	})
}
