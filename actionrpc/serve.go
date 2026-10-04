package actionrpc

import (
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
)

// Serve serves the action that newAction builds to the workflow runner that
// started this process, and returns once the runner is done with it. The
// action logs through the logger it is given, which writes JSON to stderr
// for the runner to filter.
func Serve(newAction func(log hclog.Logger) Action) {
	log := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Debug,
		Output:     os.Stderr,
		JSONFormat: true,
	})

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: Handshake,
		Plugins:         map[string]plugin.Plugin{"actions": &Plugin{Impl: newAction(log)}},
		GRPCServer:      plugin.DefaultGRPCServer,
	})
}
