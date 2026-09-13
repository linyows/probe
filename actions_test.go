package probe

import (
	"context"
	"errors"
	"testing"

	"github.com/linyows/probe/v2/pb"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"
)

// MockActions implements the Actions interface for testing
type MockActions struct {
	RunFunc func(with map[string]any) (map[string]any, error)
}

func (m *MockActions) Run(with map[string]any) (map[string]any, error) {
	if m.RunFunc != nil {
		return m.RunFunc(with)
	}
	return map[string]any{"result": "success"}, nil
}

func TestActionsTypes(t *testing.T) {
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

func TestActionsPlugin_GRPCServer(t *testing.T) {
	plugin := &ActionsPlugin{
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

func TestActionsPlugin_GRPCClient(t *testing.T) {
	plugin := &ActionsPlugin{}

	// Test that GRPCClient method exists and can be called
	// We can't easily test the actual GRPC client creation without complex setup
	client, err := plugin.GRPCClient(context.Background(), nil, nil)
	if err != nil {
		t.Errorf("GRPCClient() returned error: %v", err)
	}

	// Verify the client is of the expected type
	if _, ok := client.(*ActionsClient); !ok {
		t.Errorf("GRPCClient() returned wrong type: %T", client)
	}
}

func TestActionsServer_Run(t *testing.T) {
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

			server := &ActionsServer{
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
func TestActionsInterface(t *testing.T) {
	// Test that MockActions implements Actions interface
	var _ Actions = &MockActions{}

	// Test that ActionsClient implements Actions interface
	var _ Actions = &ActionsClient{}

	// Test type aliases
	var args ActionsArgs = []string{"test"}
	if len(args) != 1 {
		t.Errorf("ActionsArgs length = %d, want 1", len(args))
	}

	var params ActionsParams = map[string]any{"key": "value"}
	if len(params) != 1 {
		t.Errorf("ActionsParams length = %d, want 1", len(params))
	}
}

func TestMockActionRunner(t *testing.T) {
	mock := NewMockActionRunner()

	// Test default behavior
	result, err := mock.RunActions("test", map[string]any{"key": "value"}, false)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result["mock"] != true {
		t.Errorf("Expected mock=true in default result")
	}
	if result["action"] != "test" {
		t.Errorf("Expected action='test', got %v", result["action"])
	}

	// Test custom result
	customResult := map[string]any{
		"code":    0,
		"results": map[string]any{"text": "Hello World"},
	}
	mock.SetResult("http", customResult)

	result, err = mock.RunActions("http", map[string]any{}, false)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result["code"] != 0 {
		t.Errorf("Expected code=0, got %v", result["code"])
	}

	// Test error behavior
	testErr := errors.New("test error")
	mock.SetError("failing-action", testErr)

	result, err = mock.RunActions("failing-action", map[string]any{}, false)
	if err != testErr {
		t.Errorf("Expected test error, got %v", err)
	}
	if result != nil {
		t.Errorf("Expected nil result on error, got %v", result)
	}
}

func TestStepWithMockRunner(t *testing.T) {
	// Create a step with mock runner
	step := &Step{
		Name: "Test Step",
		Uses: "http",
		With: map[string]any{"url": "http://example.com"},
	}

	// Set up mock runner
	mock := NewMockActionRunner()
	mock.SetResult("http", map[string]any{
		"code": 0,
		"results": map[string]any{
			"status": 200,
			"body":   "OK",
		},
	})

	// Set the mock runner directly on the step
	step.actionRunner = mock

	// Create minimal job context for testing
	jCtx := &JobContext{
		Config: Config{Verbose: false},
	}

	// Initialize expression evaluator
	step.Expr = &Expr{}
	step.ctx = StepContext{}

	// Execute action
	result, err := step.executeAction("Test Step", jCtx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result["code"] != 0 {
		t.Errorf("Expected code=0, got %v", result["code"])
	}

	results, ok := result["results"].(map[string]any)
	if !ok {
		t.Errorf("Expected results to be map[string]any")
	} else {
		if results["status"] != 200 {
			t.Errorf("Expected status=200, got %v", results["status"])
		}
	}
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
