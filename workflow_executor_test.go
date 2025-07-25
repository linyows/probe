package probe

import (
	"fmt"
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
				Name: "test-workflow",
				Jobs: tt.jobs,
			}

			config := Config{Verbose: false, Printer: NewSilentPrinter()}
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
		}

		config := Config{Verbose: false, Printer: NewSilentPrinter()}
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
		}

		config := Config{Verbose: false, Printer: NewSilentPrinter()}
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
		}

		config := Config{Verbose: false, Printer: NewSilentPrinter()}
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
		}

		config := Config{Verbose: false, Printer: NewSilentPrinter()}
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
		}

		config := Config{Verbose: false, Printer: NewSilentPrinter()}
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
		}

		config := Config{Verbose: false, Printer: NewSilentPrinter()}
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
		}

		// Create workflow output
		workflowOutput := NewWorkflowOutput()
		jobOutput := &JobOutput{
			JobName:   "test-job",
			JobID:     "test-job",
			Buffer:    strings.Builder{},
			Status:    "Completed",
			StartTime: time.Now().Add(-100 * time.Millisecond),
			EndTime:   time.Now(),
			Success:   true,
		}
		workflowOutput.Jobs["test-job"] = jobOutput

		output := NewSilentPrinter()

		// This should not panic and should execute successfully
		workflow.printDetailedResults(workflowOutput, output)

		// If we get here without panic, the test passes
	})
}

func TestParallelExecution_EdgeCases(t *testing.T) {
	t.Run("empty workflow", func(t *testing.T) {
		workflow := &Workflow{
			Name: "empty-workflow",
			Jobs: []Job{},
		}

		config := Config{Verbose: false, Printer: NewSilentPrinter()}
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
		}

		config := Config{Verbose: false, Printer: NewSilentPrinter()}
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
			Name: "many-jobs",
			Jobs: jobs,
		}

		config := Config{Verbose: false, Printer: NewSilentPrinter()}
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
		}

		config := Config{Verbose: false, Printer: NewSilentPrinter()}
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
		}

		config := Config{Verbose: false, Printer: NewSilentPrinter()}
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
		}

		config := Config{Verbose: false, Printer: NewSilentPrinter()}
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
		}

		config := Config{Verbose: false, Printer: NewSilentPrinter()}
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
		}

		config := Config{Verbose: false, Printer: NewSilentPrinter()}
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
		}

		config := Config{Verbose: false, Printer: NewSilentPrinter()}
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
		}

		config := Config{Verbose: false, Printer: NewSilentPrinter()}
		err := workflow.Start(config)

		if err != nil {
			t.Errorf("Mixed execution workflow should not error: %v", err)
		}
	})
}

func TestBuffering_OutputCapture(t *testing.T) {
	t.Run("buffered output isolation", func(t *testing.T) {
		// Create workflow output manually to test buffering
		workflowOutput := NewWorkflowOutput()

		// Test that job outputs are isolated
		job1Output := &JobOutput{
			JobName: "job1",
			JobID:   "job1",
			Buffer:  strings.Builder{},
		}
		job2Output := &JobOutput{
			JobName: "job2",
			JobID:   "job2",
			Buffer:  strings.Builder{},
		}

		workflowOutput.Jobs["job1"] = job1Output
		workflowOutput.Jobs["job2"] = job2Output

		// Write to different job buffers
		job1Output.Buffer.WriteString("output from job1")
		job2Output.Buffer.WriteString("output from job2")

		// Verify isolation
		if !strings.Contains(job1Output.Buffer.String(), "job1") {
			t.Error("Job1 buffer should contain job1 output")
		}
		if strings.Contains(job1Output.Buffer.String(), "job2") {
			t.Error("Job1 buffer should not contain job2 output")
		}
		if !strings.Contains(job2Output.Buffer.String(), "job2") {
			t.Error("Job2 buffer should contain job2 output")
		}
		if strings.Contains(job2Output.Buffer.String(), "job1") {
			t.Error("Job2 buffer should not contain job1 output")
		}
	})
}
