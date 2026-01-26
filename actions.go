package probe

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/exec"
	"reflect"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/linyows/probe/pb"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"
)

var (
	BuiltinCmd = "builtin-actions"
	Handshake  = plugin.HandshakeConfig{ProtocolVersion: 1, MagicCookieKey: "probe", MagicCookieValue: "actions"}
	PluginMap  = map[string]plugin.Plugin{"actions": &ActionsPlugin{}}
)

type ActionsArgs []string
type ActionsParams map[string]any

type Actions interface {
	Run(args []string, with map[string]any) (map[string]any, error)
}

type ActionsPlugin struct {
	plugin.Plugin
	Impl Actions
}

func (p *ActionsPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	log := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Debug,
		Output:     os.Stderr,
		JSONFormat: true,
	})
	pb.RegisterActionsServer(s, &ActionsServer{Impl: p.Impl, log: log})
	return nil
}

func (p *ActionsPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (any, error) {
	return &ActionsClient{client: pb.NewActionsClient(c)}, nil
}

type ActionsClient struct {
	client pb.ActionsClient
}

func (m *ActionsClient) Run(args []string, with map[string]any) (map[string]any, error) {
	// Convert map[string]any directly to protobuf.Struct
	withStruct, err := structpb.NewStruct(with)
	if err != nil {
		return nil, fmt.Errorf("failed to convert parameters to protobuf struct: %v", err)
	}

	runRes, err := m.client.Run(context.Background(), &pb.RunRequest{
		Args: args,
		With: withStruct,
	})

	if err != nil {
		return nil, err
	}

	// Convert protobuf.Struct back to map[string]any
	// Apply convertFloatToInt to restore integer types that were converted to float64 by protobuf
	if runRes.Result != nil {
		result := runRes.Result.AsMap()
		if converted, ok := convertFloatToInt(result).(map[string]any); ok {
			return converted, nil
		}
		return result, nil
	}

	return map[string]any{}, nil
}

type ActionsServer struct {
	Impl Actions
	log  hclog.Logger
}

func (m *ActionsServer) Run(ctx context.Context, req *pb.RunRequest) (*pb.RunResponse, error) {
	if m.log != nil {
		m.log.Debug("ActionsServer.Run called", "request", req)
	}

	// Convert protobuf.Struct to map[string]any
	// Apply convertFloatToInt to restore integer types that were converted to float64 by protobuf
	var withMap map[string]any
	if req.With != nil {
		result := req.With.AsMap()
		if converted, ok := convertFloatToInt(result).(map[string]any); ok {
			withMap = converted
		} else {
			withMap = result
		}
	} else {
		withMap = make(map[string]any)
	}

	v, err := m.Impl.Run(req.Args, withMap)
	if err != nil {
		if m.log != nil {
			m.log.Error("Action execution failed", "error", err)
		}
		return &pb.RunResponse{}, err
	}

	if m.log != nil {
		m.log.Debug("ActionsServer received from action", "result", v)
	}

	// Convert map[string]any to protobuf.Struct
	convertedResult := convertForProtobuf(v)
	resultMap, ok := convertedResult.(map[string]any)
	if !ok {
		if m.log != nil {
			m.log.Error("ActionsServer convertForProtobuf did not return map[string]any", "type", fmt.Sprintf("%T", convertedResult))
		}
		return &pb.RunResponse{}, fmt.Errorf("convertForProtobuf returned invalid type: expected map[string]any, got %T", convertedResult)
	}

	resultStruct, err := structpb.NewStruct(resultMap)
	if err != nil {
		if m.log != nil {
			m.log.Error("ActionsServer failed to convert result to protobuf struct", "error", err)
		}
		return &pb.RunResponse{}, fmt.Errorf("failed to convert result to protobuf struct: %v", err)
	}

	if m.log != nil {
		m.log.Debug("ActionsServer final result struct", "result", resultStruct)
	}
	return &pb.RunResponse{Result: resultStruct}, nil
}

// convertForProtobuf converts unsupported types to protobuf-compatible types
func convertForProtobuf(value any) any {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case time.Duration:
		// Convert Duration to string
		return v.String()
	case time.Time:
		// Convert Time to RFC3339 string format
		return v.Format(time.RFC3339)
	case map[string]string:
		// Convert map[string]string to map[string]any
		result := make(map[string]any)
		for k, val := range v {
			result[k] = val
		}
		return result
	case map[string]any:
		// Recursively convert nested maps
		result := make(map[string]any)
		for k, val := range v {
			result[k] = convertForProtobuf(val)
		}
		return result
	case []any:
		// Recursively convert arrays
		result := make([]any, len(v))
		for i, val := range v {
			result[i] = convertForProtobuf(val)
		}
		return result
	case []string:
		// Convert []string to []any
		result := make([]any, len(v))
		for i, str := range v {
			result[i] = str
		}
		return result
	default:
		// Check if it's a slice using reflection
		rv := reflect.ValueOf(value)
		if rv.Kind() == reflect.Slice {
			// Handle other slice types
			result := make([]any, rv.Len())
			for i := 0; i < rv.Len(); i++ {
				result[i] = convertForProtobuf(rv.Index(i).Interface())
			}
			return result
		}
		// Check if it's a pointer using reflection
		if rv.Kind() == reflect.Ptr {
			if rv.IsNil() {
				return nil
			}
			// Dereference pointer and recurse
			return convertForProtobuf(rv.Elem().Interface())
		}
		// Check if it's a struct using reflection
		if rv.Kind() == reflect.Struct {
			// Convert struct to map[string]any
			result := make(map[string]any)
			rt := rv.Type()
			for i := 0; i < rv.NumField(); i++ {
				field := rt.Field(i)
				fieldValue := rv.Field(i)

				// Skip unexported fields
				if !field.IsExported() {
					continue
				}

				// Use struct tag if available, otherwise use field name
				fieldName := field.Name
				if mapTag := field.Tag.Get("map"); mapTag != "" {
					fieldName = mapTag
				}

				result[fieldName] = convertForProtobuf(fieldValue.Interface())
			}
			return result
		}
		// Check if it's a map with string keys using reflection
		if rv.Kind() == reflect.Map {
			// Handle map[string]interface{} and similar types
			result := make(map[string]any)
			for _, key := range rv.MapKeys() {
				if keyStr, ok := key.Interface().(string); ok {
					mapValue := rv.MapIndex(key).Interface()
					result[keyStr] = convertForProtobuf(mapValue)
				}
			}
			return result
		}
		// Return as-is for supported types (string, int, float64, bool, etc.)
		return value
	}
}

