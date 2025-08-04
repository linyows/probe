package probe

import (
	"fmt"
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
		{
			name:     "colorSkipped",
			colorFn:  colorSkipped,
			text:     "skipped",
			expected: "skipped",
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
		{StatusSkipped, "skipped"},
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

// Test StatusType constants have expected values
func TestStatusTypeConstants(t *testing.T) {
	// Test that status constants are assigned in expected order
	if StatusSuccess != 0 {
		t.Errorf("StatusSuccess should be 0, got %d", StatusSuccess)
	}
	if StatusError != 1 {
		t.Errorf("StatusError should be 1, got %d", StatusError)
	}
	if StatusWarning != 2 {
		t.Errorf("StatusWarning should be 2, got %d", StatusWarning)
	}
	if StatusSkipped != 3 {
		t.Errorf("StatusSkipped should be 3, got %d", StatusSkipped)
	}

	// Test that all constants are different
	statuses := []StatusType{StatusSuccess, StatusError, StatusWarning, StatusSkipped}
	for i, status1 := range statuses {
		for j, status2 := range statuses {
			if i != j && status1 == status2 {
				t.Errorf("StatusType constants should be unique, but %d and %d are both %d", i, j, status1)
			}
		}
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

// Generate method tests
func TestPrinter_generateHeader(t *testing.T) {
	// Disable color output for consistent testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	printer := NewPrinter(false, []string{})

	tests := []struct {
		name        string
		title       string
		description string
		want        string
	}{
		{
			name:        "empty name",
			title:       "",
			description: "Some description",
			want:        "",
		},
		{
			name:        "name only",
			title:       "Test Workflow",
			description: "",
			want:        "Test Workflow\n\n",
		},
		{
			name:        "name and description",
			title:       "Test Workflow",
			description: "This is a test workflow",
			want:        "Test Workflow\nThis is a test workflow\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := printer.generateHeader(tt.title, tt.description)
			if result != tt.want {
				t.Errorf("generateHeader() = %q, want %q", result, tt.want)
			}
		})
	}
}

func TestPrinter_generateError(t *testing.T) {
	// Disable color output for consistent testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	printer := NewPrinter(false, []string{})

	tests := []struct {
		name   string
		format string
		args   []interface{}
		want   string
	}{
		{
			name:   "simple error",
			format: "Something went wrong",
			args:   []interface{}{},
			want:   "Error: Something went wrong\n",
		},
		{
			name:   "formatted error",
			format: "Failed to process %s with code %d",
			args:   []interface{}{"file.txt", 404},
			want:   "Error: Failed to process file.txt with code 404\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := printer.generateError(tt.format, tt.args...)
			if result != tt.want {
				t.Errorf("generateError() = %q, want %q", result, tt.want)
			}
		})
	}
}

func TestPrinter_generateLogDebug(t *testing.T) {
	printer := NewPrinter(false, []string{})

	tests := []struct {
		name   string
		format string
		args   []interface{}
		want   string
	}{
		{
			name:   "simple debug",
			format: "Debug message",
			args:   []interface{}{},
			want:   "[DEBUG] Debug message\n",
		},
		{
			name:   "formatted debug",
			format: "Processing item %d of %d",
			args:   []interface{}{5, 10},
			want:   "[DEBUG] Processing item 5 of 10\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := printer.generateLogDebug(tt.format, tt.args...)
			if result != tt.want {
				t.Errorf("generateLogDebug() = %q, want %q", result, tt.want)
			}
		})
	}
}

func TestPrinter_generateLogError(t *testing.T) {
	// Disable color output for consistent testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	printer := NewPrinter(false, []string{})

	tests := []struct {
		name   string
		format string
		args   []interface{}
		want   string
	}{
		{
			name:   "simple log error",
			format: "Critical error occurred",
			args:   []interface{}{},
			want:   "[ERROR] Critical error occurred\n",
		},
		{
			name:   "formatted log error",
			format: "Database connection failed: %s",
			args:   []interface{}{"timeout"},
			want:   "[ERROR] Database connection failed: timeout\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := printer.generateLogError(tt.format, tt.args...)
			if result != tt.want {
				t.Errorf("generateLogError() = %q, want %q", result, tt.want)
			}
		})
	}
}
func TestPrinter_generateJobStatus(t *testing.T) {
	// Disable color output for consistent testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

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
			name:     "skipped status",
			jobID:    "job3",
			jobName:  "Skipped Job",
			status:   StatusSkipped,
			duration: 0.0,
			want:     "⏺ Skipped Job (SKIPPED)\n",
		},
		{
			name:     "warning status",
			jobID:    "job4",
			jobName:  "Warning Job",
			status:   StatusWarning,
			duration: 0.1,
			want:     "⏺ Warning Job (Unknown status in 0.10s)\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output strings.Builder
			printer.generateJobStatus(tt.jobID, tt.jobName, tt.status, tt.duration, &output)

			result := output.String()
			if result != tt.want {
				t.Errorf("generateJobStatus() = %q, want %q", result, tt.want)
			}
		})
	}
}

