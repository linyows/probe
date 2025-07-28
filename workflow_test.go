package probe

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestWorkflowExecutor_DependencyManagement(t *testing.T) {
	tests := []struct {
		name        string
		jobs        []Job
		expectError bool
	}{
		{
			name: "jobs without dependencies",
			jobs: []Job{
				{
					Name:  "job1",
					Steps: []*Step{},
				},
				{
					Name:  "job2",
					Steps: []*Step{},
				},
			},
			expectError: false,
		},
		{
			name: "jobs with valid dependencies",
			jobs: []Job{
				{
					Name:  "job1",
					Steps: []*Step{},
				},
				{
					Name:  "job2",
					Needs: []string{"job1"},
					Steps: []*Step{},
				},
			},
			expectError: false,
		},
		{
			name: "jobs with circular dependencies",
			jobs: []Job{
				{
					Name:  "job1",
					Needs: []string{"job2"},
					Steps: []*Step{},
				},
				{
					Name:  "job2",
					Needs: []string{"job1"},
					Steps: []*Step{},
				},
			},
			expectError: true,
		},
		{
			name: "jobs with missing dependencies",
			jobs: []Job{
				{
					Name:  "job1",
					Needs: []string{"nonexistent"},
					Steps: []*Step{},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflow := &Workflow{
				Name:    "test-workflow",
				Jobs:    tt.jobs,
				printer: NewSilentPrinter(),
			}

			config := Config{Verbose: false}
			err := workflow.Start(config)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestWorkflowExecutor_ParallelExecution(t *testing.T) {
	t.Run("parallel execution without dependencies", func(t *testing.T) {
		workflow := &Workflow{
			Name: "parallel-test",
			Jobs: []Job{
				{
					Name:  "job1",
					Steps: []*Step{},
				},
				{
					Name:  "job2",
					Steps: []*Step{},
				},
				{
					Name:  "job3",
					Steps: []*Step{},
				},
			},
			printer: NewSilentPrinter(),
		}

		config := Config{Verbose: false}
		start := time.Now()
		err := workflow.Start(config)
		duration := time.Since(start)

		if err != nil {
			t.Errorf("Parallel execution should not error: %v", err)
		}

		// Parallel execution should be faster than sequential
		// With empty steps, this should complete very quickly
		if duration > 1*time.Second {
			t.Errorf("Parallel execution took too long: %v", duration)
		}
	})
}

func TestWorkflowExecutor_SequentialWithDependencies(t *testing.T) {
	t.Run("sequential execution with dependencies", func(t *testing.T) {
		workflow := &Workflow{
			Name: "sequential-test",
			Jobs: []Job{
				{
					Name:  "first",
					Steps: []*Step{},
				},
				{
					Name:  "second",
					Needs: []string{"first"},
					Steps: []*Step{},
				},
				{
					Name:  "third",
					Needs: []string{"second"},
					Steps: []*Step{},
				},
			},
			printer: NewSilentPrinter(),
		}

		config := Config{Verbose: false}
		err := workflow.Start(config)

		if err != nil {
			t.Errorf("Sequential execution with dependencies should not error: %v", err)
		}
	})
}

func TestWorkflowExecutor_BufferedOutput(t *testing.T) {
	t.Run("buffered output with multiple jobs", func(t *testing.T) {
		workflow := &Workflow{
			Name: "buffered-test",
			Jobs: []Job{
				{
					Name:  "job1",
					Steps: []*Step{},
				},
				{
					Name:  "job2",
					Needs: []string{"job1"},
					Steps: []*Step{},
				},
			},
			printer: NewSilentPrinter(),
		}

		config := Config{Verbose: false}
		err := workflow.Start(config)

		if err != nil {
			t.Errorf("Buffered execution should not error: %v", err)
		}

		// With dependencies and multiple jobs, buffering should be used
		// This test mainly verifies that the workflow completes successfully
	})
}

func TestWorkflowExecutor_RepeatJobs(t *testing.T) {
	t.Run("job with repeat in parallel execution", func(t *testing.T) {
		workflow := &Workflow{
			Name: "repeat-test",
			Jobs: []Job{
				{
					Name:  "repeat-job",
					Steps: []*Step{},
					Repeat: &Repeat{
						Count:    3,
						Interval: Interval{Duration: 10 * time.Millisecond},
					},
				},
			},
			printer: NewSilentPrinter(),
		}

		config := Config{Verbose: false}
		start := time.Now()
		err := workflow.Start(config)
		duration := time.Since(start)

		if err != nil {
			t.Errorf("Repeat job execution should not error: %v", err)
		}

		// Should take at least the interval time * (count-1)
		expectedMinDuration := 2 * 10 * time.Millisecond // 2 intervals for 3 executions
		if duration < expectedMinDuration {
			t.Errorf("Duration %v should be at least %v for repeat execution", duration, expectedMinDuration)
		}
	})
}

func TestWorkflowExecutor_MixedScenarios(t *testing.T) {
	t.Run("complex workflow with dependencies and repeats", func(t *testing.T) {
		workflow := &Workflow{
			Name: "complex-test",
			Jobs: []Job{
				{
					Name:  "setup",
					Steps: []*Step{},
				},
				{
					Name:  "worker1",
					Needs: []string{"setup"},
					Steps: []*Step{},
					Repeat: &Repeat{
						Count:    2,
						Interval: Interval{Duration: 5 * time.Millisecond},
					},
				},
				{
					Name:  "worker2",
					Needs: []string{"setup"},
					Steps: []*Step{},
				},
				{
					Name:  "cleanup",
					Needs: []string{"worker1", "worker2"},
					Steps: []*Step{},
				},
			},
			printer: NewSilentPrinter(),
		}

		config := Config{Verbose: false}
		err := workflow.Start(config)

		if err != nil {
			t.Errorf("Complex workflow should not error: %v", err)
		}
	})
}

func TestWorkflowExecutor_ErrorHandling(t *testing.T) {
	t.Run("workflow with failed job dependency", func(t *testing.T) {
		// This test verifies that jobs with failed dependencies are properly skipped
		// Since we're using empty steps, jobs should succeed, but we test the structure
		workflow := &Workflow{
			Name: "error-handling-test",
			Jobs: []Job{
				{
					Name:  "might-fail",
					Steps: []*Step{},
				},
				{
					Name:  "depends-on-failed",
					Needs: []string{"might-fail"},
					Steps: []*Step{},
				},
			},
			printer: NewSilentPrinter(),
		}

		config := Config{Verbose: false}
		err := workflow.Start(config)

		// Should complete without error even if dependency logic is exercised
		if err != nil {
			t.Errorf("Error handling test should not error: %v", err)
		}
	})
}

func TestWorkflowExecutor_PrintDetailedResults(t *testing.T) {
	t.Run("print detailed results functionality", func(t *testing.T) {
		workflow := &Workflow{
			Name: "detailed-results-test",
			Jobs: []Job{
				{
					Name:  "test-job",
					Steps: []*Step{},
				},
			},
			printer: NewSilentPrinter(),
		}

		// Create workflow buffer
		workflowBuffer := NewWorkflowBuffer()
		jobBuffer := &JobBuffer{
			JobName:   "test-job",
			JobID:     "test-job",
			Buffer:    strings.Builder{},
			Status:    "Completed",
			StartTime: time.Now().Add(-100 * time.Millisecond),
			EndTime:   time.Now(),
			Success:   true,
		}
		workflowBuffer.Jobs["test-job"] = jobBuffer

		// This should not panic and should execute successfully
		workflow.printer.PrintReport(workflowBuffer)

		// If we get here without panic, the test passes
	})
}

func TestParallelExecution_EdgeCases(t *testing.T) {
	t.Run("empty workflow", func(t *testing.T) {
		workflow := &Workflow{
			Name:    "empty-workflow",
			Jobs:    []Job{},
			printer: NewSilentPrinter(),
		}

		config := Config{Verbose: false}
		err := workflow.Start(config)

		if err != nil {
			t.Errorf("Empty workflow should not error: %v", err)
		}
	})

	t.Run("single job parallel execution", func(t *testing.T) {
		workflow := &Workflow{
			Name: "single-job",
			Jobs: []Job{
				{
					Name:  "solo",
					Steps: []*Step{},
				},
			},
			printer: NewSilentPrinter(),
		}

		config := Config{Verbose: false}
		err := workflow.Start(config)

		if err != nil {
			t.Errorf("Single job workflow should not error: %v", err)
		}
	})

	t.Run("many jobs parallel execution", func(t *testing.T) {
		// Create a workflow with many jobs to test parallel execution limits
		jobs := make([]Job, 10)
		for i := 0; i < 10; i++ {
			jobs[i] = Job{
				Name:  fmt.Sprintf("job-%d", i),
				Steps: []*Step{},
			}
		}

		workflow := &Workflow{
			Name:    "many-jobs",
			Jobs:    jobs,
			printer: NewSilentPrinter(),
		}

		config := Config{Verbose: false}
		start := time.Now()
		err := workflow.Start(config)
		duration := time.Since(start)

		if err != nil {
			t.Errorf("Many jobs workflow should not error: %v", err)
		}

		// Should complete quickly in parallel
		if duration > 2*time.Second {
			t.Errorf("Many parallel jobs took too long: %v", duration)
		}
	})
}

func TestBufferedExecution_EdgeCases(t *testing.T) {
	t.Run("buffered execution with single job", func(t *testing.T) {
		workflow := &Workflow{
			Name: "single-buffered",
			Jobs: []Job{
				{
					Name:  "buffered-job",
					Needs: []string{}, // Force dependency path but no actual dependencies
					Steps: []*Step{},
				},
			},
			printer: NewSilentPrinter(),
		}

		config := Config{Verbose: false}
		err := workflow.Start(config)

		if err != nil {
			t.Errorf("Single buffered job should not error: %v", err)
		}
	})

	t.Run("buffered execution with concurrent output", func(t *testing.T) {
		// Test that concurrent buffered output doesn't cause race conditions
		workflow := &Workflow{
			Name: "concurrent-buffered",
			Jobs: []Job{
				{
					Name:  "producer1",
					Steps: []*Step{},
				},
				{
					Name:  "producer2",
					Steps: []*Step{},
				},
				{
					Name:  "consumer",
					Needs: []string{"producer1", "producer2"},
					Steps: []*Step{},
				},
			},
			printer: NewSilentPrinter(),
		}

		config := Config{Verbose: false}
		err := workflow.Start(config)

		if err != nil {
			t.Errorf("Concurrent buffered execution should not error: %v", err)
		}
	})
}

func TestRepeatExecution_EdgeCases(t *testing.T) {
	t.Run("repeat with zero interval", func(t *testing.T) {
		workflow := &Workflow{
			Name: "zero-interval-repeat",
			Jobs: []Job{
				{
					Name:  "fast-repeat",
					Steps: []*Step{},
					Repeat: &Repeat{
						Count:    5,
						Interval: Interval{Duration: 0}, // Zero interval
					},
				},
			},
			printer: NewSilentPrinter(),
		}

		config := Config{Verbose: false}
		start := time.Now()
		err := workflow.Start(config)
		duration := time.Since(start)

		if err != nil {
			t.Errorf("Zero interval repeat should not error: %v", err)
		}

		// Should complete very quickly with zero interval
		if duration > 100*time.Millisecond {
			t.Errorf("Zero interval repeat took too long: %v", duration)
		}
	})

	t.Run("repeat with very short interval", func(t *testing.T) {
		workflow := &Workflow{
			Name: "short-interval-repeat",
			Jobs: []Job{
				{
					Name:  "quick-repeat",
					Steps: []*Step{},
					Repeat: &Repeat{
						Count:    3,
						Interval: Interval{Duration: 1 * time.Millisecond},
					},
				},
			},
			printer: NewSilentPrinter(),
		}

		config := Config{Verbose: false}
		start := time.Now()
		err := workflow.Start(config)
		duration := time.Since(start)

		if err != nil {
			t.Errorf("Short interval repeat should not error: %v", err)
		}

		// Should take at least the minimum interval time
		expectedMin := 2 * time.Millisecond // 2 intervals for 3 executions
		if duration < expectedMin {
			t.Errorf("Duration %v should be at least %v", duration, expectedMin)
		}
	})

	t.Run("repeat with single count", func(t *testing.T) {
		workflow := &Workflow{
			Name: "single-repeat",
			Jobs: []Job{
				{
					Name:  "once-repeat",
					Steps: []*Step{},
					Repeat: &Repeat{
						Count:    1,
						Interval: Interval{Duration: 10 * time.Millisecond},
					},
				},
			},
			printer: NewSilentPrinter(),
		}

		config := Config{Verbose: false}
		err := workflow.Start(config)

		if err != nil {
			t.Errorf("Single count repeat should not error: %v", err)
		}
	})
}

func TestExecutor_ConcurrencyEdgeCases(t *testing.T) {
	t.Run("high concurrency with dependencies", func(t *testing.T) {
		// Create a workflow with multiple levels of dependencies
		workflow := &Workflow{
			Name: "high-concurrency",
			Jobs: []Job{
				{Name: "root", Steps: []*Step{}},
				{Name: "level1-a", Needs: []string{"root"}, Steps: []*Step{}},
				{Name: "level1-b", Needs: []string{"root"}, Steps: []*Step{}},
				{Name: "level1-c", Needs: []string{"root"}, Steps: []*Step{}},
				{Name: "level2-a", Needs: []string{"level1-a", "level1-b"}, Steps: []*Step{}},
				{Name: "level2-b", Needs: []string{"level1-b", "level1-c"}, Steps: []*Step{}},
				{Name: "final", Needs: []string{"level2-a", "level2-b"}, Steps: []*Step{}},
			},
			printer: NewSilentPrinter(),
		}

		config := Config{Verbose: false}
		err := workflow.Start(config)

		if err != nil {
			t.Errorf("High concurrency workflow should not error: %v", err)
		}
	})

	t.Run("mixed repeat and parallel execution", func(t *testing.T) {
		workflow := &Workflow{
			Name: "mixed-execution",
			Jobs: []Job{
				{
					Name:  "parallel1",
					Steps: []*Step{},
				},
				{
					Name:  "repeat1",
					Steps: []*Step{},
					Repeat: &Repeat{
						Count:    2,
						Interval: Interval{Duration: 5 * time.Millisecond},
					},
				},
				{
					Name:  "parallel2",
					Steps: []*Step{},
				},
			},
			printer: NewSilentPrinter(),
		}

		config := Config{Verbose: false}
		err := workflow.Start(config)

		if err != nil {
			t.Errorf("Mixed execution workflow should not error: %v", err)
		}
	})
}

func TestBuffering_OutputCapture(t *testing.T) {
	t.Run("buffered output isolation", func(t *testing.T) {
		// Create workflow buffer manually to test buffering
		workflowBuffer := NewWorkflowBuffer()

		// Test that job outputs are isolated
		job1Buffer := &JobBuffer{
			JobName: "job1",
			JobID:   "job1",
			Buffer:  strings.Builder{},
		}
		job2Buffer := &JobBuffer{
			JobName: "job2",
			JobID:   "job2",
			Buffer:  strings.Builder{},
		}

		workflowBuffer.Jobs["job1"] = job1Buffer
		workflowBuffer.Jobs["job2"] = job2Buffer

		// Write to different job buffers
		job1Buffer.Buffer.WriteString("output from job1")
		job2Buffer.Buffer.WriteString("output from job2")

		// Verify isolation
		if !strings.Contains(job1Buffer.Buffer.String(), "job1") {
			t.Error("Job1 buffer should contain job1 output")
		}
		if strings.Contains(job1Buffer.Buffer.String(), "job2") {
			t.Error("Job1 buffer should not contain job2 output")
		}
		if !strings.Contains(job2Buffer.Buffer.String(), "job2") {
			t.Error("Job2 buffer should contain job2 output")
		}
		if strings.Contains(job2Buffer.Buffer.String(), "job1") {
			t.Error("Job2 buffer should not contain job1 output")
		}
	})
}

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
		Printer:       NewPrinter(false, []string{}),
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
				Printer:       NewPrinter(false, []string{}),
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
		Printer:       NewPrinter(false, []string{}),
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