// convertFloatToInt converts float64 values that represent integers back to int64.
// This is needed because protobuf.Struct converts all numbers to float64,
// which causes large integers like Unix timestamps to display in E notation.
func convertFloatToInt(value any) any {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case float64:
		// Check if the float64 is actually an integer value
		if v == math.Trunc(v) && v >= math.MinInt64 && v <= math.MaxInt64 {
			return int64(v)
		}
		return v
	case map[string]any:
		result := make(map[string]any)
		for k, val := range v {
			result[k] = convertFloatToInt(val)
		}
		return result
	case []any:
		result := make([]any, len(v))
		for i, val := range v {
			result[i] = convertFloatToInt(val)
		}
		return result
	default:
		return value
	}
}

// ActionRunner defines the interface for running actions
type ActionRunner interface {
	RunActions(name string, args []string, with map[string]any, verbose bool) (map[string]any, error)
}

// PluginActionRunner implements ActionRunner using the plugin system
type PluginActionRunner struct{}

// RunActions executes an action using the plugin system
func (p *PluginActionRunner) RunActions(name string, args []string, with map[string]any, verbose bool) (map[string]any, error) {
	loglevel := hclog.Warn
	if verbose {
		loglevel = hclog.Debug
	}
	log := hclog.New(&hclog.LoggerOptions{
		Name:   "actions",
		Output: os.Stderr,
		Level:  loglevel,
	})
	cl := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig:  Handshake,
		Plugins:          PluginMap,
		Cmd:              exec.Command(os.Args[0], BuiltinCmd, name),
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolNetRPC, plugin.ProtocolGRPC},
		Logger:           log,
		UnixSocketConfig: &plugin.UnixSocketConfig{
			TempDir: os.TempDir(),
		},
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

// MockActionRunner implements ActionRunner for testing
type MockActionRunner struct {
	Results map[string]map[string]any
	Errors  map[string]error
}

// NewMockActionRunner creates a new mock action runner
func NewMockActionRunner() *MockActionRunner {
	return &MockActionRunner{
		Results: make(map[string]map[string]any),
		Errors:  make(map[string]error),
	}
}

// SetResult sets the expected result for an action
func (m *MockActionRunner) SetResult(actionName string, result map[string]any) {
	m.Results[actionName] = result
}

// SetError sets the expected error for an action
func (m *MockActionRunner) SetError(actionName string, err error) {
	m.Errors[actionName] = err
}

// RunActions returns the mocked result or error for the given action
func (m *MockActionRunner) RunActions(name string, args []string, with map[string]any, verbose bool) (map[string]any, error) {
	if err, exists := m.Errors[name]; exists {
		return nil, err
	}

	if result, exists := m.Results[name]; exists {
		return result, nil
	}

	// Default mock response
	return map[string]any{
		"code":    0,
		"mock":    true,
		"action":  name,
		"with":    with,
		"results": map[string]any{},
	}, nil
}
