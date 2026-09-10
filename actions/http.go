package actions

import (
	"errors"
	hp "net/http"

	"github.com/hashicorp/go-hclog"
	httpclient "github.com/linyows/probe/http"
)

func runHTTP(log hclog.Logger, with map[string]any) (map[string]any, error) {
	// Validate that required parameters are provided
	if len(with) == 0 {
		return map[string]any{}, errors.New("http action requires parameters in 'with' section. Please specify request details like url, method, or use method fields (get, post, etc.)")
	}

	logParams(log, "received request parameters", with)

	before := httpclient.WithBefore(func(req *hp.Request) {
		log.Debug("http request prepared", "request", req)
	})
	after := httpclient.WithAfter(func(res *hp.Response) {
		log.Debug("http response received", "response", res)
	})
	ret, err := httpclient.Request(with, before, after)

	logOutcome(log, "http request", ret, err)

	return ret, err
}
