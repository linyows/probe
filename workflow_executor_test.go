package probe

import (
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

			config := Config{Verbose: false, Output: NewSilentOutput()}
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

		config := Config{Verbose: false, Output: NewSilentOutput()}
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

		config := Config{Verbose: false, Output: NewSilentOutput()}
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

		config := Config{Verbose: false, Output: NewSilentOutput()}
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

		config := Config{Verbose: false, Output: NewSilentOutput()}
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

		config := Config{Verbose: false, Output: NewSilentOutput()}
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

		config := Config{Verbose: false, Output: NewSilentOutput()}
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

		output := NewSilentOutput()

		// This should not panic and should execute successfully
		workflow.printDetailedResults(workflowOutput, output)

		// If we get here without panic, the test passes
	})
}
