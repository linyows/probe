package hello

import (
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/linyows/probe"
)

type Action struct {
	log hclog.Logger
}

func (a *Action) Run(args []string, with map[string]any) (map[string]any, error) {
	a.log.Info("Hello!")

	// Create response data
	res := make(map[string]any)
	for k, v := range with {
		res[k] = v
	}
	res["status"] = 0   // Always success for hello action (ExitStatusSuccess)
	res["dump"] = false // Don't dump request/response for hello action

	// Return in expected structure
	result := map[string]any{
		"req":    with,
		"res":    res,
		"rt":     "",
		"status": 0,
	}

	return result, nil
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
