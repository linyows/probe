package http

import (
	"errors"
	hp "net/http"

	"github.com/hashicorp/go-hclog"
	"github.com/linyows/probe/actionrpc"
)

type Action struct {
	log hclog.Logger
}

func (a *Action) Run(with map[string]any) (map[string]any, error) {
	ret, _, err := a.RunWithState(with, nil)
	return ret, err
}

// RunWithState runs the request with the cookies the job keeps, when the
// step asks for keep_cookies, and returns the cookies to keep.
func (a *Action) RunWithState(with, state map[string]any) (map[string]any, map[string]any, error) {
	// Validate that required parameters are provided
	if len(with) == 0 {
		return map[string]any{}, nil, errors.New("http action requires parameters in 'with' section. Please specify request details like url, method, or use method fields (get, post, etc.)")
	}

	actionrpc.LogParams(a.log, "received request parameters", with)

	before := WithBefore(func(req *hp.Request) {
		a.log.Debug("http request prepared", "request", req)
	})
	after := WithAfter(func(res *hp.Response) {
		a.log.Debug("http response received", "response", res)
	})
	ret, newState, err := RequestWithState(with, state, before, after)

	actionrpc.LogOutcome(a.log, "http request", ret, err)

	return ret, newState, err
}

func Serve() {
	actionrpc.Serve(func(log hclog.Logger) actionrpc.Action {
		return &Action{log: log}
	})
}
