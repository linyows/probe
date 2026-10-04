package actionrpc

import (
	"crypto/sha256"
	"os"
	"os/exec"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
)

// Run starts the action named name from the running probe binary, calls it
// once with the given parameters, and stops it. The action's own log records
// reach log, which decides what of them is shown.
func Run(name string, with map[string]any, log hclog.Logger) (map[string]any, error) {
	return run(exec.Command(os.Args[0], BuiltinCmd, name), nil, with, log)
}

// RunExecutable starts the action served by the executable at path, calls it
// once with the given parameters, and stops it, as Run does for a built-in
// one. When sum is not empty, the executable is started only if its SHA-256
// digest is sum.
func RunExecutable(path string, sum []byte, with map[string]any, log hclog.Logger) (map[string]any, error) {
	var secure *plugin.SecureConfig
	if len(sum) > 0 {
		secure = &plugin.SecureConfig{Checksum: sum, Hash: sha256.New()}
	}
	return run(exec.Command(path), secure, with, log)
}

func run(cmd *exec.Cmd, secure *plugin.SecureConfig, with map[string]any, log hclog.Logger) (map[string]any, error) {
	cl := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig:  Handshake,
		Plugins:          PluginMap,
		Cmd:              cmd,
		SecureConfig:     secure,
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
