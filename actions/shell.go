package actions

import (
	"errors"

	"github.com/hashicorp/go-hclog"
	"github.com/linyows/probe/shell"
)

func runShell(log hclog.Logger, with map[string]any) (map[string]any, error) {
	// Validate that required parameters are provided
	if len(with) == 0 {
		return map[string]any{}, errors.New("shell action requires parameters in 'with' section. Please specify command details like cmd")
	}

	logParams(log, "received shell request parameters", with)

	before := shell.WithBefore(func(cmd string, shell string, workdir string) {
		log.Debug("shell command prepared", "cmd", cmd, "shell", shell, "workdir", workdir)
	})
	after := shell.WithAfter(func(result *shell.Result) {
		log.Debug("shell command completed", "result", result)
	})
	ret, err := shell.Execute(with, before, after)

	logOutcome(log, "shell command", ret, err)

	return ret, err
}
