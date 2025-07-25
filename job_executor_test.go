package probe

import (
	"strings"
	"testing"
	"time"
)

// Test basic executor creation and interface compliance
func TestJobExecutor_Creation(t *testing.T) {
	workflow := &Workflow{Name: "test-workflow"}
	
	t.Run("ParallelJobExecutor creation", func(t *testing.T) {
		executor := NewParallelJobExecutor(workflow)
		if executor == nil {
			t.Fatal("NewParallelJobExecutor should not return nil")
		}
		
		// Ensure it implements JobExecutor interface
		var _ JobExecutor = executor
	})
	
	t.Run("SequentialJobExecutor creation", func(t *testing.T) {
		executor := NewSequentialJobExecutor(workflow)
		if executor == nil {
			t.Fatal("NewSequentialJobExecutor should not return nil")
		}
		
		// Ensure it implements JobExecutor interface
		var _ JobExecutor = executor
	})
	
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
	workflowOutput := NewWorkflowOutput()
	
	config := ExecutionConfig{
		UseBuffering:     true,
		UseParallel:      false,
		HasDependencies:  true,
		WorkflowOutput:   workflowOutput,
		JobScheduler:     scheduler,
	}
	
	if !config.UseBuffering {
		t.Error("ExecutionConfig.UseBuffering should be true")
	}
	
	if config.UseParallel {
		t.Error("ExecutionConfig.UseParallel should be false")
	}
	
	if !config.HasDependencies {
		t.Error("ExecutionConfig.HasDependencies should be true")
	}
	
	if config.WorkflowOutput != workflowOutput {
		t.Error("ExecutionConfig.WorkflowOutput should match assigned value")
	}
	
	if config.JobScheduler != scheduler {
		t.Error("ExecutionConfig.JobScheduler should match assigned value")
	}
}

func TestBufferedJobExecutor_PrintRepeatStepResults(t *testing.T) {
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
	
	jobOutput := &JobOutput{
		JobName: job.Name,
		JobID:   "test-job",
		Buffer:  strings.Builder{},
	}
	
	// Call the method
	executor.printRepeatStepResults(&ctx, job, jobOutput)
	
	// Check if output was captured
	output := jobOutput.Buffer.String()
	if len(output) == 0 {
		t.Error("printRepeatStepResults should generate output")
	}
	
	// Should contain step outputs (success ratio or similar indicator)
	if !strings.Contains(output, "success") {
		t.Errorf("Output should contain success indicator, got: %s", output)
	}
}

func TestJobExecutor_Integration_WithMockJob(t *testing.T) {
	// Test that the executors can handle basic job execution scenarios
	// without relying on the actual job.Start() method which has plugin dependencies
	
	t.Run("executor creation and interface compliance", func(t *testing.T) {
		workflow := &Workflow{Name: "test-workflow"}
		
		// Test that all executors can be created and implement the interface
		parallelExecutor := NewParallelJobExecutor(workflow)
		sequentialExecutor := NewSequentialJobExecutor(workflow) 
		bufferedExecutor := NewBufferedJobExecutor(workflow)
		
		if parallelExecutor == nil {
			t.Error("ParallelJobExecutor creation failed")
		}
		if sequentialExecutor == nil {
			t.Error("SequentialJobExecutor creation failed")
		}
		if bufferedExecutor == nil {
			t.Error("BufferedJobExecutor creation failed")
		}
		
		// Verify interface compliance
		var _ JobExecutor = parallelExecutor
		var _ JobExecutor = sequentialExecutor
		var _ JobExecutor = bufferedExecutor
	})
}