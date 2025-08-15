package smtp

import (
	"errors"
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/linyows/probe"
	"github.com/linyows/probe/mail"
)

type Action struct {
	log hclog.Logger
}

func (a *Action) Run(args []string, with map[string]string) (map[string]string, error) {
	// Validate that required parameters are provided
	if len(with) == 0 {
		return map[string]string{}, errors.New("smtp action requires parameters in 'with' section. Please specify email details like addr, from, to")
	}

	// Use default truncate length, can be overridden by caller
	truncateLength := probe.MaxLogStringLength

	// Truncate long parameters for logging to prevent log bloat
	truncatedParams := probe.TruncateMapStringString(with, truncateLength)
	a.log.Debug("received smtp request parameters", "params", truncatedParams)

	before := mail.WithBefore(func(from string, to string, subject string) {
		a.log.Debug("email prepared", "from", from, "to", to, "subject", subject)
	})
	after := mail.WithAfter(func(result *mail.Result) {
		a.log.Debug("email delivery completed", "result", result)
	})
	ret, err := mail.Send(with, before, after)

	if err != nil {
		a.log.Error("email delivery failed", "error", err)
	} else {
		// Truncate result for logging to prevent log bloat
		truncatedResult := probe.TruncateMapStringString(ret, truncateLength)
		a.log.Debug("email delivery completed successfully", "result", truncatedResult)
	}

	return ret, err
}

func Serve() {
	log := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Info,
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
