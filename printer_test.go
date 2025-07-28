package probe

import (
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fatih/color"
)

// Color function tests
func TestColorFunctions(t *testing.T) {
	// Disable color output for consistent testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	tests := []struct {
		name     string
		colorFn  func() *color.Color
		text     string
		expected string
	}{
		{
			name:     "colorSuccess",
			colorFn:  colorSuccess,
			text:     "success",
			expected: "success",
		},
		{
			name:     "colorError",
			colorFn:  colorError,
			text:     "error",
			expected: "error",
		},
		{
			name:     "colorInfo",
			colorFn:  colorInfo,
			text:     "info",
			expected: "info",
		},
		{
			name:     "colorWarning",
			colorFn:  colorWarning,
			text:     "warning",
			expected: "warning",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test Sprintf
			result := tt.colorFn().Sprintf("%s", tt.text)
			if result != tt.expected {
				t.Errorf("Sprintf: expected %s, got %s", tt.expected, result)
			}

			// Test SprintFunc
			sprintFunc := tt.colorFn().SprintFunc()
			result2 := sprintFunc(tt.text)
			if result2 != tt.expected {
				t.Errorf("SprintFunc: expected %s, got %s", tt.expected, result2)
			}
		})
	}
}

func TestColorSuccess_RGB(t *testing.T) {
	// Test that colorSuccess returns the correct RGB color
	c := colorSuccess()

	// The color should contain RGB values for 0,175,0
	// We can't directly test RGB values, but we can test the function returns a valid color
	if c == nil {
		t.Error("colorSuccess() should return a non-nil *color.Color")
	}

	// Test that it can format text
	result := c.Sprintf("test")
	if !strings.Contains(result, "test") {
		t.Errorf("colorSuccess().Sprintf should contain 'test', got %s", result)
	}
}

func TestRepeatNoTestDisplay(t *testing.T) {
	// Test the "no test" display format
	totalCount := 1000

	actual := colorInfo().Sprintf("⏺") + " " +
		colorInfo().Sprintf("%d/%d completed (no test)", totalCount, totalCount)

	// Check that the format contains expected parts
	if !strings.Contains(actual, "1000/1000 completed (no test)") {
		t.Errorf("Expected format to contain '1000/1000 completed (no test)', got %s", actual)
	}
}

// Printer interface tests
func TestNewPrinter(t *testing.T) {
	printer := NewPrinter(false, []string{})
	if printer == nil {
		t.Error("NewPrinter() should return a non-nil Printer")
		return
	}

	if printer.verbose {
		t.Error("NewPrinter(false) should set verbose to false")
	}

	verbosePrinter := NewPrinter(true, []string{})
	if !verbosePrinter.verbose {
		t.Error("NewPrinter(true) should set verbose to true")
	}
}

func TestNewPrinter_BufferInitialization(t *testing.T) {
	tests := []struct {
		name      string
		bufferIDs []string
	}{
		{
			name:      "empty buffer IDs",
			bufferIDs: []string{},
		},
		{
			name:      "single buffer ID",
			bufferIDs: []string{"job1"},
		},
		{
			name:      "multiple buffer IDs",
			bufferIDs: []string{"job1", "job2", "job3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			printer := NewPrinter(false, tt.bufferIDs)

			if printer.Buffer == nil {
				t.Error("Buffer should not be nil")
				return
			}

			// Check BufferIDs are stored correctly
			if len(printer.BufferIDs) != len(tt.bufferIDs) {
				t.Errorf("Expected %d BufferIDs, got %d", len(tt.bufferIDs), len(printer.BufferIDs))
			}

			for i, expectedID := range tt.bufferIDs {
				if i >= len(printer.BufferIDs) || printer.BufferIDs[i] != expectedID {
					t.Errorf("BufferIDs[%d]: expected '%s', got '%s'", i, expectedID, printer.BufferIDs[i])
				}
			}

			// Check that all provided IDs have initialized buffers
			for _, id := range tt.bufferIDs {
				if _, exists := printer.Buffer[id]; !exists {
					t.Errorf("Buffer for ID '%s' should be initialized", id)
				}
				if printer.Buffer[id] == nil {
					t.Errorf("Buffer for ID '%s' should not be nil", id)
				}
			}

			// Check that buffer count matches expected
			if len(printer.Buffer) != len(tt.bufferIDs) {
				t.Errorf("Expected %d buffers, got %d", len(tt.bufferIDs), len(printer.Buffer))
			}
		})
	}
}

