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

func (a *Action) Run(args []string, with map[string]any) (map[string]any, error) {
	// Validate that required parameters are provided
	if len(with) == 0 {
		return map[string]any{}, errors.New("grpc action requires parameters in 'with' section. Please specify request details like addr, service, method")
	}

	// Use default truncate length, can be overridden by caller
	truncateLength := probe.MaxLogStringLength

	// Truncate long parameters for logging to prevent log bloat
	truncatedParams := probe.TruncateMapStringAny(with, truncateLength)
	a.log.Debug("received grpc request parameters", "params", truncatedParams)

	before := grpcpkg.WithBefore(func(ctx context.Context, service, method string) {
		a.log.Debug("grpc request prepared", "service", service, "method", method)
	})
	after := grpcpkg.WithAfter(func(res *grpcpkg.Res) {
		a.log.Debug("grpc response received", "status", res.StatusCode)
	})
	ret, err := grpcpkg.Request(with, before, after)

	if err != nil {
		a.log.Error("grpc request failed", "error", err)
	} else {
		// Truncate result for logging to prevent log bloat
		truncatedResult := probe.TruncateMapStringAny(ret, truncateLength)
		a.log.Debug("grpc request completed successfully", "result", truncatedResult)
	}

	return ret, err
}

func Serve() {
	log := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Info,
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
