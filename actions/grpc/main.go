package grpc

import (
	"context"
	"errors"
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/linyows/probe"
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

	probe.LogActionParams(a.log, "received grpc request parameters", with)

	before := grpcpkg.WithBefore(func(ctx context.Context, service, method string) {
		a.log.Debug("grpc request prepared", "service", service, "method", method)
	})
	after := grpcpkg.WithAfter(func(res *grpcpkg.Res) {
		a.log.Debug("grpc response received", "status", res.StatusCode)
	})
	ret, err := grpcpkg.Request(with, before, after)

	probe.LogActionOutcome(a.log, "grpc request", ret, err)

	return ret, err
}

func Serve() {
	log := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Debug,
		Output:     os.Stderr,
		JSONFormat: true,
	})

	pl := &probe.ActionsPlugin{
		Impl: &Action{log: log},
	}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: probe.Handshake,
		Plugins:         map[string]plugin.Plugin{"actions": pl},
		GRPCServer:      plugin.DefaultGRPCServer,
	})
}
