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
	results := wb.GetStepResults(jobID)
	if len(results) != 2 {
		t.Errorf("Expected 2 step results, got %d", len(results))
	}

	if results[0].Name != "Step 1" {
		t.Errorf("Expected first step name 'Step 1', got '%s'", results[0].Name)
	}

	if results[1].Name != "Step 2" {
		t.Errorf("Expected second step name 'Step 2', got '%s'", results[1].Name)
	}

	if results[1].RepeatCounter == nil {
		t.Error("Expected RepeatCounter to be set for second step")
	} else if results[1].RepeatCounter.SuccessCount != 3 {
		t.Errorf("Expected RepeatCounter.SuccessCount = 3, got %d", results[1].RepeatCounter.SuccessCount)
	}
}

func TestWorkflowBuffer_GetStepResults_NonExistentJob(t *testing.T) {
	wb := NewWorkflowBuffer()

	results := wb.GetStepResults("non-existent-job")
	if results != nil {
		t.Errorf("Expected nil for non-existent job, got %v", results)
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

	// Verify no results are returned
	results := wb.GetStepResults("non-existent-job")
	if results != nil {
		t.Errorf("Expected nil for non-existent job, got %v", results)
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

	// Goroutine 2: Read step results
	go func() {
		for i := 0; i < 5; i++ {
			wb.GetStepResults(jobID)
		}
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	// Verify final state
	results := wb.GetStepResults(jobID)
	if len(results) != 10 {
		t.Errorf("Expected 10 step results after concurrent operations, got %d", len(results))
	}
}
