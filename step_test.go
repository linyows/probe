package probe

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestStep_parseWaitDuration(t *testing.T) {
	tests := []struct {
		name     string
		wait     string
		expected time.Duration
		hasError bool
	}{
		{
			name:     "seconds as integer string",
			wait:     "5",
			expected: 5 * time.Second,
			hasError: false,
		},
		{
			name:     "duration string with seconds",
			wait:     "3s",
			expected: 3 * time.Second,
			hasError: false,
		},
		{
			name:     "duration string with milliseconds",
			wait:     "500ms",
			expected: 500 * time.Millisecond,
			hasError: false,
		},
		{
			name:     "duration string with minutes",
			wait:     "2m",
			expected: 2 * time.Minute,
			hasError: false,
		},
		{
			name:     "invalid format",
			wait:     "invalid",
			expected: 0,
			hasError: true,
		},
		{
			name:     "empty string",
			wait:     "",
			expected: 0,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := &Step{}
			duration, err := step.parseWaitDuration(tt.wait)

			if tt.hasError {
				if err == nil {
					t.Errorf("expected error for wait '%s', but got none", tt.wait)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for wait '%s': %v", tt.wait, err)
				}
				if duration != tt.expected {
					t.Errorf("expected %v, got %v", tt.expected, duration)
				}
			}
		})
	}
}

