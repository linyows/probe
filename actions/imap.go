package actions

import (
	"errors"

	"github.com/hashicorp/go-hclog"
	"github.com/linyows/probe/imap"
)

func runIMAP(log hclog.Logger, with map[string]any) (map[string]any, error) {
	// Validate that required parameters are provided
	if len(with) == 0 {
		return map[string]any{}, errors.New("imap action requires parameters in 'with' section. Please specify connection details like host, username, password")
	}

	logParams(log, "received imap request parameters", with)

	before := imap.WithBefore(func(req *imap.Req) {
		log.Debug("imap request prepared", "request", req)
	})
	after := imap.WithAfter(func(res *imap.Res) {
		log.Debug("imap response received", "response", res)
	})
	ret, err := imap.Request(with, before, after)

	logOutcome(log, "imap request", ret, err)

	return ret, err
}
