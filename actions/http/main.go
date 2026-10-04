package http

import (
	"errors"
	hp "net/http"

	"github.com/hashicorp/go-hclog"
	"github.com/linyows/probe/actionrpc"
	"github.com/linyows/probe/http"
)

type Action struct {
	log hclog.Logger
}

func (a *Action) Run(with map[string]any) (map[string]any, error) {
	// Validate that required parameters are provided
	if len(with) == 0 {
		return map[string]any{}, errors.New("http action requires parameters in 'with' section. Please specify request details like url, method, or use method fields (get, post, etc.)")
	}

	actionrpc.LogParams(a.log, "received request parameters", with)

	before := http.WithBefore(func(req *hp.Request) {
		a.log.Debug("http request prepared", "request", req)
	})
	after := http.WithAfter(func(res *hp.Response) {
		a.log.Debug("http response received", "response", res)
	})
	ret, err := http.Request(with, before, after)

	actionrpc.LogOutcome(a.log, "http request", ret, err)

	return ret, err
}

func Serve() {
	actionrpc.Serve(func(log hclog.Logger) actionrpc.Action {
		return &Action{log: log}
	})
}
