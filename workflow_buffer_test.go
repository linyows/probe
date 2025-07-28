package probe

import (
	"testing"
	"time"
)

func TestWorkflowBuffer_AddStepResult(t *testing.T) {
	wb := NewWorkflowBuffer()
	jobID := "test-job"

	// Add a job buffer first
	wb.Jobs[jobID] = &JobBuffer{
		JobID:       jobID,
		JobName:     "Test Job",
		StartTime:   time.Now(),
		StepResults: []StepResult{},
	}

	// Create test step results
	stepResult1 := StepResult{
		Index:  0,
		Name:   "Step 1",
		Status: StatusSuccess,
	}

	stepResult2 := StepResult{
		Index:  1,
		Name:   "Step 2",
		Status: StatusError,
		RepeatCounter: &StepRepeatCounter{
			SuccessCount: 3,
			FailureCount: 1,
		},
	}

	// Add step results
	wb.AddStepResult(jobID, stepResult1)
	wb.AddStepResult(jobID, stepResult2)

	// Verify step results were added
	jobBuffer, exists := wb.Jobs[jobID]
	if !exists {
		t.Fatal("Job buffer should exist")
	}
	if len(jobBuffer.StepResults) != 2 {
		t.Errorf("Expected 2 step results, got %d", len(jobBuffer.StepResults))
	}

	if jobBuffer.StepResults[0].Name != "Step 1" {
		t.Errorf("Expected first step name 'Step 1', got '%s'", jobBuffer.StepResults[0].Name)
	}

	if jobBuffer.StepResults[1].Name != "Step 2" {
		t.Errorf("Expected second step name 'Step 2', got '%s'", jobBuffer.StepResults[1].Name)
	}

	if jobBuffer.StepResults[1].RepeatCounter == nil {
		t.Error("Expected RepeatCounter to be set for second step")
	} else if jobBuffer.StepResults[1].RepeatCounter.SuccessCount != 3 {
		t.Errorf("Expected RepeatCounter.SuccessCount = 3, got %d", jobBuffer.StepResults[1].RepeatCounter.SuccessCount)
	}
}


func TestWorkflowBuffer_AddStepResult_NonExistentJob(t *testing.T) {
	wb := NewWorkflowBuffer()

	stepResult := StepResult{
		Index:  0,
		Name:   "Step 1",
		Status: StatusSuccess,
	}

	// This should not panic even if job doesn't exist
	wb.AddStepResult("non-existent-job", stepResult)

	// Verify no job buffer was created
	if _, exists := wb.Jobs["non-existent-job"]; exists {
		t.Error("Job buffer should not be created for non-existent job")
	}
}

func TestWorkflowBuffer_ConcurrentAccess(t *testing.T) {
	wb := NewWorkflowBuffer()
	jobID := "test-job"

	// Add a job buffer first
	wb.Jobs[jobID] = &JobBuffer{
		JobID:       jobID,
		JobName:     "Test Job",
		StartTime:   time.Now(),
		StepResults: []StepResult{},
	}

	// Test concurrent add and get operations
	done := make(chan bool, 2)

	// Goroutine 1: Add step results
	go func() {
		for i := 0; i < 10; i++ {
			stepResult := StepResult{
				Index:  i,
				Name:   "Step " + string(rune('0'+i)),
				Status: StatusSuccess,
			}
			wb.AddStepResult(jobID, stepResult)
		}
		done <- true
	}()

	// Goroutine 2: Read job buffer
	go func() {
		for i := 0; i < 5; i++ {
			jobBuffer := wb.Jobs[jobID]
			if jobBuffer != nil {
				_ = len(jobBuffer.StepResults)
			}
		}
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	// Verify final state
	jobBuffer, exists := wb.Jobs[jobID]
	if !exists {
		t.Fatal("Job buffer should exist after concurrent operations")
	}
	if len(jobBuffer.StepResults) != 10 {
		t.Errorf("Expected 10 step results after concurrent operations, got %d", len(jobBuffer.StepResults))
	}
}
