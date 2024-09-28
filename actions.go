package probe

import (
	"context"
	"os"
	"os/exec"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/linyows/probe/pb"
	"google.golang.org/grpc"
)

var BuiltinCmd = "builtin-actions"

var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "PROBE",
	MagicCookieValue: "actions",
}

var PluginMap = map[string]plugin.Plugin{
	"actions": &ActionsPlugin{},
}

type Actions interface {
	Run(args []string, with map[string]string) (map[string]string, error)
}

type ActionsPlugin struct {
	plugin.Plugin
	Impl Actions
}

func (p *ActionsPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	pb.RegisterActionsServer(s, &ActionsServer{Impl: p.Impl})
	return nil
}

func (p *ActionsPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (any, error) {
	return &ActionsClient{client: pb.NewActionsClient(c)}, nil
}

type ActionsClient struct {
	client pb.ActionsClient
}

func (m *ActionsClient) Run(args []string, with map[string]string) (map[string]string, error) {
	runRes, err := m.client.Run(context.Background(), &pb.RunRequest{
		Args: args,
		With: with,
	})
	return runRes.Result, err
}

type ActionsServer struct {
	Impl Actions
}

func (m *ActionsServer) Run(ctx context.Context, req *pb.RunRequest) (*pb.RunResponse, error) {
	v, err := m.Impl.Run(req.Args, req.With)
	return &pb.RunResponse{Result: v}, err
}

func RunActions(name string, args []string, with map[string]string) (any, error) {
	log := hclog.New(&hclog.LoggerOptions{
		Name:   "actions",
		Output: os.Stdout,
		Level:  hclog.Info,
	})

	cl := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig:  Handshake,
		Plugins:          PluginMap,
		Cmd:              exec.Command(os.Args[0], BuiltinCmd, name),
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolNetRPC, plugin.ProtocolGRPC},
		Logger:           log,
	})
	defer cl.Kill()

	protocol, err := cl.Client()
	if err != nil {
		return nil, err
	}

	raw, err := protocol.Dispense("actions")
	if err != nil {
		return nil, err
	}

	actions := raw.(Actions)
	result, err := actions.Run(args, with)
	if err != nil {
		return nil, err
	}

	return result, nil
}
