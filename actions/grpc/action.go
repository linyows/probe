package grpc

import (
	"context"
	"errors"

	"github.com/hashicorp/go-hclog"
	"github.com/linyows/probe/actionrpc"
	"github.com/linyows/probe/mapping"
)

type Action struct {
	log hclog.Logger
}

func (a *Action) Run(with map[string]any) (map[string]any, error) {
	// Validate that required parameters are provided
	if len(with) == 0 {
		return map[string]any{}, errors.New("grpc action requires parameters in 'with' section. Please specify request details like addr, service, method")
	}

	actionrpc.LogParams(a.log, "received grpc request parameters", with)

	before := WithBefore(func(ctx context.Context, service, method string) {
		a.log.Debug("grpc request prepared", "service", service, "method", method)
	})
	after := WithAfter(func(res *Res) {
		a.log.Debug("grpc response received", "status", res.StatusCode)
	})
	ret, err := Request(with, before, after)

	actionrpc.LogOutcome(a.log, "grpc request", ret, err)

	return ret, err
}

func Serve() {
	actionrpc.Serve(func(log hclog.Logger) actionrpc.Action {
		return &Action{log: log}
	})
}

// Params returns the keys the grpc action takes in with: those of Req, and
// proto, which it reads apart from Req.
func Params() []string {
	return append(mapping.FieldTags(Req{}), "proto")
}