// Test that job status uses correct colors for each status type - with actual color detection
func TestPrinter_generateJobStatus_ColorMapping(t *testing.T) {
	// Enable colors to detect actual color differences
	color.NoColor = false
	defer func() { color.NoColor = true }()

	printer := NewPrinter(false, []string{})

	tests := []struct {
		name              string
		status            StatusType
		expectedColorCode string // ANSI color code we expect
	}{
		{
			name:              "success uses green color",
			status:            StatusSuccess,
			expectedColorCode: "\x1b[38;2;0;175;0m", // RGB(0,175,0) from colorSuccess
		},
		{
			name:              "error uses red color", 
			status:            StatusError,
			expectedColorCode: "\x1b[31m", // Red from colorError
		},
		{
			name:              "skipped uses gray color",
			status:            StatusSkipped,
			expectedColorCode: "\x1b[90m", // Bright black (gray) from colorSkipped
		},
		{
			name:              "warning uses yellow color",
			status:            StatusWarning,
			expectedColorCode: "\x1b[33m", // Yellow from colorWarning
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output strings.Builder
			printer.generateJobStatus("test-job", "Test Job", tt.status, 1.0, &output)

			result := output.String()
			
			// Verify basic content is present
			if !strings.Contains(result, "Test Job") {
				t.Errorf("generateJobStatus() should contain job name, got %q", result)
			}
			
			// Verify the expected color code is present in the output
			if !strings.Contains(result, tt.expectedColorCode) {
				t.Errorf("generateJobStatus() should contain color code %q for %s, got %q", tt.expectedColorCode, tt.name, result)
			}
		})
	}
}

// Test that SUCCESS status specifically does NOT use blue color (colorInfo)
func TestPrinter_generateJobStatus_SuccessNotBlue(t *testing.T) {
	// Enable colors to detect actual color differences
	color.NoColor = false
	defer func() { color.NoColor = true }()

	printer := NewPrinter(false, []string{})

	var output strings.Builder
	printer.generateJobStatus("success-job", "Success Job", StatusSuccess, 1.5, &output)

	result := output.String()
	
	// Blue color code from colorInfo
	blueColorCode := "\x1b[34m"
	
	// SUCCESS jobs should NOT contain blue color
	if strings.Contains(result, blueColorCode) {
		t.Errorf("StatusSuccess should NOT use blue color (colorInfo), but found blue color code in output: %q", result)
	}
	
	// SUCCESS jobs SHOULD contain green color
	greenColorCode := "\x1b[38;2;0;175;0m" // RGB(0,175,0) from colorSuccess
	if !strings.Contains(result, greenColorCode) {
		t.Errorf("StatusSuccess should use green color (colorSuccess), but green color code not found in output: %q", result)
	}
}

// Test that skipped jobs have gray formatting for entire line
func TestPrinter_generateJobStatus_SkippedFormatting(t *testing.T) {
	// Disable color output for consistent testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	printer := NewPrinter(false, []string{})

	// Test skipped job formatting
	var output strings.Builder
	printer.generateJobStatus("skip-job", "Skipped Job", StatusSkipped, 0.0, &output)

	result := output.String()
	expected := "⏺ Skipped Job (SKIPPED)\n"

	if result != expected {
		t.Errorf("generateJobStatus() for skipped job = %q, want %q", result, expected)
	}

	// Test non-skipped job formatting for comparison
	var output2 strings.Builder
	printer.generateJobStatus("success-job", "Success Job", StatusSuccess, 1.5, &output2)

	result2 := output2.String()
	expected2 := "⏺ Success Job (Completed in 1.50s)\n"

	if result2 != expected2 {
		t.Errorf("generateJobStatus() for success job = %q, want %q", result2, expected2)
	}
}

