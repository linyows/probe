package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"

	"connectrpc.com/connect"
	"google.golang.org/grpc/status"
)

// unary serves method with the Connect protocol, answering an error
// with the code of the gRPC status it carries.
func unary[Req, Res any](mux *http.ServeMux, procedure string, method func(context.Context, *Req) (*Res, error)) {
	mux.Handle(procedure, connect.NewUnaryHandler(procedure, func(ctx context.Context, req *connect.Request[Req]) (*connect.Response[Res], error) {
		res, err := method(ctx, req.Msg)
		if err != nil {
			st := status.Convert(err)
			return nil, connect.NewError(connect.Code(st.Code()), errors.New(st.Message()))
		}
		return connect.NewResponse(res), nil
	}))
}

// StartConnect serves the UserService with the Connect protocol on port,
// over HTTP/1.1, as a server behind a proxy that passes no HTTP/2 is
// reached.
func (s *Server) StartConnect(port string) error {
	mux := http.NewServeMux()
	unary(mux, "/UserService/GetUser", s.GetUser)
	unary(mux, "/UserService/CreateUser", s.CreateUser)
	unary(mux, "/UserService/UpdateUser", s.UpdateUser)
	unary(mux, "/UserService/ListUsers", s.ListUsers)
	unary(mux, "/UserService/DeleteUser", s.DeleteUser)

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	s.connect = &http.Server{Handler: mux}
	go func() {
		if err := s.connect.Serve(lis); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Printf("Failed to serve Connect: %v\n", err)
		}
	}()
	return nil
}
