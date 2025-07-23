package probe

import (
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

// Output interface tests
func TestNewOutput(t *testing.T) {
	output := NewOutput(false)
	if output == nil {
		t.Error("NewOutput() should return a non-nil Output")
		return
	}
	
	if output.verbose {
		t.Error("NewOutput(false) should set verbose to false")
	}
	
	verboseOutput := NewOutput(true)
	if !verboseOutput.verbose {
		t.Error("NewOutput(true) should set verbose to true")
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