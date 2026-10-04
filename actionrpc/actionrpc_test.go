package actionrpc

import (
	"context"
	"errors"
	"testing"

	"github.com/linyows/probe/pb"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"
)

// MockActions implements the Action interface for testing
type MockActions struct {
	RunFunc func(with map[string]any) (map[string]any, error)
}

func (m *MockActions) Run(with map[string]any) (map[string]any, error) {
	if m.RunFunc != nil {
		return m.RunFunc(with)
	}
	return map[string]any{"result": "success"}, nil
}

func TestProtocolConstants(t *testing.T) {
	// Test that the basic types and constants are defined correctly
	if BuiltinCmd == "" {
		t.Error("BuiltinCmd should not be empty")
	}

	expectedBuiltinCmd := "builtin-actions"
	if BuiltinCmd != expectedBuiltinCmd {
		t.Errorf("BuiltinCmd = %q, want %q", BuiltinCmd, expectedBuiltinCmd)
	}

	// Test HandshakeConfig
	if Handshake.ProtocolVersion != 1 {
		t.Errorf("Handshake.ProtocolVersion = %d, want 1", Handshake.ProtocolVersion)
	}

	if Handshake.MagicCookieKey != "probe" {
		t.Errorf("Handshake.MagicCookieKey = %q, want %q", Handshake.MagicCookieKey, "probe")
	}

	if Handshake.MagicCookieValue != "actions" {
		t.Errorf("Handshake.MagicCookieValue = %q, want %q", Handshake.MagicCookieValue, "actions")
	}

	// Test PluginMap
	if len(PluginMap) != 1 {
		t.Errorf("PluginMap length = %d, want 1", len(PluginMap))
	}

	if _, exists := PluginMap["actions"]; !exists {
		t.Error("PluginMap should contain 'actions' key")
	}
}

func TestPlugin_GRPCServer(t *testing.T) {
	plugin := &Plugin{
		Impl: &MockActions{},
	}

	// Create a valid gRPC server for testing
	server := grpc.NewServer()
	defer server.Stop()

	// Test that GRPCServer method exists and returns nil for valid input
	err := plugin.GRPCServer(nil, server)
	if err != nil {
		t.Errorf("GRPCServer() returned error: %v", err)
	}
}

func TestPlugin_GRPCClient(t *testing.T) {
	plugin := &Plugin{}

	// Test that GRPCClient method exists and can be called
	// We can't easily test the actual GRPC client creation without complex setup
	client, err := plugin.GRPCClient(context.Background(), nil, nil)
	if err != nil {
		t.Errorf("GRPCClient() returned error: %v", err)
	}

	// Verify the client is of the expected type
	if _, ok := client.(*Client); !ok {
		t.Errorf("GRPCClient() returned wrong type: %T", client)
	}
}

func TestServer_Run(t *testing.T) {
	tests := []struct {
		name        string
		mockFunc    func(with map[string]any) (map[string]any, error)
		with        map[string]any
		expectError bool
		expectedRes map[string]any
	}{
		{
			name: "successful run",
			mockFunc: func(with map[string]any) (map[string]any, error) {
				return map[string]any{
					"status": "success",
					"action": with["action"],
				}, nil
			},
			with:        map[string]any{"action": "test-action", "param": "value"},
			expectError: false,
			expectedRes: map[string]any{
				"status": "success",
				"action": "test-action",
			},
		},
		{
			name: "error case",
			mockFunc: func(with map[string]any) (map[string]any, error) {
				return nil, errors.New("mock error")
			},
			with:        map[string]any{},
			expectError: true,
			expectedRes: nil,
		},
		{
			name: "empty result",
			mockFunc: func(with map[string]any) (map[string]any, error) {
				return map[string]any{}, nil
			},
			with:        map[string]any{},
			expectError: false,
			expectedRes: map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockActions := &MockActions{
				RunFunc: tt.mockFunc,
			}

			server := &Server{
				Impl: mockActions,
			}

			// Convert with to structpb.Struct
			withStruct, err := structpb.NewStruct(tt.with)
			if err != nil {
				t.Fatalf("Failed to convert with to struct: %v", err)
			}

			req := &pb.RunRequest{
				With: withStruct,
			}

			resp, err := server.Run(context.Background(), req)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if resp == nil {
				t.Fatal("response should not be nil")
				return
			}

			// Convert result back to map for comparison
			resultMap := resp.Result.AsMap()

			// Compare results
			if len(resultMap) != len(tt.expectedRes) {
				t.Errorf("result length = %d, want %d", len(resultMap), len(tt.expectedRes))
			}

			for k, v := range tt.expectedRes {
				if resultMap[k] != v {
					t.Errorf("result[%q] = %v, want %v", k, resultMap[k], v)
				}
			}
		})
	}
}

