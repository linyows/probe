package embedded

import (
	"errors"
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/linyows/probe"
	"github.com/linyows/probe/embedded"
)

type Action struct {
	log hclog.Logger
}

func (a *Action) Run(args []string, with map[string]any) (map[string]any, error) {
	// Validate that required parameters are provided
	if len(with) == 0 {
		return map[string]any{}, errors.New("embedded action requires parameters in 'with' section. Please specify embedded job details like path")
	}

	// Use default truncate length, can be overridden by caller
	truncateLength := probe.MaxLogStringLength

	// Truncate long parameters for logging to prevent log bloat
	truncatedParams := probe.TruncateMapStringAny(with, truncateLength)
	a.log.Debug("received embedded request parameters", "params", truncatedParams)

	before := embedded.WithBefore(func(path string, vars map[string]any) {
		a.log.Debug("embedded job prepared", "path", path, "vars", vars)
	})
	after := embedded.WithAfter(func(result *embedded.Result) {
		a.log.Debug("embedded job completed", "result", result)
	})
	ret, err := embedded.Execute(with, before, after)

	if err != nil {
		a.log.Error("embedded job execution failed", "error", err)
	} else {
		// Truncate result for logging to prevent log bloat
		truncatedResult := probe.TruncateMapStringAny(ret, truncateLength)
		a.log.Debug("embedded job completed successfully", "result", truncatedResult)
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
