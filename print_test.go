package probe

import (
	"io"
	"os"
	"strings"
	"testing"

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

	actual := colorWarning().Sprintf("⏺") + " " +
		colorWarning().Sprintf("%d/%d completed (no test)", totalCount, totalCount)

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

func TestPrintBuffer_OrderPreservation(t *testing.T) {
	// Test that PrintBuffer outputs in the order specified by BufferIDs
	bufferIDs := []string{"job3", "job1", "job2"} // Intentionally out of alphabetical order
	printer := NewPrinter(false, bufferIDs)

	// Add content to buffers in different order
	printer.appendToBuffer("job1", "Content from job1\n")
	printer.appendToBuffer("job2", "Content from job2\n")
	printer.appendToBuffer("job3", "Content from job3\n")

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printer.PrintBuffer()

	w.Close()
	os.Stdout = oldStdout

	output, _ := io.ReadAll(r)
	outputStr := string(output)

	expectedOutput := "Content from job3\nContent from job1\nContent from job2\n"
	if outputStr != expectedOutput {
		t.Errorf("Expected output:\n%s\nGot output:\n%s", expectedOutput, outputStr)
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
