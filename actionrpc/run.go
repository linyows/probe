package actionrpc

import (
	"os"
	"os/exec"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
)

// Run starts the action named name from the running probe binary, calls it
// once with the given parameters, and stops it. The action's own log records
// reach log, which decides what of them is shown.
func Run(name string, with map[string]any, log hclog.Logger) (map[string]any, error) {
	cl := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig:  Handshake,
		Plugins:          PluginMap,
		Cmd:              exec.Command(os.Args[0], BuiltinCmd, name),
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolNetRPC, plugin.ProtocolGRPC},
		Logger:           log,
		UnixSocketConfig: &plugin.UnixSocketConfig{
			TempDir: os.TempDir(),
		},
	})
	defer cl.Kill()

	protocol, err := cl.Client()
	if err != nil {
		return nil, err
	}

	raw, err := protocol.Dispense("actions")
	if err != nil {
		return nil, err
	}

	action := raw.(Action)
	result, err := action.Run(with)
	if err != nil {
		return nil, err
	}
	return result, nil
}
