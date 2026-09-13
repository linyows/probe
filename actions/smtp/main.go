package smtp

import (
	"errors"
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/linyows/probe/v2"
	"github.com/linyows/probe/v2/mail"
)

type Action struct {
	log hclog.Logger
}

func (a *Action) Run(with map[string]any) (map[string]any, error) {
	// Validate that required parameters are provided
	if len(with) == 0 {
		return map[string]any{}, errors.New("smtp action requires parameters in 'with' section. Please specify email details like addr, from, to")
	}

	probe.LogActionParams(a.log, "received smtp request parameters", with)

	before := mail.WithBefore(func(from string, to string, subject string) {
		a.log.Debug("email prepared", "from", from, "to", to, "subject", subject)
	})
	after := mail.WithAfter(func(result *mail.Result) {
		a.log.Debug("email delivery completed", "result", result)
	})
	ret, err := mail.Send(with, before, after)

	probe.LogActionOutcome(a.log, "email delivery", ret, err)

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
