package probe

import (
	"strings"
	"testing"
	"time"
)

// Test basic executor creation and interface compliance
func TestJobExecutor_Creation(t *testing.T) {
	workflow := &Workflow{Name: "test-workflow"}
	
	t.Run("BufferedJobExecutor creation", func(t *testing.T) {
		executor := NewBufferedJobExecutor(workflow)
		if executor == nil {
			t.Fatal("NewBufferedJobExecutor should not return nil")
		}
		
		// Ensure it implements JobExecutor interface
		var _ JobExecutor = executor
	})
}

func TestExecutionResult_Structure(t *testing.T) {
	result := ExecutionResult{
		Success:  true,
		Duration: 100 * time.Millisecond,
		Output:   "test output",
		Error:    nil,
	}
	
	if !result.Success {
		t.Error("ExecutionResult.Success should be true")
	}
	
	if result.Duration != 100*time.Millisecond {
		t.Errorf("ExecutionResult.Duration = %v, want %v", result.Duration, 100*time.Millisecond)
	}
	
	if result.Output != "test output" {
		t.Errorf("ExecutionResult.Output = %v, want %v", result.Output, "test output")
	}
	
	if result.Error != nil {
		t.Errorf("ExecutionResult.Error = %v, want nil", result.Error)
	}
}

func TestExecutionConfig_Structure(t *testing.T) {
	scheduler := NewJobScheduler()
	workflowBuffer := NewWorkflowBuffer()
	
	config := ExecutionConfig{
		HasDependencies: true,
		WorkflowBuffer:  workflowBuffer,
		JobScheduler:    scheduler,
	}
	
	if !config.HasDependencies {
		t.Error("ExecutionConfig.HasDependencies should be true")
	}
	
	if config.WorkflowBuffer != workflowBuffer {
		t.Error("ExecutionConfig.WorkflowBuffer should match assigned value")
	}
	
	if config.JobScheduler != scheduler {
		t.Error("ExecutionConfig.JobScheduler should match assigned value")
	}
}

func TestBufferedJobExecutor_AppendRepeatStepResults(t *testing.T) {
	workflow := &Workflow{Name: "test-workflow"}
	executor := NewBufferedJobExecutor(workflow)
	
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
		Config: Config{Verbose: false, Printer: NewPrinter(false)},
		Printer: NewPrinter(false),
	}
	
	job := &Job{
		Name: "test-job",
		Steps: []*Step{
			{
				Name: "test-step",
				Test: "status == 200",
			},
		},
	}
	
	jobBuffer := &JobBuffer{
		JobName: job.Name,
		JobID:   "test-job",
		Buffer:  strings.Builder{},
	}
	
	// Call the method
	executor.appendRepeatStepResults(&ctx, job, jobBuffer)
	
	// Check if output was captured
	output := jobBuffer.Buffer.String()
	if len(output) == 0 {
		t.Error("appendRepeatStepResults should generate output")
	}
	
	// Should contain step outputs (success ratio or similar indicator)
	if !strings.Contains(output, "success") {
		t.Errorf("Output should contain success indicator, got: %s", output)
	}
}

func TestJobExecutor_Integration_WithMockJob(t *testing.T) {
	// Test that the executor can handle basic job execution scenarios
	// without relying on the actual job.Start() method which has plugin dependencies
	
	t.Run("executor creation and interface compliance", func(t *testing.T) {
		workflow := &Workflow{Name: "test-workflow"}
		
		// Test that the executor can be created and implement the interface
		bufferedExecutor := NewBufferedJobExecutor(workflow)
		
		if bufferedExecutor == nil {
			t.Error("BufferedJobExecutor creation failed")
		}
		
		// Verify interface compliance
		var _ JobExecutor = bufferedExecutor
	})
}