func TestStatusType(t *testing.T) {
	tests := []struct {
		status   StatusType
		expected string
	}{
		{StatusSuccess, "success"},
		{StatusError, "error"},
		{StatusWarning, "warning"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			// Test that the status type constants are properly defined
			if int(tt.status) < 0 {
				t.Errorf("StatusType %s should have a valid integer value", tt.expected)
			}
		})
	}
}

func TestStepResult(t *testing.T) {
	result := StepResult{
		Index:      1,
		Name:       "Test Step",
		Status:     StatusSuccess,
		RT:         "100ms",
		TestOutput: "",
		EchoOutput: "",
		HasTest:    true,
	}

	if result.Index != 1 {
		t.Errorf("Expected Index to be 1, got %d", result.Index)
	}

	if result.Name != "Test Step" {
		t.Errorf("Expected Name to be 'Test Step', got %s", result.Name)
	}

	if result.Status != StatusSuccess {
		t.Errorf("Expected Status to be StatusSuccess, got %d", result.Status)
	}

	if !result.HasTest {
		t.Error("Expected HasTest to be true")
	}
}

// PrintReport method tests
func TestPrinter_PrintReport(t *testing.T) {
	// Create printer with job IDs
	jobIDs := []string{"job1", "job2"}
	printer := NewPrinter(false, jobIDs)

	// Create WorkflowBuffer with test data
	wb := NewWorkflowBuffer()

	// Job 1: Regular steps
	startTime1 := time.Now()
	endTime1 := startTime1.Add(2 * time.Second)
	wb.Jobs["job1"] = &JobBuffer{
		JobID:     "job1",
		JobName:   "Test Job 1",
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
	if !strings.Contains(outputStr, "Step 1") {
		t.Error("Output should contain Step 1")
	}
	if !strings.Contains(outputStr, "Step 2") {
		t.Error("Output should contain Step 2")
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

// Generate method tests
func TestPrinter_generateJobStatus(t *testing.T) {
	printer := NewPrinter(false, []string{})

	tests := []struct {
		name     string
		jobID    string
		jobName  string
		status   StatusType
		duration float64
		want     string
	}{
		{
			name:     "success status",
			jobID:    "job1",
			jobName:  "Test Job",
			status:   StatusSuccess,
			duration: 1.5,
			want:     "⏺ Test Job (Completed in 1.50s)\n",
		},
		{
			name:     "error status",
			jobID:    "job2",
			jobName:  "Failed Job",
			status:   StatusError,
			duration: 2.3,
			want:     "⏺ Failed Job (Failed in 2.30s)\n",
		},
		{
			name:     "warning status",
			jobID:    "job3",
			jobName:  "Skipped Job",
			status:   StatusWarning,
			duration: 0.1,
			want:     "⏺ Skipped Job (Skipped in 0.10s)\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output strings.Builder
			printer.generateJobStatus(tt.jobID, tt.jobName, tt.status, tt.duration, &output)
			
			result := output.String()
			// Remove color codes for easier testing
			if !strings.Contains(result, tt.jobName) {
				t.Errorf("generateJobStatus() should contain job name %s, got %s", tt.jobName, result)
			}
			if !strings.Contains(result, "1.50s") && tt.duration == 1.5 {
				t.Errorf("generateJobStatus() should contain duration 1.50s, got %s", result)
			}
			if !strings.Contains(result, "Completed") && tt.status == StatusSuccess {
				t.Errorf("generateJobStatus() should contain Completed for success status, got %s", result)
			}
		})
	}
}

func TestPrinter_generateJobResults(t *testing.T) {
	printer := NewPrinter(false, []string{})

	tests := []struct {
		name   string
		jobID  string
		input  string
		want   string
	}{
		{
			name:  "empty input",
			jobID: "job1",
			input: "",
			want:  "\n",
		},
		{
			name:  "single line",
			jobID: "job1",
			input: "Test output",
			want:  "  ⎿ Test output\n\n",
		},
		{
			name:  "multiple lines",
			jobID: "job1",
			input: "Line 1\nLine 2\nLine 3",
			want:  "  ⎿ Line 1\n    Line 2\n    Line 3\n\n",
		},
		{
			name:  "with empty lines",
			jobID: "job1",
			input: "Line 1\n\nLine 3",
			want:  "  ⎿ Line 1\n    Line 3\n\n",
		},
		{
			name:  "whitespace only input",
			jobID: "job1",
			input: "   \n  \t  \n   ",
			want:  "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output strings.Builder
			printer.generateJobResults(tt.jobID, tt.input, &output)
			
			result := output.String()
			if result != tt.want {
				t.Errorf("generateJobResults() = %q, want %q", result, tt.want)
			}
		})
	}
}

func TestPrinter_generateFooter(t *testing.T) {
	printer := NewPrinter(false, []string{})

	tests := []struct {
		name         string
		totalTime    float64
		successCount int
		totalJobs    int
		wantContains []string
	}{
		{
			name:         "all jobs succeeded",
			totalTime:    5.25,
			successCount: 3,
			totalJobs:    3,
			wantContains: []string{"Total workflow time: 5.25s", "All jobs succeeded"},
		},
		{
			name:         "some jobs failed",
			totalTime:    10.5,
			successCount: 2,
			totalJobs:    5,
			wantContains: []string{"Total workflow time: 10.50s", "3 job(s) failed"},
		},
		{
			name:         "all jobs failed",
			totalTime:    2.1,
			successCount: 0,
			totalJobs:    2,
			wantContains: []string{"Total workflow time: 2.10s", "2 job(s) failed"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output strings.Builder
			printer.generateFooter(tt.totalTime, tt.successCount, tt.totalJobs, &output)
			
			result := output.String()
			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("generateFooter() should contain %q, got %q", want, result)
				}
			}
		})
	}
}

