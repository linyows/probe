package smtp

import (
	"errors"

	"github.com/hashicorp/go-hclog"
	"github.com/linyows/probe/actionrpc"
	"github.com/linyows/probe/mail"
)

type Action struct {
	log hclog.Logger
}

func (a *Action) Run(with map[string]any) (map[string]any, error) {
	// Validate that required parameters are provided
	if len(with) == 0 {
		return map[string]any{}, errors.New("smtp action requires parameters in 'with' section. Please specify email details like addr, from, to")
	}

	actionrpc.LogParams(a.log, "received smtp request parameters", with)

	before := mail.WithBefore(func(from string, to string, subject string) {
		a.log.Debug("email prepared", "from", from, "to", to, "subject", subject)
	})
	after := mail.WithAfter(func(result *mail.Result) {
		a.log.Debug("email delivery completed", "result", result)
	})
	ret, err := mail.Send(with, before, after)

	actionrpc.LogOutcome(a.log, "email delivery", ret, err)

	return ret, err
}

func Serve() {
	actionrpc.Serve(func(log hclog.Logger) actionrpc.Action {
		return &Action{log: log}
	})
}
