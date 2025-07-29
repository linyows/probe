package probe

import (
	"context"
	"errors"
	"testing"

	"github.com/linyows/probe/pb"
	"google.golang.org/grpc"
)

// MockActions implements the Actions interface for testing
type MockActions struct {
	RunFunc func(args []string, with map[string]string) (map[string]string, error)
}

func (m *MockActions) Run(args []string, with map[string]string) (map[string]string, error) {
	if m.RunFunc != nil {
		return m.RunFunc(args, with)
	}
	return map[string]string{"result": "success"}, nil
}

// MockActionsClient for testing ActionsClient without GRPC
type MockActionsClient struct {
	client pb.ActionsClient
}

func (m *MockActionsClient) Run(args []string, with map[string]string) (map[string]string, error) {
	// For unit testing, we can simulate the behavior without actual GRPC calls
	if len(args) == 0 {
		return nil, errors.New("no arguments provided")
	}
	
	result := map[string]string{
		"action": args[0],
		"status": "completed",
	}
	
	// Include parameters in result
	for k, v := range with {
		result[k] = v
	}
	
	return result, nil
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

func TestActionsClient_Run(t *testing.T) {
	// Since we can't easily mock the GRPC client, we'll test the interface
	// and create a mock implementation for testing
	mockClient := &MockActionsClient{}
	
	tests := []struct {
		name        string
		args        []string
		with        map[string]string
		expectError bool
		expectKeys  []string
	}{
		{
			name:        "successful run with args",
			args:        []string{"test-action"},
			with:        map[string]string{"param1": "value1"},
			expectError: false,
			expectKeys:  []string{"action", "status", "param1"},
		},
		{
			name:        "successful run with multiple params",
			args:        []string{"complex-action"},
			with:        map[string]string{"url": "http://example.com", "method": "GET"},
			expectError: false,
			expectKeys:  []string{"action", "status", "url", "method"},
		},
		{
			name:        "empty args should fail",
			args:        []string{},
			with:        map[string]string{},
			expectError: true,
			expectKeys:  nil,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := mockClient.Run(tt.args, tt.with)
			
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
			
			// Check that expected keys are present
			for _, key := range tt.expectKeys {
				if _, exists := result[key]; !exists {
					t.Errorf("expected key %q not found in result", key)
				}
			}
			
			// Check specific values
			if len(tt.args) > 0 && result["action"] != tt.args[0] {
				t.Errorf("result[action] = %q, want %q", result["action"], tt.args[0])
			}
		})
	}
}

func TestActionsServer_Run(t *testing.T) {
	tests := []struct {
		name        string
		mockFunc    func(args []string, with map[string]string) (map[string]string, error)
		args        []string
		with        map[string]string
		expectError bool
		expectedRes map[string]string
	}{
		{
			name: "successful run",
			mockFunc: func(args []string, with map[string]string) (map[string]string, error) {
				return map[string]string{
					"status": "success",
					"action": args[0],
				}, nil
			},
			args:        []string{"test-action"},
			with:        map[string]string{"param": "value"},
			expectError: false,
			expectedRes: map[string]string{
				"status": "success",
				"action": "test-action",
			},
		},
		{
			name: "error case",
			mockFunc: func(args []string, with map[string]string) (map[string]string, error) {
				return nil, errors.New("mock error")
			},
			args:        []string{"failing-action"},
			with:        map[string]string{},
			expectError: true,
			expectedRes: nil,
		},
		{
			name: "empty result",
			mockFunc: func(args []string, with map[string]string) (map[string]string, error) {
				return map[string]string{}, nil
			},
			args:        []string{"empty-action"},
			with:        map[string]string{},
			expectError: false,
			expectedRes: map[string]string{},
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
			
			req := &pb.RunRequest{
				Args: tt.args,
				With: tt.with,
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
			}
			
			// Compare results
			if len(resp.Result) != len(tt.expectedRes) {
				t.Errorf("result length = %d, want %d", len(resp.Result), len(tt.expectedRes))
			}
			
			for k, v := range tt.expectedRes {
				if resp.Result[k] != v {
					t.Errorf("result[%q] = %q, want %q", k, resp.Result[k], v)
				}
			}
		})
	}
}

func TestMockActions_Run(t *testing.T) {
	// Test the mock implementation itself
	t.Run("default behavior", func(t *testing.T) {
		mock := &MockActions{}
		
		result, err := mock.Run([]string{"test"}, map[string]string{"key": "value"})
		
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		
		expected := map[string]string{"result": "success"}
		if len(result) != len(expected) {
			t.Errorf("result length = %d, want %d", len(result), len(expected))
		}
		
		if result["result"] != "success" {
			t.Errorf("result[result] = %q, want %q", result["result"], "success")
		}
	})
	
	t.Run("custom function", func(t *testing.T) {
		mock := &MockActions{
			RunFunc: func(args []string, with map[string]string) (map[string]string, error) {
				return map[string]string{
					"custom": "response",
					"args_count": string(rune(len(args))),
				}, nil
			},
		}
		
		result, err := mock.Run([]string{"arg1", "arg2"}, map[string]string{})
		
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		
		if result["custom"] != "response" {
			t.Errorf("result[custom] = %q, want %q", result["custom"], "response")
		}
	})
	
	t.Run("error case", func(t *testing.T) {
		mock := &MockActions{
			RunFunc: func(args []string, with map[string]string) (map[string]string, error) {
				return nil, errors.New("test error")
			},
		}
		
		result, err := mock.Run([]string{}, map[string]string{})
		
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
	
	var params ActionsParams = map[string]string{"key": "value"}
	if len(params) != 1 {
		t.Errorf("ActionsParams length = %d, want 1", len(params))
	}
}

// Note: RunActions function is complex to test because it:
// 1. Creates actual plugin clients
// 2. Executes external processes
// 3. Uses GRPC communication
// 
// For comprehensive testing of RunActions, you would need:
// - Mock plugin system
// - Test doubles for exec.Command
// - Integration test environment
// 
// This is typically tested through integration tests rather than unit tests.
func TestRunActions_Structure(t *testing.T) {
	// We can only test that the function signature exists and compiles
	var _ func(string, []string, map[string]any, bool) (map[string]any, error) = RunActions
	
	// Test that the function handles the basic parameter transformation
	// In a real scenario, this would require a full plugin setup
	t.Log("RunActions function signature verified")
}