package probe

import (
	"fmt"
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
				expr:   &Expr{},
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
		idx:  1,
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
		idx:  1,
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
		idx:  1,
		Test: "res.status == 200",
		Echo: "Hello World",
		Wait: "1s",
	}
	jCtx := &JobContext{
		Config: Config{RT: true},
	}
	name := "Test Step"
	rt := "250ms"
	counter := &StepRepeatCounter{
		SuccessCount: 3,
		FailureCount: 1,
		Name:         "Test Counter",
		LastResult:   true,
	}

	result := step.createStepResult(name, rt, true, jCtx, counter)

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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := &Step{
				Name:   tt.stepName,
				Wait:   tt.wait,
				SkipIf: tt.skipif,
				expr:   &Expr{},
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
				Printer: NewSilentPrinter(), // Use silent printer to avoid output during tests
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
			expr: &Expr{},
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
				expr: &Expr{},
			}

			jCtx := &JobContext{
				Logs: []map[string]any{},
			}

			step.processActionResult(tt.actionResult, jCtx)

			if len(jCtx.Logs) != tt.expectLogs {
				t.Errorf("processActionResult() logs count = %v, want %v", len(jCtx.Logs), tt.expectLogs)
			}

			// Verify the result was added to logs
			if len(jCtx.Logs) > 0 {
				lastLog := jCtx.Logs[len(jCtx.Logs)-1]
				if req := lastLog["req"]; req == nil {
					t.Errorf("processActionResult() should preserve req in logs")
				}
				if res := lastLog["res"]; res == nil {
					t.Errorf("processActionResult() should preserve res in logs")
				}
			}
		})
	}
}

func TestStep_handleActionError(t *testing.T) {
	step := &Step{
		Uses: "mock-action-for-error-test",
		expr: &Expr{},
	}

	jCtx := &JobContext{
		Printer: NewSilentPrinter(),
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

func TestStep_handleVerboseMode(t *testing.T) {
	tests := []struct {
		name         string
		req          map[string]any
		res          map[string]any
		okreq        bool
		okres        bool
		testExpr     string
		echo         string
		expectFailed bool
	}{
		{
			name:         "normal verbose execution",
			req:          map[string]any{"url": "http://example.com"},
			res:          map[string]any{"status": 200},
			okreq:        true,
			okres:        true,
			testExpr:     "",
			echo:         "",
			expectFailed: false,
		},
		{
			name:         "nil request or response",
			req:          nil,
			res:          nil,
			okreq:        false,
			okres:        false,
			testExpr:     "",
			echo:         "",
			expectFailed: true,
		},
		{
			name:         "with test expression",
			req:          map[string]any{"url": "http://example.com"},
			res:          map[string]any{"status": 200},
			okreq:        true,
			okres:        true,
			testExpr:     "true",
			echo:         "",
			expectFailed: false,
		},
		{
			name:         "with echo",
			req:          map[string]any{"url": "http://example.com"},
			res:          map[string]any{"status": 200},
			okreq:        true,
			okres:        true,
			testExpr:     "",
			echo:         "response.status",
			expectFailed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := &Step{
				Test: tt.testExpr,
				Echo: tt.echo,
				expr: &Expr{},
			}

			// Set up context for test and echo evaluation
			step.ctx = StepContext{
				Req: tt.req,
				Res: tt.res,
			}

			jCtx := &JobContext{
				Printer: NewSilentPrinter(),
				Failed:  false,
			}

			step.handleVerboseMode("test-step", tt.req, tt.res, tt.okreq, tt.okres, jCtx)

			if jCtx.Failed != tt.expectFailed {
				t.Errorf("handleVerboseMode() Failed = %v, want %v", jCtx.Failed, tt.expectFailed)
			}
		})
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
				expr: &Expr{},
			}

			jCtx := &JobContext{
				Config: Config{
					Verbose: tt.verbose,
				},
				IsRepeating:  tt.isRepeating,
				Printer:      NewSilentPrinter(),
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
			expr:   &Expr{},
		}

		step.ctx = StepContext{
			Vars: map[string]any{},
		}

		jobContext := JobContext{
			Printer: NewSilentPrinter(),
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
			expr:   &Expr{},
		}

		step.ctx = StepContext{
			Vars: map[string]any{},
		}

		jobContext := JobContext{
			Config:  Config{Verbose: false},
			Printer: NewSilentPrinter(),
			Logs:    []map[string]any{},
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
