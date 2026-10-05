// Package actionrpc is the protocol between the workflow runner and its
// actions. Each action runs in a process of its own, started from the probe
// binary, and is called over go-plugin's gRPC transport.
package actionrpc

import (
	"context"
	"fmt"
	"math"
	"os"
	"reflect"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/linyows/probe/mask"
	"github.com/linyows/probe/pb"
	"github.com/linyows/probe/truncate"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"
)

var (
	// BuiltinCmd is the subcommand of the probe binary that serves an action.
	BuiltinCmd = "builtin-actions"
	// Handshake is the go-plugin handshake both sides of an action agree on.
	Handshake = plugin.HandshakeConfig{ProtocolVersion: 1, MagicCookieKey: "probe", MagicCookieValue: "actions"}
	// PluginMap is the set of plugins the runner can dispense.
	PluginMap = map[string]plugin.Plugin{"actions": &Plugin{}}
)

// Action is what an action implements: it runs once with the step's `with`
// parameters and returns its result.
type Action interface {
	Run(with map[string]any) (map[string]any, error)
}

// Plugin serves an Action over gRPC, and is what the runner dispenses to call
// one.
type Plugin struct {
	plugin.Plugin
	Impl Action
}

func (p *Plugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	log := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Debug,
		Output:     os.Stderr,
		JSONFormat: true,
	})
	pb.RegisterActionsServer(s, &Server{Impl: p.Impl, log: log})
	return nil
}

func (p *Plugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (any, error) {
	return &Client{client: pb.NewActionsClient(c)}, nil
}

// Client calls an Action in another process.
type Client struct {
	client pb.ActionsClient
}

// Run calls the action with the given parameters.
func (m *Client) Run(with map[string]any) (map[string]any, error) {
	// Convert map[string]any directly to protobuf.Struct
	withStruct, err := structpb.NewStruct(with)
	if err != nil {
		return nil, fmt.Errorf("failed to convert parameters to protobuf struct: %v", err)
	}

	runRes, err := m.client.Run(context.Background(), &pb.RunRequest{
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

// Server answers the runner's calls with an Action.
type Server struct {
	Impl Action
	log  hclog.Logger
}

// Run runs the action with the request's parameters.
func (m *Server) Run(ctx context.Context, req *pb.RunRequest) (*pb.RunResponse, error) {
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

	v, err := m.Impl.Run(withMap)
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
		if rv.Kind() == reflect.Pointer {
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

// LogParams records the parameters an action received, truncating long
// values so that a large request body does not flood the log. Actions are
// separate processes, so their log records reach the workflow runner as JSON
// on stderr and are re-filtered there by the runner's own level.
func LogParams(log hclog.Logger, msg string, with map[string]any) {
	log.Debug(msg, "params", forLog(with))
}

// LogOutcome records how an action finished. subject names what was
// attempted, for example "http request", and is used to build the message.
func LogOutcome(log hclog.Logger, subject string, ret map[string]any, err error) {
	if err != nil {
		log.Error(subject+" failed", "error", err)
		return
	}
	log.Debug(subject+" completed successfully", "result", forLog(ret))
}

// forLog returns a copy of data fit to log: credentials hidden, then long
// values truncated. The runner hides credentials in the log as well, but
// only where it finds them whole, and a truncated value is no longer whole,
// so they are hidden here first.
func forLog(data map[string]any) map[string]any {
	m := mask.New(nil, nil)
	m.Learn(data)
	return truncate.Map(m.Map(data), truncate.MaxLogLength)
}
