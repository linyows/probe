package probe

import (
	"testing"
)

// Test basic executor creation and interface compliance
func TestJobExecutor_Creation(t *testing.T) {
	workflow := &Workflow{Name: "test-workflow"}
	job := &Job{Name: "test-job", Steps: []*Step{}}

	t.Run("Executor creation", func(t *testing.T) {
		executor := NewExecutor(workflow, job)
		if executor == nil {
			t.Fatal("NewExecutor should not return nil")
		}
	})
}

// TestExecutionResult_Structure is no longer needed as ExecutionResult has been removed

// TestExecutionConfig_Structure is no longer needed as ExecutionConfig has been removed

func TestExecutor_AppendRepeatStepResults(t *testing.T) {
	workflow := &Workflow{Name: "test-workflow"}
	job := &Job{
		Name: "test-job",
		ID:   "test-job",
		Steps: []*Step{
			{
				Name: "test-step",
				Test: "status == 200",
			},
		},
	}
	executor := NewExecutor(workflow, job)

	// Create WorkflowBuffer with JobResult
	result := NewResult()
	jobResult := &JobResult{
		JobName: job.Name,
		JobID:   "test-job",
	}
	result.Jobs["test-job"] = jobResult

	// Create test context with step counters
	ctx := JobContext{
		StepCounters: map[int]StepRepeatCounter{
			0: {
				SuccessCount: 3,
				FailureCount: 1,
				Name:         "test-step",
				LastResult:   true,
			},
		},
		Config:         Config{Verbose: false},
		Printer:        NewPrinter(false, []string{}),
		Result: result,
	}

	// Call the method
	executor.appendRepeatStepResults(&ctx)

	// Check if step results were added to WorkflowBuffer
	jobResult, exists := result.Jobs["test-job"]
	if !exists {
		t.Fatal("Job buffer should exist after appendRepeatStepResults")
	}
	if len(jobResult.StepResults) == 0 {
		t.Error("appendRepeatStepResults should add StepResults to WorkflowBuffer")
	}

	// Should have created a StepResult with RepeatCounter
	if len(jobResult.StepResults) != 1 {
		t.Errorf("Expected 1 step result, got %d", len(jobResult.StepResults))
	}

	stepResult := jobResult.StepResults[0]
	if stepResult.RepeatCounter == nil {
		t.Error("StepResult should have RepeatCounter")
	}

	if stepResult.RepeatCounter.SuccessCount != 3 {
		t.Errorf("Expected SuccessCount=3, got %d", stepResult.RepeatCounter.SuccessCount)
	}

	if stepResult.RepeatCounter.FailureCount != 1 {
		t.Errorf("Expected FailureCount=1, got %d", stepResult.RepeatCounter.FailureCount)
	}
}

func TestJobExecutor_Integration_WithMockJob(t *testing.T) {
	// Test that the executor can handle basic job execution scenarios
	// without relying on the actual job.Start() method which has plugin dependencies

	t.Run("executor creation and interface compliance", func(t *testing.T) {
		workflow := &Workflow{Name: "test-workflow"}
		job := &Job{Name: "test-job", Steps: []*Step{}}

		// Test that the executor can be created
		executor := NewExecutor(workflow, job)

		if executor == nil {
			t.Error("Executor creation failed")
		}
	})
}