func TestPrinter_generateJobResults(t *testing.T) {
	printer := NewPrinter(false, []string{})

	tests := []struct {
		name  string
		jobID string
		input string
		want  string
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
	rs := NewResult()

	// Add job1 - successful
	startTime1 := time.Now()
	endTime1 := startTime1.Add(1 * time.Second)
	rs.Jobs["job1"] = &JobResult{
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
	rs.Jobs["job2"] = &JobResult{
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

	result := printer.generateReport(rs)

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

	rs := NewResult()
	result = printer.generateReport(rs)

	// Should contain at least the footer
	if !strings.Contains(result, "Total workflow time: 0.00s") {
		t.Errorf("generateReport() with empty buffer should contain footer, got %q", result)
	}
}

func TestPrinter_generateReport_WithRepeatStep(t *testing.T) {
	printer := NewPrinter(false, []string{"job1"})

	rs := NewResult()

	startTime := time.Now()
	endTime := startTime.Add(1 * time.Second)
	rs.Jobs["job1"] = &JobResult{
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

	result := printer.generateReport(rs)

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

// Truncate function tests
func TestGetTruncationMessage(t *testing.T) {
	// Disable color output for consistent testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	result := GetTruncationMessage()
	expected := "... [⚠︎ probe truncated]"

	if result != expected {
		t.Errorf("GetTruncationMessage() = %q, want %q", result, expected)
	}
}

func TestTruncateString(t *testing.T) {
	// Disable color output for consistent testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "short string",
			input:    "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "exact length",
			input:    "hello",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "long string",
			input:    "this is a very long string that exceeds the limit",
			maxLen:   10,
			expected: "this is a ... [⚠︎ probe truncated]",
		},
		{
			name:     "empty string",
			input:    "",
			maxLen:   5,
			expected: "",
		},
		{
			name:     "zero max length",
			input:    "hello",
			maxLen:   0,
			expected: "... [⚠︎ probe truncated]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateString(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("TruncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}

func TestTruncateMapStringString(t *testing.T) {
	// Disable color output for consistent testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	tests := []struct {
		name     string
		input    map[string]string
		maxLen   int
		expected map[string]string
	}{
		{
			name: "short values",
			input: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			maxLen: 10,
			expected: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name: "mixed length values",
			input: map[string]string{
				"short": "abc",
				"long":  "this is a very long string that will be truncated",
			},
			maxLen: 10,
			expected: map[string]string{
				"short": "abc",
				"long":  "this is a ... [⚠︎ probe truncated]",
			},
		},
		{
			name: "all long values",
			input: map[string]string{
				"url":  "https://example.com/very/long/path/that/exceeds/the/limit",
				"body": "this is a very long request body that contains lots of data",
			},
			maxLen: 15,
			expected: map[string]string{
				"url":  "https://example... [⚠︎ probe truncated]",
				"body": "this is a very ... [⚠︎ probe truncated]",
			},
		},
		{
			name:     "empty map",
			input:    map[string]string{},
			maxLen:   10,
			expected: map[string]string{},
		},
		{
			name: "zero max length",
			input: map[string]string{
				"key": "value",
			},
			maxLen: 0,
			expected: map[string]string{
				"key": "... [⚠︎ probe truncated]",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateMapStringString(tt.input, tt.maxLen)

			if len(result) != len(tt.expected) {
				t.Errorf("TruncateMapStringString() returned map with %d keys, expected %d", len(result), len(tt.expected))
			}

			for key, expectedValue := range tt.expected {
				actualValue, exists := result[key]
				if !exists {
					t.Errorf("TruncateMapStringString() missing key %q", key)
					continue
				}
				if actualValue != expectedValue {
					t.Errorf("TruncateMapStringString() key %q = %q, want %q", key, actualValue, expectedValue)
				}
			}
		})
	}
}

func TestMaxLogStringLength(t *testing.T) {
	// Test that the constant is properly defined
	if MaxLogStringLength <= 0 {
		t.Errorf("MaxLogStringLength should be positive, got %d", MaxLogStringLength)
	}

	// Test that it has a reasonable value (expected to be 200)
	expectedValue := 200
	if MaxLogStringLength != expectedValue {
		t.Errorf("MaxLogStringLength = %d, expected %d", MaxLogStringLength, expectedValue)
	}
}

func TestMaxStringLength(t *testing.T) {
	// Test that the constant is properly defined
	if MaxStringLength <= 0 {
		t.Errorf("MaxStringLength should be positive, got %d", MaxStringLength)
	}

	// Test that it has a reasonable value (expected to be 1000000)
	expectedValue := 1000000
	if MaxStringLength != expectedValue {
		t.Errorf("MaxStringLength = %d, expected %d", MaxStringLength, expectedValue)
	}
}

// Tests for new Step Output Formatting Functions

func TestPrinter_generateEchoOutput(t *testing.T) {
	// Disable color output for consistent testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	printer := NewPrinter(false, []string{})

	tests := []struct {
		name     string
		content  string
		err      error
		expected string
	}{
		{
			name:     "single line content",
			content:  "Hello World",
			err:      nil,
			expected: "       Hello World\n",
		},
		{
			name:     "multi-line content with explicit newlines",
			content:  "Line 1\nLine 2\nLine 3",
			err:      nil,
			expected: "       Line 1\n       Line 2\n       Line 3\n",
		},
		{
			name:     "complex multiline with indentation",
			content:  "Header\n  Indented\n    More indented\nBack to left",
			err:      nil,
			expected: "       Header\n         Indented\n           More indented\n       Back to left\n",
		},
		{
			name:     "empty line handling",
			content:  "Line 1\n\nLine 3",
			err:      nil,
			expected: "       Line 1\n       \n       Line 3\n",
		},
		{
			name:     "error case",
			content:  "",
			err:      fmt.Errorf("template error"),
			expected: "Echo\nerror: &errors.errorString{s:\"template error\"}\n",
		},
		{
			name:     "empty content",
			content:  "",
			err:      nil,
			expected: "       \n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := printer.generateEchoOutput(tt.content, tt.err)
			if result != tt.expected {
				t.Errorf("generateEchoOutput() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestPrinter_generateTestFailure(t *testing.T) {
	// Disable color output for consistent testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	printer := NewPrinter(false, []string{})

	tests := []struct {
		name     string
		testExpr string
		result   interface{}
		req      map[string]any
		res      map[string]any
		expected string
	}{
		{
			name:     "simple test failure",
			testExpr: "res.status == 200",
			result:   false,
			req:      map[string]any{"method": "GET", "url": "http://example.com"},
			res:      map[string]any{"status": 404, "body": "Not Found"},
			expected: "       request: map[string]interface {}{\"method\":\"GET\", \"url\":\"http://example.com\"}\n       response: map[string]interface {}{\"body\":\"Not Found\", \"status\":404}\n",
		},
		{
			name:     "empty maps",
			testExpr: "true",
			result:   false,
			req:      map[string]any{},
			res:      map[string]any{},
			expected: "       request: map[string]interface {}{}\n       response: map[string]interface {}{}\n",
		},
		{
			name:     "nil maps",
			testExpr: "false",
			result:   false,
			req:      nil,
			res:      nil,
			expected: "       request: map[string]interface {}(nil)\n       response: map[string]interface {}(nil)\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := printer.generateTestFailure(tt.testExpr, tt.result, tt.req, tt.res)
			if result != tt.expected {
				t.Errorf("generateTestFailure() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestPrinter_generateTestError(t *testing.T) {
	printer := NewPrinter(false, []string{})

	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "simple error",
			err:      fmt.Errorf("compilation error"),
			expected: "Test\nerror: &errors.errorString{s:\"compilation error\"}\n",
		},
		{
			name:     "complex error message",
			err:      fmt.Errorf("invalid expression: res.status == \"200\""),
			expected: "Test\nerror: &errors.errorString{s:\"invalid expression: res.status == \\\"200\\\"\"}\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := printer.generateTestError("test expression", tt.err)
			if result != tt.expected {
				t.Errorf("generateTestError() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestPrinter_generateTestTypeMismatch(t *testing.T) {
	printer := NewPrinter(false, []string{})

	tests := []struct {
		name     string
		testExpr string
		result   interface{}
		expected string
	}{
		{
			name:     "string result instead of bool",
			testExpr: "res.status",
			result:   "200",
			expected: "Test: `res.status` = 200\n",
		},
		{
			name:     "number result instead of bool",
			testExpr: "res.code",
			result:   404,
			expected: "Test: `res.code` = 404\n",
		},
		{
			name:     "map result instead of bool",
			testExpr: "res.body",
			result:   map[string]any{"error": "not found"},
			expected: "Test: `res.body` = map[error:not found]\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := printer.generateTestTypeMismatch(tt.testExpr, tt.result)
			if result != tt.expected {
				t.Errorf("generateTestTypeMismatch() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestPrinter_PrintTestResult(t *testing.T) {
	// Disable color output for consistent testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	tests := []struct {
		name      string
		success   bool
		testExpr  string
		context   interface{}
		verbose   bool
		expectLog bool
	}{
		{
			name:      "successful test in verbose mode",
			success:   true,
			testExpr:  "res.status == 200",
			context:   map[string]any{"status": 200},
			verbose:   true,
			expectLog: true,
		},
		{
			name:      "failed test in verbose mode",
			success:   false,
			testExpr:  "res.status == 200",
			context:   map[string]any{"status": 404},
			verbose:   true,
			expectLog: true,
		},
		{
			name:      "test in non-verbose mode",
			success:   true,
			testExpr:  "res.status == 200",
			context:   map[string]any{"status": 200},
			verbose:   false,
			expectLog: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			printer := NewPrinter(tt.verbose, []string{})

			// This method prints debug output, we mainly test it doesn't panic
			// and the method signature is correct
			printer.PrintTestResult(tt.success, tt.testExpr, tt.context)

			// Test passes if no panic occurs
		})
	}
}

func TestPrinter_PrintEchoContent(t *testing.T) {
	tests := []struct {
		name    string
		content string
		verbose bool
	}{
		{
			name:    "single line in verbose mode",
			content: "Hello World",
			verbose: true,
		},
		{
			name:    "multi-line in verbose mode",
			content: "Line 1\nLine 2\nLine 3",
			verbose: true,
		},
		{
			name:    "content in non-verbose mode",
			content: "Hello World",
			verbose: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			printer := NewPrinter(tt.verbose, []string{})

			// This method prints debug output, we mainly test it doesn't panic
			printer.PrintEchoContent(tt.content)

			// Test passes if no panic occurs
		})
	}
}

func TestPrinter_PrintRequestResponse(t *testing.T) {
	// Disable color output for consistent testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	tests := []struct {
		name     string
		stepIdx  int
		stepName string
		req      map[string]any
		res      map[string]any
		rt       string
		verbose  bool
	}{
		{
			name:     "simple request response in verbose mode",
			stepIdx:  1,
			stepName: "Test Step",
			req:      map[string]any{"method": "GET", "url": "http://example.com"},
			res:      map[string]any{"status": 200, "body": "OK"},
			rt:       "123ms",
			verbose:  true,
		},
		{
			name:     "nested data in verbose mode",
			stepIdx:  2,
			stepName: "Complex Step",
			req:      map[string]any{"headers": map[string]any{"Accept": "application/json"}},
			res:      map[string]any{"data": map[string]any{"id": 123, "name": "test"}},
			rt:       "456ms",
			verbose:  true,
		},
		{
			name:     "request response in non-verbose mode",
			stepIdx:  1,
			stepName: "Test Step",
			req:      map[string]any{"method": "GET"},
			res:      map[string]any{"status": 200},
			rt:       "100ms",
			verbose:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			printer := NewPrinter(tt.verbose, []string{})

			// This method prints debug output, we mainly test it doesn't panic
			printer.PrintRequestResponse(tt.stepIdx, tt.stepName, tt.req, tt.res, tt.rt)

			// Test passes if no panic occurs
		})
	}
}

func TestPrinter_PrintMapData(t *testing.T) {
	tests := []struct {
		name    string
		data    map[string]any
		verbose bool
	}{
		{
			name: "simple map data",
			data: map[string]any{
				"key1": "value1",
				"key2": 123,
				"key3": true,
			},
			verbose: true,
		},
		{
			name: "nested map data",
			data: map[string]any{
				"simple": "value",
				"nested": map[string]any{
					"inner1": "value1",
					"inner2": 456,
				},
			},
			verbose: true,
		},
		{
			name:    "empty map",
			data:    map[string]any{},
			verbose: true,
		},
		{
			name: "non-verbose mode",
			data: map[string]any{
				"key": "value",
			},
			verbose: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			printer := NewPrinter(tt.verbose, []string{})

			// This method prints debug output, we mainly test it doesn't panic
			printer.PrintMapData(tt.data)

			// Test passes if no panic occurs
		})
	}
}