func TestStep_formatWaitTime(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "exact seconds",
			duration: 5 * time.Second,
			expected: "5s",
		},
		{
			name:     "milliseconds",
			duration: 500 * time.Millisecond,
			expected: "500ms",
		},
		{
			name:     "mixed duration",
			duration: 2*time.Minute + 30*time.Second,
			expected: "150s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := &Step{}
			result := step.formatWaitTime(tt.duration)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestStep_getWaitTimeForDisplay(t *testing.T) {
	tests := []struct {
		name     string
		wait     string
		expected string
	}{
		{
			name:     "no wait",
			wait:     "",
			expected: "",
		},
		{
			name:     "seconds",
			wait:     "5",
			expected: "5s",
		},
		{
			name:     "duration string",
			wait:     "500ms",
			expected: "500ms",
		},
		{
			name:     "invalid format returns empty",
			wait:     "invalid",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := &Step{Wait: tt.wait}
			result := step.getWaitTimeForDisplay()
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestStep_handleWait(t *testing.T) {
	tests := []struct {
		name         string
		wait         string
		expectedTime string
		minDuration  time.Duration
		maxDuration  time.Duration
	}{
		{
			name:         "no wait",
			wait:         "",
			expectedTime: "",
			minDuration:  0,
			maxDuration:  10 * time.Millisecond,
		},
		{
			name:         "short wait",
			wait:         "10ms",
			expectedTime: "10ms",
			minDuration:  8 * time.Millisecond,
			maxDuration:  50 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := &Step{Wait: tt.wait}
			jCtx := &JobContext{
				Printer: &Printer{},
			}

			start := time.Now()
			step.handleWait(jCtx)
			duration := time.Since(start)

			// Skip expectedTime check for empty wait case
			if tt.wait != "" {
				expectedDuration, _ := time.ParseDuration(tt.expectedTime)
				if duration < expectedDuration {
					t.Errorf("expected at least %s, got %s", expectedDuration, duration)
				}
			}

			if duration < tt.minDuration {
				t.Errorf("Duration %v should be at least %v", duration, tt.minDuration)
			}

			if duration > tt.maxDuration {
				t.Errorf("Duration %v should not exceed %v", duration, tt.maxDuration)
			}
		})
	}
}

func TestStep_SkipIfWithWaitTiming(t *testing.T) {
	tests := []struct {
		name        string
		wait        string
		skipif      string
		expectSkip  bool
		maxDuration time.Duration
		minDuration time.Duration
	}{
		{
			name:        "skipped step with wait should not wait",
			wait:        "1s",
			skipif:      "true",
			expectSkip:  true,
			maxDuration: 50 * time.Millisecond, // Should be very fast since no wait
			minDuration: 0,
		},
		{
			name:        "non-skipped step with wait should wait",
			wait:        "100ms",
			skipif:      "false",
			expectSkip:  false,
			maxDuration: 200 * time.Millisecond, // Allow some overhead
			minDuration: 80 * time.Millisecond,  // Should wait at least this long
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := &Step{
				Name:   "Test Step",
				Uses:   "hello",
				Wait:   tt.wait,
				SkipIf: tt.skipif,
				Expr:   &Expr{},
			}

			jCtx := &JobContext{
				Printer: NewPrinter(false, []string{}), // Silent printer
				Config:  Config{},
			}

			start := time.Now()
			name, shouldContinue := step.prepare(jCtx)
			duration := time.Since(start)

			// Check skip behavior
			if tt.expectSkip && shouldContinue {
				t.Errorf("Expected step to be skipped (shouldContinue=false), got shouldContinue=%v", shouldContinue)
			}
			if !tt.expectSkip && !shouldContinue {
				t.Errorf("Expected step to continue (shouldContinue=true), got shouldContinue=%v", shouldContinue)
			}

			// Check timing
			if duration > tt.maxDuration {
				t.Errorf("Expected duration <= %v, got %v", tt.maxDuration, duration)
			}
			if duration < tt.minDuration {
				t.Errorf("Expected duration >= %v, got %v", tt.minDuration, duration)
			}

			if name != "Test Step" {
				t.Errorf("Expected step name 'Test Step', got %v", name)
			}
		})
	}
}

func TestStep_shouldSkip(t *testing.T) {
	tests := []struct {
		name     string
		skipif   string
		context  StepContext
		expected bool
		hasError bool
	}{
		{
			name:     "no skipif condition",
			skipif:   "",
			context:  StepContext{},
			expected: false,
			hasError: false,
		},
		{
			name:   "skipif returns true",
			skipif: "true",
			context: StepContext{
				Vars: map[string]any{},
			},
			expected: true,
			hasError: false,
		},
		{
			name:   "skipif returns false",
			skipif: "false",
			context: StepContext{
				Vars: map[string]any{},
			},
			expected: false,
			hasError: false,
		},
		{
			name:   "skipif with variable condition - skip",
			skipif: `vars.env == "production"`,
			context: StepContext{
				Vars: map[string]any{
					"env": "production",
				},
			},
			expected: true,
			hasError: false,
		},
		{
			name:   "skipif with variable condition - don't skip",
			skipif: `vars.env == "production"`,
			context: StepContext{
				Vars: map[string]any{
					"env": "development",
				},
			},
			expected: false,
			hasError: false,
		},
		{
			name:   "skipif with empty variable condition",
			skipif: `vars.url == ""`,
			context: StepContext{
				Vars: map[string]any{
					"url": "",
				},
			},
			expected: true,
			hasError: false,
		},
		{
			name:   "skipif with contains condition",
			skipif: `vars.url contains "production"`,
			context: StepContext{
				Vars: map[string]any{
					"url": "https://production.example.com",
				},
			},
			expected: true,
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := &Step{
				SkipIf: tt.skipif,
				ctx:    tt.context,
				Expr:   &Expr{},
			}
			jCtx := &JobContext{
				Printer: &Printer{},
			}

			result := step.shouldSkip(jCtx)
			if result != tt.expected {
				t.Errorf("shouldSkip() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestStep_createSkippedStepResult(t *testing.T) {
	step := &Step{
		Idx:  1,
		Wait: "2s",
	}
	jCtx := &JobContext{}
	name := "Test Step"

	result := step.createSkippedStepResult(name, jCtx, nil)

	if result.Index != 1 {
		t.Errorf("Index = %v, want %v", result.Index, 1)
	}
	if result.Name != "Test Step (SKIPPED)" {
		t.Errorf("Name = %v, want %v", result.Name, "Test Step (SKIPPED)")
	}
	if result.Status != StatusSkipped {
		t.Errorf("Status = %v, want %v", result.Status, StatusSkipped)
	}
	if result.WaitTime != "2s" {
		t.Errorf("WaitTime = %v, want %v", result.WaitTime, "2s")
	}
	if result.HasTest != false {
		t.Errorf("HasTest = %v, want %v", result.HasTest, false)
	}
	if result.RepeatCounter != nil {
		t.Errorf("RepeatCounter should be nil for non-repeat step, got %v", result.RepeatCounter)
	}
}

func TestStep_createSkippedStepResult_WithRepeatCounter(t *testing.T) {
	step := &Step{
		Idx:  1,
		Wait: "2s",
	}
	jCtx := &JobContext{}
	name := "Test Step"
	counter := &StepRepeatCounter{
		SuccessCount: 5,
		FailureCount: 2,
		Name:         "Test Counter",
		LastResult:   true,
	}

	result := step.createSkippedStepResult(name, jCtx, counter)

	if result.RepeatCounter == nil {
		t.Error("RepeatCounter should not be nil when provided")
	}
	if result.RepeatCounter != counter {
		t.Error("RepeatCounter should be the same instance that was passed")
	}
	if result.RepeatCounter.SuccessCount != 5 {
		t.Errorf("RepeatCounter.SuccessCount = %v, want %v", result.RepeatCounter.SuccessCount, 5)
	}
}

func TestStep_createStepResult_WithRepeatCounter(t *testing.T) {
	step := &Step{
		Idx:  1,
		Test: "res.status == 200",
		Echo: "Hello World",
		Wait: "1s",
		Expr: &Expr{},
		ctx: StepContext{
			Res: map[string]any{"status": 200},
			RT:  "250ms",
		},
	}
	jCtx := &JobContext{
		Config:  Config{RT: true},
		Printer: NewPrinter(false, []string{}),
	}
	name := "Test Step"
	counter := &StepRepeatCounter{
		SuccessCount: 3,
		FailureCount: 1,
		Name:         "Test Counter",
		LastResult:   true,
	}

	result := step.createStepResult(name, jCtx, counter)

	if result.RepeatCounter == nil {
		t.Error("RepeatCounter should not be nil when provided")
	}
	if result.RepeatCounter != counter {
		t.Error("RepeatCounter should be the same instance that was passed")
	}
	if result.RepeatCounter.SuccessCount != 3 {
		t.Errorf("RepeatCounter.SuccessCount = %v, want %v", result.RepeatCounter.SuccessCount, 3)
	}
	if result.RT != "250ms" {
		t.Errorf("RT = %v, want %v", result.RT, "250ms")
	}
}

func TestStep_createFailedStepResult(t *testing.T) {
	testErr := fmt.Errorf("test error message")
	step := &Step{
		Idx:  2,
		Test: "res.code == 200",
		Wait: "3s",
		err:  testErr,
		ctx: StepContext{
			RT:  "500ms",
			Res: map[string]any{"report": "HTTP error occurred"},
		},
	}
	jCtx := &JobContext{
		Config: Config{RT: true},
	}
	name := "Failed Step"

	result := step.createFailedStepResult(name, jCtx, nil)

	if result.Index != 2 {
		t.Errorf("Index = %v, want %v", result.Index, 2)
	}
	if result.Name != "Failed Step" {
		t.Errorf("Name = %v, want %v", result.Name, "Failed Step")
	}
	if result.Status != StatusError {
		t.Errorf("Status = %v, want %v", result.Status, StatusError)
	}
	if result.WaitTime != "3s" {
		t.Errorf("WaitTime = %v, want %v", result.WaitTime, "3s")
	}
	if result.HasTest != true {
		t.Errorf("HasTest = %v, want %v", result.HasTest, true)
	}
	if result.RT != "500ms" {
		t.Errorf("RT = %v, want %v", result.RT, "500ms")
	}
	if result.Report != "HTTP error occurred" {
		t.Errorf("Report = %v, want %v", result.Report, "HTTP error occurred")
	}
	if result.TestOutput != "test error message" {
		t.Errorf("TestOutput = %v, want %v", result.TestOutput, "test error message")
	}
	if result.RepeatCounter != nil {
		t.Errorf("RepeatCounter should be nil for non-repeat step, got %v", result.RepeatCounter)
	}
}

func TestStep_createFailedStepResult_WithRepeatCounter(t *testing.T) {
	testErr := fmt.Errorf("connection timeout")
	step := &Step{
		Idx:  3,
		Test: "res.status < 400",
		Wait: "1s",
		err:  testErr,
		ctx: StepContext{
			RT: "10s",
		},
	}
	jCtx := &JobContext{
		Config: Config{RT: true},
	}
	name := "Timeout Step"
	counter := &StepRepeatCounter{
		SuccessCount: 1,
		FailureCount: 4,
		Name:         "Timeout Counter",
		LastResult:   false,
	}

	result := step.createFailedStepResult(name, jCtx, counter)

	if result.RepeatCounter == nil {
		t.Error("RepeatCounter should not be nil when provided")
	}
	if result.RepeatCounter != counter {
		t.Error("RepeatCounter should be the same instance that was passed")
	}
	if result.RepeatCounter.FailureCount != 4 {
		t.Errorf("RepeatCounter.FailureCount = %v, want %v", result.RepeatCounter.FailureCount, 4)
	}
	if result.Status != StatusError {
		t.Errorf("Status = %v, want %v", result.Status, StatusError)
	}
	if result.TestOutput != "connection timeout" {
		t.Errorf("TestOutput = %v, want %v", result.TestOutput, "connection timeout")
	}
}

func TestStep_createFailedStepResult_NoTest(t *testing.T) {
	step := &Step{
		Idx: 1,
		// No Test field
		ctx: StepContext{},
	}
	jCtx := &JobContext{}
	name := "No Test Step"

	result := step.createFailedStepResult(name, jCtx, nil)

	if result.HasTest != false {
		t.Errorf("HasTest = %v, want %v", result.HasTest, false)
	}
	if result.TestOutput != "" {
		t.Errorf("TestOutput = %v, want empty string", result.TestOutput)
	}
}

// Tests for refactored methods

func TestStep_prepare(t *testing.T) {
	tests := []struct {
		name           string
		stepName       string
		wait           string
		skipif         string
		skipCondition  map[string]any
		expectName     string
		expectContinue bool
		expectError    bool
	}{
		{
			name:           "normal step preparation",
			stepName:       "Test Step",
			wait:           "",
			skipif:         "",
			expectName:     "Test Step",
			expectContinue: true,
			expectError:    false,
		},
		{
			name:           "empty name gets default",
			stepName:       "",
			wait:           "",
			skipif:         "",
			expectName:     "Unknown Step",
			expectContinue: true,
			expectError:    false,
		},
		{
			name:           "step with wait",
			stepName:       "Wait Step",
			wait:           "10ms",
			skipif:         "",
			expectName:     "Wait Step",
			expectContinue: true,
			expectError:    false,
		},
		{
			name:           "skipped step",
			stepName:       "Skipped Step",
			wait:           "",
			skipif:         "true",
			expectName:     "Skipped Step",
			expectContinue: false,
			expectError:    false,
		},
		{
			name:           "step with conditional skip - skip",
			stepName:       "Conditional Step",
			wait:           "",
			skipif:         `vars.env == "test"`,
			skipCondition:  map[string]any{"env": "test"},
			expectName:     "Conditional Step",
			expectContinue: false,
			expectError:    false,
		},
		{
			name:           "step with conditional skip - continue",
			stepName:       "Conditional Step",
			wait:           "",
			skipif:         `vars.env == "test"`,
			skipCondition:  map[string]any{"env": "prod"},
			expectName:     "Conditional Step",
			expectContinue: true,
			expectError:    false,
		},
		{
			name:           "skipped step with wait should not wait",
			stepName:       "Skipped Wait Step",
			wait:           "2s",
			skipif:         "true",
			expectName:     "Skipped Wait Step",
			expectContinue: false,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := &Step{
				Name:   tt.stepName,
				Wait:   tt.wait,
				SkipIf: tt.skipif,
				Expr:   &Expr{},
			}

			// Set up context
			vars := tt.skipCondition
			if vars == nil {
				vars = map[string]any{}
			}
			step.ctx = StepContext{
				Vars: vars,
			}

			jCtx := &JobContext{
				Printer: NewPrinter(false, []string{}), // Use silent printer to avoid output during tests
			}

			name, shouldContinue := step.prepare(jCtx)

			if name != tt.expectName {
				t.Errorf("prepare() name = %v, want %v", name, tt.expectName)
			}
			if shouldContinue != tt.expectContinue {
				t.Errorf("prepare() shouldContinue = %v, want %v", shouldContinue, tt.expectContinue)
			}
		})
	}
}

func TestStep_executeAction(t *testing.T) {
	t.Run("method signature verification only", func(t *testing.T) {
		// This test only verifies that the executeAction method exists with the correct signature.
		// We cannot test actual execution due to plugin system complexity and timeouts.
		// Instead, we verify the method signature by reflection or compilation.

		step := &Step{
			Uses: "test-action",
			With: map[string]any{},
			Expr: &Expr{},
		}

		// Verify method signature exists (this will compile successfully if signature is correct)
		var methodFunc func(string, *JobContext) (map[string]any, error) = step.executeAction

		// We don't call the method to avoid plugin system timeout
		_ = methodFunc

		t.Logf("executeAction() method signature verified successfully")
	})
}

func TestStep_processActionResult(t *testing.T) {
	tests := []struct {
		name         string
		actionResult map[string]any
		expectLogs   int
	}{
		{
			name: "normal action result",
			actionResult: map[string]any{
				"req": map[string]any{"url": "http://example.com"},
				"res": map[string]any{"status": 200, "body": "response"},
				"rt":  "100ms",
			},
			expectLogs: 1,
		},
		{
			name: "action result with JSON body",
			actionResult: map[string]any{
				"req": map[string]any{"url": "http://example.com"},
				"res": map[string]any{
					"status": 200,
					"body":   `{"message": "success"}`,
				},
				"rt": "150ms",
			},
			expectLogs: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := &Step{
				Expr: &Expr{},
			}

			jCtx := &JobContext{}

			step.processActionResult(tt.actionResult, jCtx)
		})
	}
}

func TestStep_handleActionError(t *testing.T) {
	step := &Step{
		Uses: "mock-action-for-error-test",
		Expr: &Expr{},
	}

	jCtx := &JobContext{
		Printer: NewPrinter(false, []string{}),
	}

	originalErr := fmt.Errorf("test error")
	step.handleActionError(originalErr, "test-step", jCtx)

	// Check that error was set
	if step.err == nil {
		t.Errorf("handleActionError() should set step.err")
	}

	// Check that job was marked as failed
	if !jCtx.Failed {
		t.Errorf("handleActionError() should mark job as failed")
	}

	// Check error details
	if probeErr, ok := step.err.(*ProbeError); ok {
		if probeErr.Context["step_name"] != "test-step" {
			t.Errorf("handleActionError() should set step_name context")
		}
		if probeErr.Context["action_type"] != "mock-action-for-error-test" {
			t.Errorf("handleActionError() should set action_type context")
		}
	} else {
		t.Errorf("handleActionError() should create ProbeError")
	}
}

func TestStep_finalize(t *testing.T) {
	tests := []struct {
		name         string
		verbose      bool
		isRepeating  bool
		actionResult map[string]any
		expectCalled string // "verbose", "repeat", or "standard"
	}{
		{
			name:        "verbose mode",
			verbose:     true,
			isRepeating: false,
			actionResult: map[string]any{
				"req": map[string]any{"url": "http://example.com"},
				"res": map[string]any{"status": 200},
				"rt":  "100ms",
			},
			expectCalled: "verbose",
		},
		{
			name:        "repeat mode",
			verbose:     false,
			isRepeating: true,
			actionResult: map[string]any{
				"req": map[string]any{"url": "http://example.com"},
				"res": map[string]any{"status": 200},
				"rt":  "100ms",
			},
			expectCalled: "repeat",
		},
		{
			name:        "standard mode",
			verbose:     false,
			isRepeating: false,
			actionResult: map[string]any{
				"req": map[string]any{"url": "http://example.com"},
				"res": map[string]any{"status": 200},
				"rt":  "100ms",
			},
			expectCalled: "standard",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := &Step{
				Expr: &Expr{},
			}

			jCtx := &JobContext{
				Config: Config{
					Verbose: tt.verbose,
				},
				IsRepeating:  tt.isRepeating,
				Printer:      NewPrinter(false, []string{}),
				StepCounters: make(map[int]StepRepeatCounter),
			}

			// Note: This test mainly verifies that the correct code path is taken
			// More detailed testing of each path should be done in their specific test functions
			step.finalize("test-step", tt.actionResult, jCtx)

			// For now, we just verify the method doesn't panic
			// In a more complete test, we might track which methods were called
		})
	}
}

// Integration test for the refactored Do() method
func TestStep_Do_Integration(t *testing.T) {
	t.Run("step with skip condition", func(t *testing.T) {
		step := Step{
			Name:   "Skipped Step",
			Uses:   "", // Empty action since this step should be skipped anyway
			SkipIf: "true",
			Expr:   &Expr{},
		}

		step.ctx = StepContext{
			Vars: map[string]any{},
		}

		jobContext := JobContext{
			Printer: NewPrinter(false, []string{}),
		}

		// Execute the step (should be skipped)
		step.Do(&jobContext)

		// Since step is skipped, job should not fail
		if jobContext.Failed {
			t.Errorf("Do() with skip condition should not fail the job")
		}
	})

	t.Run("refactored method structure works", func(t *testing.T) {
		// This test verifies that the refactored Do() method calls prepare() correctly
		// We avoid the executeAction() call by using a skip condition
		step := Step{
			Name:   "Test Step",
			Uses:   "any-action", // This won't be executed due to skip
			SkipIf: "true",       // Always skip to avoid plugin execution
			Expr:   &Expr{},
		}

		step.ctx = StepContext{
			Vars: map[string]any{},
		}

		jobContext := JobContext{
			Config:  Config{Verbose: false},
			Printer: NewPrinter(false, []string{}),
		}

		// The Do() method should execute without panicking
		// Since skip=true, it should only call prepare() and return early
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Do() method panicked: %v", r)
			}
		}()

		step.Do(&jobContext)

		// Verify the step was handled correctly (should be skipped)
		if jobContext.Failed {
			t.Errorf("Do() with skip condition should not fail the job")
		}

		t.Logf("Do() method executed successfully with refactored structure (skipped execution)")
	})
}

func TestSleepWithMessage(t *testing.T) {
	tests := []struct {
		name           string
		duration       time.Duration
		message        string
		expectFnCalled bool
		minCalls       int
		maxCalls       int
	}{
		{
			name:           "Short duration under 1s - no call",
			duration:       500 * time.Millisecond,
			message:        "ignored",
			expectFnCalled: false,
		},
		{
			name:           "2.5s duration - expect 2 to 3 calls",
			duration:       2500 * time.Millisecond,
			message:        "hello",
			expectFnCalled: true,
			minCalls:       2,
			maxCalls:       3,
		},
		{
			name:           "Exact 1s duration - expect 1 call",
			duration:       1 * time.Second,
			message:        "one",
			expectFnCalled: true,
			minCalls:       1,
			maxCalls:       1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mu sync.Mutex
			var calls []string

			sleepWithMessage(tt.duration, tt.message, func(m string) {
				mu.Lock()
				defer mu.Unlock()
				calls = append(calls, m)
			})

			if tt.expectFnCalled {
				count := len(calls)
				if count < tt.minCalls || count > tt.maxCalls {
					t.Errorf("expected between %d and %d calls, got %d", tt.minCalls, tt.maxCalls, count)
				}
				for _, msg := range calls {
					if msg != tt.message {
						t.Errorf("unexpected message: got %s, want %s", msg, tt.message)
					}
				}
			} else {
				if len(calls) > 0 {
					t.Errorf("expected no calls, but got %d", len(calls))
				}
			}
		})
	}
}

