package shell

import (
	"errors"
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/linyows/probe/v2"
	"github.com/linyows/probe/v2/shell"
)

type Action struct {
	log hclog.Logger
}

func (a *Action) Run(with map[string]any) (map[string]any, error) {
	// Validate that required parameters are provided
	if len(with) == 0 {
		return map[string]any{}, errors.New("shell action requires parameters in 'with' section. Please specify command details like cmd")
	}

	probe.LogActionParams(a.log, "received shell request parameters", with)

	before := shell.WithBefore(func(cmd string, shell string, workdir string) {
		a.log.Debug("shell command prepared", "cmd", cmd, "shell", shell, "workdir", workdir)
	})
	after := shell.WithAfter(func(result *shell.Result) {
		a.log.Debug("shell command completed", "result", result)
	})
	ret, err := shell.Execute(with, before, after)

	probe.LogActionOutcome(a.log, "shell command", ret, err)

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
