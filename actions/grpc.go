package actions

import (
	"context"
	"errors"

	"github.com/hashicorp/go-hclog"
	grpcpkg "github.com/linyows/probe/grpc"
)

func runGRPC(log hclog.Logger, with map[string]any) (map[string]any, error) {
	// Validate that required parameters are provided
	if len(with) == 0 {
		return map[string]any{}, errors.New("grpc action requires parameters in 'with' section. Please specify request details like addr, service, method")
	}

	logParams(log, "received grpc request parameters", with)

	before := grpcpkg.WithBefore(func(ctx context.Context, service, method string) {
		log.Debug("grpc request prepared", "service", service, "method", method)
	})
	after := grpcpkg.WithAfter(func(res *grpcpkg.Res) {
		log.Debug("grpc response received", "status", res.StatusCode)
	})
	ret, err := grpcpkg.Request(with, before, after)

	logOutcome(log, "grpc request", ret, err)

	return ret, err
}
