package embedded

import (
	"errors"

	"github.com/hashicorp/go-hclog"
	"github.com/linyows/probe/actionrpc"
)

type Action struct {
	log hclog.Logger
}

func (a *Action) Run(with map[string]any) (map[string]any, error) {
	// Validate that required parameters are provided
	if len(with) == 0 {
		return map[string]any{}, errors.New("embedded action requires parameters in 'with' section. Please specify embedded job details like path")
	}

	actionrpc.LogParams(a.log, "received embedded request parameters", with)

	before := WithBefore(func(path string, vars map[string]any) {
		a.log.Debug("embedded job prepared", "path", path, "vars", vars)
	})
	after := WithAfter(func(result *Result) {
		a.log.Debug("embedded job completed", "result", result)
	})
	ret, err := Execute(with, before, after)

	actionrpc.LogOutcome(a.log, "embedded job", ret, err)

	return ret, err
}

func Serve() {
	actionrpc.Serve(func(log hclog.Logger) actionrpc.Action {
		return &Action{log: log}
	})
}
