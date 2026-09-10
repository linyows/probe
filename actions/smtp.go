package actions

import (
	"errors"

	"github.com/hashicorp/go-hclog"
	"github.com/linyows/probe/mail"
)

func runSMTP(log hclog.Logger, with map[string]any) (map[string]any, error) {
	// Validate that required parameters are provided
	if len(with) == 0 {
		return map[string]any{}, errors.New("smtp action requires parameters in 'with' section. Please specify email details like addr, from, to")
	}

	logParams(log, "received smtp request parameters", with)

	before := mail.WithBefore(func(from string, to string, subject string) {
		log.Debug("email prepared", "from", from, "to", to, "subject", subject)
	})
	after := mail.WithAfter(func(result *mail.Result) {
		log.Debug("email delivery completed", "result", result)
	})
	ret, err := mail.Send(with, before, after)

	logOutcome(log, "email delivery", ret, err)

	return ret, err
}