func TestMockActions_Run(t *testing.T) {
	// Test the mock implementation itself
	t.Run("default behavior", func(t *testing.T) {
		mock := &MockActions{}

		result, err := mock.Run(map[string]any{"key": "value"})

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		expected := map[string]any{"result": "success"}
		if len(result) != len(expected) {
			t.Errorf("result length = %d, want %d", len(result), len(expected))
		}

		if result["result"] != "success" {
			t.Errorf("result[result] = %q, want %q", result["result"], "success")
		}
	})

	t.Run("custom function", func(t *testing.T) {
		mock := &MockActions{
			RunFunc: func(with map[string]any) (map[string]any, error) {
				return map[string]any{
					"custom":    "response",
					"with_size": len(with),
				}, nil
			},
		}

		result, err := mock.Run(map[string]any{"a": 1, "b": 2})

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if result["custom"] != "response" {
			t.Errorf("result[custom] = %q, want %q", result["custom"], "response")
		}
	})

	t.Run("error case", func(t *testing.T) {
		mock := &MockActions{
			RunFunc: func(with map[string]any) (map[string]any, error) {
				return nil, errors.New("test error")
			},
		}

		result, err := mock.Run(map[string]any{})

		if err == nil {
			t.Error("expected error but got none")
		}

		if result != nil {
			t.Errorf("result should be nil on error, got %v", result)
		}
	})
}

// Test type definitions and interfaces
func TestActionInterface(t *testing.T) {
	// Test that MockActions implements Action interface
	var _ Action = &MockActions{}

	// Test that Client implements Action interface
	var _ Action = &Client{}

}

func TestConvertFloatToInt(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected any
	}{
		{"nil value", nil, nil},
		{"float64 integer value", float64(1767851301), int64(1767851301)},
		{"float64 with decimal", float64(0.091026392), float64(0.091026392)},
		{"float64 zero", float64(0), int64(0)},
		{"float64 negative integer", float64(-12345), int64(-12345)},
		{"string unchanged", "test", "test"},
		{"bool unchanged", true, true},
		{"int64 unchanged", int64(12345), int64(12345)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertFloatToInt(tt.input)
			if result != tt.expected {
				t.Errorf("convertFloatToInt(%v) = %v (%T), want %v (%T)",
					tt.input, result, result, tt.expected, tt.expected)
			}
		})
	}
}

func TestConvertFloatToInt_Map(t *testing.T) {
	input := map[string]any{
		"time":  float64(1767851301),
		"value": float64(0.091),
		"name":  "test",
	}

	result := convertFloatToInt(input).(map[string]any)

	if result["time"] != int64(1767851301) {
		t.Errorf("time = %v (%T), want int64(1767851301)", result["time"], result["time"])
	}
	if result["value"] != float64(0.091) {
		t.Errorf("value = %v (%T), want float64(0.091)", result["value"], result["value"])
	}
	if result["name"] != "test" {
		t.Errorf("name = %v, want test", result["name"])
	}
}

func TestConvertFloatToInt_Array(t *testing.T) {
	input := []any{
		map[string]any{
			"name":  "api.response.time",
			"time":  float64(1767851301),
			"value": float64(0.091026392),
		},
	}

	result := convertFloatToInt(input).([]any)
	m := result[0].(map[string]any)

	if m["time"] != int64(1767851301) {
		t.Errorf("time = %v (%T), want int64(1767851301)", m["time"], m["time"])
	}
	if m["value"] != float64(0.091026392) {
		t.Errorf("value = %v (%T), want float64(0.091026392)", m["value"], m["value"])
	}
}
