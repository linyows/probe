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

	// Create WorkflowBuffer with JobBuffer
	workflowBuffer := NewWorkflowBuffer()
	jobBuffer := &JobBuffer{
		JobName: job.Name,
		JobID:   "test-job",
	}
	workflowBuffer.Jobs["test-job"] = jobBuffer

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
		WorkflowBuffer: workflowBuffer,
	}

	// Call the method
	executor.appendRepeatStepResults(&ctx)

	// Check if step results were added to WorkflowBuffer
	jobBuffer, exists := workflowBuffer.Jobs["test-job"]
	if !exists {
		t.Fatal("Job buffer should exist after appendRepeatStepResults")
	}
	if len(jobBuffer.StepResults) == 0 {
		t.Error("appendRepeatStepResults should add StepResults to WorkflowBuffer")
	}

	// Should have created a StepResult with RepeatCounter
	if len(jobBuffer.StepResults) != 1 {
		t.Errorf("Expected 1 step result, got %d", len(jobBuffer.StepResults))
	}

	stepResult := jobBuffer.StepResults[0]
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
