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

var (
	BuiltinCmd = "builtin-actions"
	Handshake  = plugin.HandshakeConfig{ProtocolVersion: 1, MagicCookieKey: "probe", MagicCookieValue: "actions"}
	PluginMap  = map[string]plugin.Plugin{"actions": &ActionsPlugin{}}
)

type ActionsArgs []string
type ActionsParams map[string]string

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
	res := map[string]string{}
	runRes, err := m.client.Run(context.Background(), &pb.RunRequest{
		Args: args,
		With: with,
	})

	if err != nil {
		return res, err
	}

	return runRes.Result, err
}

type ActionsServer struct {
	Impl Actions
}

func (m *ActionsServer) Run(ctx context.Context, req *pb.RunRequest) (*pb.RunResponse, error) {
	v, err := m.Impl.Run(req.Args, req.With)
	return &pb.RunResponse{Result: v}, err
}

func RunActions(name string, args []string, with map[string]any, verbose bool) (map[string]any, error) {
	loglevel := hclog.Warn
	if verbose {
		loglevel = hclog.Debug
	}

	log := hclog.New(&hclog.LoggerOptions{
		Name:   "actions",
		Output: os.Stdout,
		Level:  loglevel,
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

	flatW := FlattenInterface(with)
	result, err := actions.Run(args, flatW)
	if err != nil {
		return nil, err
	}
	unflatR := UnflattenInterface(result)

	return unflatR, nil
}
