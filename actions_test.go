package probe

import (
	"errors"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/linyows/probe/expr"
)

func TestMockActionRunner(t *testing.T) {
	mock := NewMockActionRunner()

	// Test default behavior
	result, err := mock.RunActions("test", map[string]any{"key": "value"}, RunOptions{})
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

	result, err = mock.RunActions("http", map[string]any{}, RunOptions{})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result["code"] != 0 {
		t.Errorf("Expected code=0, got %v", result["code"])
	}

	// Test error behavior
	testErr := errors.New("test error")
	mock.SetError("failing-action", testErr)

	result, err = mock.RunActions("failing-action", map[string]any{}, RunOptions{})
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
	step.Expr = &expr.Expr{}
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

func TestRunOptions_logLevel(t *testing.T) {
	tests := []struct {
		name string
		opts RunOptions
		want hclog.Level
	}{
		{
			name: "default filters an action's debug and info records",
			opts: RunOptions{},
			want: hclog.Warn,
		},
		{
			name: "verbose lets everything through",
			opts: RunOptions{Verbose: true},
			want: hclog.Debug,
		},
		{
			name: "quiet silences an attempt that will be retried",
			opts: RunOptions{Quiet: true},
			want: hclog.Off,
		},
		{
			name: "verbose wins over quiet",
			opts: RunOptions{Verbose: true, Quiet: true},
			want: hclog.Debug,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.opts.logLevel(); got != tt.want {
				t.Errorf("RunOptions%+v.logLevel() = %v, want %v", tt.opts, got, tt.want)
			}
		})
	}
}
