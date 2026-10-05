package http

import (
	"errors"
	hp "net/http"
	"strings"

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

	// The request and the response are logged as the parameters are, with
	// credentials such as cookies hidden: the runner learns those in the
	// response only once it has the result, after these records have gone.
	before := WithBefore(func(req *hp.Request) {
		actionrpc.LogParams(a.log, "http request prepared", map[string]any{
			"method":  req.Method,
			"url":     req.URL.String(),
			"headers": joinHeader(req.Header),
		})
	})
	after := WithAfter(func(res *hp.Response) {
		actionrpc.LogParams(a.log, "http response received", map[string]any{
			"status":  res.Status,
			"headers": joinHeader(res.Header),
		})
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

// joinHeader returns h with the values of each header joined, as the result
// shows them.
func joinHeader(h hp.Header) map[string]any {
	out := make(map[string]any, len(h))
	for k, v := range h {
		out[k] = strings.Join(v, ", ")
	}
	return out
}
