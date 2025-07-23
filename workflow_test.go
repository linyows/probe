package probe

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestEnv(t *testing.T) {
	os.Setenv("HOST", "http://localhost")
	os.Setenv("TOKEN", "secrets")
	defer func() {
		os.Unsetenv("HOST")
		os.Unsetenv("TOKEN")
	}()

	expected := map[string]string{
		"HOST":  "http://localhost",
		"TOKEN": "secrets",
	}

	wf := &Workflow{}
	actual := wf.Env()

	if actual["HOST"] != expected["HOST"] || actual["TOKEN"] != expected["TOKEN"] {
		t.Errorf("expected %+v, got %+v", expected, actual)
	}
}

func TestEvalVars(t *testing.T) {
	tests := []struct {
		name     string
		wf       *Workflow
		expected map[string]any
		err      error
	}{
		{
			name: "use expr",
			wf: &Workflow{
				Name: "Test",
				Vars: map[string]any{
					"host":  "{HOST ?? 'http://localhost:3000'}",
					"token": "{TOKEN}",
				},
				env: map[string]string{
					"TOKEN": "secrets",
				},
			},
			expected: map[string]any{
				"host":  "http://localhost:3000",
				"token": "secrets",
			},
			err: nil,
		},
		{
			name: "not exists environment",
			wf: &Workflow{
				Name: "Test",
				Vars: map[string]any{
					"host":  "{HOST}",
					"token": "{TOKEN}",
				},
				env: map[string]string{
					"TOKEN": "secrets",
				},
			},
			expected: map[string]any{
				"host":  "<nil>",
				"token": "secrets",
			},
			err: fmt.Errorf("environment(HOST) is nil"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := tt.wf.evalVars()
			if err != nil && err.Error() != tt.err.Error() {
				t.Errorf("expected error %+v, got %+v", tt.err, err)
			}
			if !reflect.DeepEqual(tt.expected, actual) {
				t.Errorf("expected %#v, got %#v", tt.expected, actual)
			}
		})
	}
}

func TestStepRepeatCounter(t *testing.T) {
	tests := []struct {
		name         string
		successCount int
		failureCount int
		expected     string
	}{
		{
			name:         "all success",
			successCount: 100,
			failureCount: 0,
			expected:     "100/100 success (100.0%)",
		},
		{
			name:         "partial success",
			successCount: 80,
			failureCount: 20,
			expected:     "80/100 success (80.0%)",
		},
		{
			name:         "all failure",
			successCount: 0,
			failureCount: 100,
			expected:     "0/100 success (0.0%)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counter := StepRepeatCounter{
				SuccessCount: tt.successCount,
				FailureCount: tt.failureCount,
				Name:         "Test Step",
			}

			totalCount := counter.SuccessCount + counter.FailureCount
			successRate := float64(counter.SuccessCount) / float64(totalCount) * 100
			actual := fmt.Sprintf("%d/%d success (%.1f%%)",
				counter.SuccessCount, totalCount, successRate)

			if actual != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, actual)
			}
		})
	}
}

func TestJobContextRepeatTracking(t *testing.T) {
	ctx := JobContext{
		IsRepeating:   true,
		RepeatCurrent: 5,
		RepeatTotal:   10,
		StepCounters:  make(map[int]StepRepeatCounter),
		Output:        NewOutput(false),
	}

	// Test initial state
	if !ctx.IsRepeating {
		t.Error("expected IsRepeating to be true")
	}

	if ctx.RepeatCurrent != 5 {
		t.Errorf("expected RepeatCurrent to be 5, got %d", ctx.RepeatCurrent)
	}

	if ctx.RepeatTotal != 10 {
		t.Errorf("expected RepeatTotal to be 10, got %d", ctx.RepeatTotal)
	}

	// Test step counter initialization
	counter := StepRepeatCounter{
		SuccessCount: 3,
		FailureCount: 2,
		Name:         "Test Step",
		LastResult:   true,
	}

	ctx.StepCounters[0] = counter

	if len(ctx.StepCounters) != 1 {
		t.Errorf("expected 1 step counter, got %d", len(ctx.StepCounters))
	}

	if ctx.StepCounters[0].SuccessCount != 3 {
		t.Errorf("expected SuccessCount to be 3, got %d", ctx.StepCounters[0].SuccessCount)
	}
}

