package grpc

import (
	"context"
	"errors"

	"github.com/hashicorp/go-hclog"
	"github.com/linyows/probe/actionrpc"
	grpcpkg "github.com/linyows/probe/grpc"
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

	before := grpcpkg.WithBefore(func(ctx context.Context, service, method string) {
		a.log.Debug("grpc request prepared", "service", service, "method", method)
	})
	after := grpcpkg.WithAfter(func(res *grpcpkg.Res) {
		a.log.Debug("grpc response received", "status", res.StatusCode)
	})
	ret, err := grpcpkg.Request(with, before, after)

	actionrpc.LogOutcome(a.log, "grpc request", ret, err)

	return ret, err
}

func Serve() {
	actionrpc.Serve(func(log hclog.Logger) actionrpc.Action {
		return &Action{log: log}
	})
}