func TestStep_getEchoOutput(t *testing.T) {
	tests := []struct {
		name        string
		echo        string
		context     StepContext
		expected    string
		expectError bool
	}{
		{
			name:        "single line echo",
			echo:        "Hello World",
			context:     StepContext{},
			expected:    "       Hello World\n",
			expectError: false,
		},
		{
			name:        "multi-line echo with explicit newlines",
			echo:        "Line 1\nLine 2\nLine 3",
			context:     StepContext{},
			expected:    "       Line 1\n       Line 2\n       Line 3\n",
			expectError: false,
		},
		{
			name:        "complex multiline with indentation",
			echo:        "Header\n  Indented\n    More indented\nBack to left",
			context:     StepContext{},
			expected:    "       Header\n         Indented\n           More indented\n       Back to left\n",
			expectError: false,
		},
		{
			name:        "empty line handling",
			echo:        "Line 1\n\nLine 3",
			context:     StepContext{},
			expected:    "       Line 1\n       \n       Line 3\n",
			expectError: false,
		},
		{
			name:        "template expression",
			echo:        "Status: {{vars.status}}",
			context:     StepContext{Vars: map[string]any{"status": "OK"}},
			expected:    "       Status: OK\n",
			expectError: false,
		},
		{
			name:        "multiline template expression",
			echo:        "Status: {{vars.status}}\nCode: {{vars.code}}",
			context:     StepContext{Vars: map[string]any{"status": "OK", "code": "200"}},
			expected:    "       Status: OK\n       Code: 200\n",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := &Step{
				Echo: tt.echo,
				ctx:  tt.context,
				Expr: &Expr{},
			}
			printer := NewPrinter(false, []string{})

			result := step.getEchoOutput(printer)

			if result != tt.expected {
				t.Errorf("getEchoOutput() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestStep_getEchoOutput_Error(t *testing.T) {
	step := &Step{
		Echo: "{{invalid_expression + }}", // Invalid syntax that will cause template error
		ctx:  StepContext{},
		Expr: &Expr{},
	}
	printer := NewPrinter(false, []string{})

	result := step.getEchoOutput(printer)

	// Should contain error indication when template evaluation fails
	if !strings.Contains(result, "CompileError") && !strings.Contains(result, "RuntimeError") && !strings.Contains(result, "Echo\nerror:") {
		t.Errorf("getEchoOutput() with invalid expression should return error message, got %q", result)
	}

	// Verify indentation is applied even to error messages
	lines := strings.Split(strings.TrimSuffix(result, "\n"), "\n")
	for _, line := range lines {
		if !strings.HasPrefix(line, "       ") {
			t.Errorf("getEchoOutput() should indent all lines including errors, line without indent: %q", line)
		}
	}
}
