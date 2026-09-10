package actions

import (
	"errors"

	"github.com/hashicorp/go-hclog"
	"github.com/linyows/probe/embedded"
)

func runEmbedded(log hclog.Logger, with map[string]any) (map[string]any, error) {
	// Validate that required parameters are provided
	if len(with) == 0 {
		return map[string]any{}, errors.New("embedded action requires parameters in 'with' section. Please specify embedded job details like path")
	}

	logParams(log, "received embedded request parameters", with)

	before := embedded.WithBefore(func(path string, vars map[string]any) {
		log.Debug("embedded job prepared", "path", path, "vars", vars)
	})
	after := embedded.WithAfter(func(result *embedded.Result) {
		log.Debug("embedded job completed", "result", result)
	})
	ret, err := embedded.Execute(with, before, after)

	logOutcome(log, "embedded job", ret, err)

	return ret, err
}