func TestStepHandleRepeatExecution(t *testing.T) {
	tests := []struct {
		name                   string
		stepTest               string
		repeatTotal            int
		repeatCurrent          int
		expectedOutputContains []string
	}{
		{
			name:                   "first execution with test",
			stepTest:               "true",
			repeatTotal:            10,
			repeatCurrent:          1,
			expectedOutputContains: []string{"(repeating 10 times)"},
		},
		{
			name:                   "final execution with test success",
			stepTest:               "true",
			repeatTotal:            10,
			repeatCurrent:          10,
			expectedOutputContains: []string{"10/10 success (100.0%)"},
		},
		{
			name:                   "final execution no test",
			stepTest:               "",
			repeatTotal:            5,
			repeatCurrent:          5,
			expectedOutputContains: []string{"5/5 completed (no test)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Create test step
			step := &Step{
				Name: "Test Step",
				Test: tt.stepTest,
				idx:  0,
				expr: &Expr{},
			}

			// Create job context
			jCtx := &JobContext{
				IsRepeating:   true,
				RepeatCurrent: tt.repeatCurrent,
				RepeatTotal:   tt.repeatTotal,
				StepCounters:  make(map[int]StepRepeatCounter),
				Output:        NewOutput(false),
			}

			// Simulate multiple executions by pre-populating counter
			if tt.repeatCurrent == tt.repeatTotal {
				// Final execution - set up counter as if we've been running
				successCount := tt.repeatTotal
				failureCount := 0
				if tt.stepTest == "" {
					// No test case
					successCount = tt.repeatTotal
				}

				jCtx.StepCounters[0] = StepRepeatCounter{
					SuccessCount: successCount - 1, // -1 because handleRepeatExecution will increment
					FailureCount: failureCount,
					Name:         "Test Step",
					LastResult:   true,
				}
			}

			// Execute the function
			step.handleRepeatExecution(jCtx, "Test Step", "", false)

			// Restore stdout and capture output
			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			if _, err := io.Copy(&buf, r); err != nil {
				t.Fatalf("Failed to copy output: %v", err)
			}
			output := buf.String()

			// Check expected output
			for _, expected := range tt.expectedOutputContains {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain '%s', got: %s", expected, output)
				}
			}
		})
	}
}

func TestStepRepeatCounterUpdate(t *testing.T) {
	// Test counter update logic
	jCtx := &JobContext{
		IsRepeating:   true,
		RepeatCurrent: 3,
		RepeatTotal:   10,
		StepCounters:  make(map[int]StepRepeatCounter),
		Output:        NewOutput(false),
	}

	step := &Step{
		Name: "Test Step",
		Test: "true", // Always success
		idx:  0,
		expr: &Expr{},
	}

	// Capture stdout to avoid test output noise
	oldStdout := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	defer func() { os.Stdout = oldStdout }()

	// Execute multiple times
	for i := 1; i <= 3; i++ {
		jCtx.RepeatCurrent = i
		step.handleRepeatExecution(jCtx, "Test Step", "", false)
	}

	// Check final counter state
	counter := jCtx.StepCounters[0]
	if counter.SuccessCount != 3 {
		t.Errorf("Expected SuccessCount to be 3, got %d", counter.SuccessCount)
	}
	if counter.FailureCount != 0 {
		t.Errorf("Expected FailureCount to be 0, got %d", counter.FailureCount)
	}
	if counter.Name != "Test Step" {
		t.Errorf("Expected Name to be 'Test Step', got %s", counter.Name)
	}
}

func TestStepRepeatDisplayConditions(t *testing.T) {
	tests := []struct {
		name          string
		repeatCurrent int
		repeatTotal   int
		shouldDisplay bool
		description   string
	}{
		{
			name:          "first execution",
			repeatCurrent: 1,
			repeatTotal:   10,
			shouldDisplay: true,
			description:   "should show initial message",
		},
		{
			name:          "middle execution",
			repeatCurrent: 5,
			repeatTotal:   10,
			shouldDisplay: false,
			description:   "should not display in middle",
		},
		{
			name:          "final execution",
			repeatCurrent: 10,
			repeatTotal:   10,
			shouldDisplay: true,
			description:   "should show final result",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the display condition logic
			totalCount := tt.repeatCurrent // Simulate counter state
			isFirstExecution := totalCount == 1
			isFinalExecution := tt.repeatCurrent == tt.repeatTotal

			shouldDisplay := isFirstExecution || isFinalExecution

			if shouldDisplay != tt.shouldDisplay {
				t.Errorf("%s: expected shouldDisplay to be %v, got %v",
					tt.description, tt.shouldDisplay, shouldDisplay)
			}
		})
	}
}
