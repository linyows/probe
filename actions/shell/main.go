package shell

import (
	"errors"
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/linyows/probe"
	"github.com/linyows/probe/shell"
)

type Action struct {
	log hclog.Logger
}

func (a *Action) Run(args []string, with map[string]any) (map[string]any, error) {
	// Validate that required parameters are provided
	if len(with) == 0 {
		return map[string]any{}, errors.New("shell action requires parameters in 'with' section. Please specify command details like cmd")
	}

	// Use default truncate length, can be overridden by caller
	truncateLength := probe.MaxLogStringLength

	// Truncate long parameters for logging to prevent log bloat
	truncatedParams := probe.TruncateMapStringAny(with, truncateLength)
	a.log.Debug("received shell request parameters", "params", truncatedParams)

	before := shell.WithBefore(func(cmd string, shell string, workdir string) {
		a.log.Debug("shell command prepared", "cmd", cmd, "shell", shell, "workdir", workdir)
	})
	after := shell.WithAfter(func(result *shell.Result) {
		a.log.Debug("shell command completed", "result", result)
	})
	ret, err := shell.Execute(with, before, after)

	if err != nil {
		a.log.Error("shell command failed", "error", err)
	} else {
		// Truncate result for logging to prevent log bloat
		truncatedResult := probe.TruncateMapStringAny(ret, truncateLength)
		a.log.Debug("shell command completed successfully", "result", truncatedResult)
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