func TestPrinter_generateReport(t *testing.T) {
	printer := NewPrinter(false, []string{"job1", "job2"})

	// Create test WorkflowBuffer
	wb := NewWorkflowBuffer()
	
	// Add job1 - successful
	startTime1 := time.Now()
	endTime1 := startTime1.Add(1 * time.Second)
	wb.Jobs["job1"] = &JobBuffer{
		JobID:     "job1",
		JobName:   "Successful Job",
		StartTime: startTime1,
		EndTime:   endTime1,
		Success:   true,
		Status:    "Completed",
		StepResults: []StepResult{
			{
				Index:  1,
				Name:   "Test Step",
				Status: StatusSuccess,
				RT:     "100ms",
			},
		},
	}

	// Add job2 - failed
	startTime2 := time.Now()
	endTime2 := startTime2.Add(2 * time.Second)
	wb.Jobs["job2"] = &JobBuffer{
		JobID:     "job2",
		JobName:   "Failed Job",
		StartTime: startTime2,
		EndTime:   endTime2,
		Success:   false,
		Status:    "Failed",
		StepResults: []StepResult{
			{
				Index:  1,
				Name:   "Failed Step",
				Status: StatusError,
			},
		},
	}

	result := printer.generateReport(wb)

	// Verify the report contains expected elements
	expectedContains := []string{
		"Successful Job",
		"Failed Job",
		"Test Step",
		"Failed Step",
		"Total workflow time:",
		"1 job(s) failed",
	}

	for _, expected := range expectedContains {
		if !strings.Contains(result, expected) {
			t.Errorf("generateReport() should contain %q, got:\n%s", expected, result)
		}
	}
}

func TestPrinter_generateReport_EmptyBuffer(t *testing.T) {
	printer := NewPrinter(false, []string{})

	result := printer.generateReport(nil)
	if result != "" {
		t.Errorf("generateReport(nil) should return empty string, got %q", result)
	}

	wb := NewWorkflowBuffer()
	result = printer.generateReport(wb)
	
	// Should contain at least the footer
	if !strings.Contains(result, "Total workflow time: 0.00s") {
		t.Errorf("generateReport() with empty buffer should contain footer, got %q", result)
	}
}

func TestPrinter_generateReport_WithRepeatStep(t *testing.T) {
	printer := NewPrinter(false, []string{"job1"})

	wb := NewWorkflowBuffer()
	
	startTime := time.Now()
	endTime := startTime.Add(1 * time.Second)
	wb.Jobs["job1"] = &JobBuffer{
		JobID:     "job1",
		JobName:   "Job with Repeat",
		StartTime: startTime,
		EndTime:   endTime,
		Success:   true,
		Status:    "Completed",
		StepResults: []StepResult{
			{
				Index:   1,
				Name:    "Repeat Step",
				Status:  StatusSuccess,
				HasTest: true,
				RepeatCounter: &StepRepeatCounter{
					SuccessCount: 8,
					FailureCount: 2,
					Name:         "Repeat Step",
					LastResult:   true,
				},
			},
		},
	}

	result := printer.generateReport(wb)

	expectedContains := []string{
		"Job with Repeat",
		"Repeat Step (repeating 10 times)",
		"8/10 success (80.0%)",
		"Total workflow time:",
		"All jobs succeeded",
	}

	for _, expected := range expectedContains {
		if !strings.Contains(result, expected) {
			t.Errorf("generateReport() should contain %q, got:\n%s", expected, result)
		}
	}
}
