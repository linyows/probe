package probe

import (
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func TestPrinter_PrintReport(t *testing.T) {
	// Create printer with job IDs
	jobIDs := []string{"job1", "job2"}
	printer := NewPrinter(false, jobIDs)

	// Create WorkflowBuffer with test data
	wb := NewWorkflowBuffer()

	// Job 1: Regular steps
	startTime1 := time.Now()
	endTime1 := startTime1.Add(2 * time.Second)
	job1Buffer := strings.Builder{}
	job1Buffer.WriteString("Step 1 output\nStep 2 output\n")
	wb.Jobs["job1"] = &JobBuffer{
		JobID:     "job1",
		JobName:   "Test Job 1",
		Buffer:    job1Buffer,
		StartTime: startTime1,
		EndTime:   endTime1,
		Success:   true,
		Status:    "Completed",
		StepResults: []StepResult{
			{
				Index:  0,
				Name:   "Step 1",
				Status: StatusSuccess,
				RT:     "100ms",
			},
			{
				Index:      1,
				Name:       "Step 2",
				Status:     StatusError,
				TestOutput: "Test failed",
			},
		},
	}

	// Job 2: Repeat step
	startTime2 := time.Now()
	endTime2 := startTime2.Add(3 * time.Second)
	wb.Jobs["job2"] = &JobBuffer{
		JobID:     "job2",
		JobName:   "Test Job 2",
		StartTime: startTime2,
		EndTime:   endTime2,
		Success:   false,
		Status:    "Failed",
		StepResults: []StepResult{
			{
				Index:   0,
				Name:    "Repeat Step",
				Status:  StatusWarning,
				HasTest: true,
				RepeatCounter: &StepRepeatCounter{
					SuccessCount: 3,
					FailureCount: 2,
					Name:         "Repeat Step",
					LastResult:   false,
				},
			},
		},
	}

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Call PrintReport
	printer.PrintReport(wb)

	// Restore stdout and get output
	w.Close()
	os.Stdout = oldStdout
	output, _ := io.ReadAll(r)
	outputStr := string(output)

	// Verify output contains job names and status (from PrintJobStatus)
	if !strings.Contains(outputStr, "Test Job 1") {
		t.Error("Output should contain 'Test Job 1'")
	}
	if !strings.Contains(outputStr, "Test Job 2") {
		t.Error("Output should contain 'Test Job 2'")
	}

	// Verify job status output
	if !strings.Contains(outputStr, "(Completed in") {
		t.Error("Output should contain job completion status")
	}
	if !strings.Contains(outputStr, "(Failed in") {
		t.Error("Output should contain job failure status")
	}

	// Verify step results are properly formatted
	if !strings.Contains(outputStr, "✔︎  Step 1") {
		t.Error("Output should contain formatted Step 1 result")
	}
	if !strings.Contains(outputStr, "✘  Step 2") {
		t.Error("Output should contain formatted Step 2 result")
	}
	if !strings.Contains(outputStr, "Test failed") {
		t.Error("Output should contain test failure message")
	}
	if !strings.Contains(outputStr, "(repeating 5 times)") {
		t.Error("Output should contain repeat step information")
	}

	// Verify footer
	if !strings.Contains(outputStr, "Total workflow time") {
		t.Error("Output should contain total workflow time")
	}
}

func TestPrinter_PrintReport_EmptyBuffer(t *testing.T) {
	printer := NewPrinter(false, []string{})

	// Test with nil buffer
	printer.PrintReport(nil)

	// Test with empty buffer
	wb := NewWorkflowBuffer()
	printer.PrintReport(wb)

	// Should not panic
}

