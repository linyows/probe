package ssh

import (
	"errors"
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/linyows/probe"
	"github.com/linyows/probe/ssh"
)

type Action struct {
	log hclog.Logger
}

func (a *Action) Run(args []string, with map[string]string) (map[string]string, error) {
	// Validate that required parameters are provided
	if len(with) == 0 {
		return map[string]string{}, errors.New("ssh action requires parameters in 'with' section. Please specify connection details like host, user, cmd")
	}

	// Use default truncate length, can be overridden by caller
	truncateLength := probe.MaxLogStringLength

	// Truncate long parameters for logging to prevent log bloat
	// Note: Sensitive data like passwords and keys are excluded from logs for security
	truncatedParams := probe.TruncateMapStringString(with, truncateLength)

	// Remove sensitive information from logs
	logParams := make(map[string]string)
	for k, v := range truncatedParams {
		switch k {
		case "password", "key_passphrase":
			logParams[k] = "[REDACTED]"
		case "key_file":
			// Log only the filename, not the full path for security
			if v != "" {
				logParams[k] = "[KEY_FILE_PROVIDED]"
			}
		default:
			logParams[k] = v
		}
	}

	a.log.Debug("received ssh request parameters", "params", logParams)

	before := ssh.WithBefore(func(host string, port int, user string, cmd string) {
		a.log.Debug("ssh connection prepared", "host", host, "port", port, "user", user, "cmd", cmd)
	})
	after := ssh.WithAfter(func(result *ssh.Result) {
		// Truncate result for logging to prevent log bloat
		logResult := map[string]interface{}{
			"code":   result.Res.Code,
			"status": result.Status,
			"rt":     result.RT,
		}
		// Truncate stdout/stderr for logging
		if len(result.Res.Stdout) > truncateLength {
			logResult["stdout"] = result.Res.Stdout[:truncateLength] + "...[TRUNCATED]"
		} else {
			logResult["stdout"] = result.Res.Stdout
		}
		if len(result.Res.Stderr) > truncateLength {
			logResult["stderr"] = result.Res.Stderr[:truncateLength] + "...[TRUNCATED]"
		} else {
			logResult["stderr"] = result.Res.Stderr
		}
		a.log.Debug("ssh command completed", "result", logResult)
	})
	ret, err := ssh.Execute(with, before, after)

	if err != nil {
		a.log.Error("ssh command failed", "error", err)
	} else {
		// Truncate result for logging to prevent log bloat
		truncatedResult := probe.TruncateMapStringString(ret, truncateLength)
		a.log.Debug("ssh command completed successfully", "result_keys", getMapKeys(truncatedResult))
	}

	return ret, err
}

// getMapKeys returns the keys of a map for logging purposes
func getMapKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
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
