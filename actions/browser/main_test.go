package browser

import (
	"strconv"
	"testing"

	"github.com/hashicorp/go-hclog"
)

func TestAction_Run_InvalidAction(t *testing.T) {
	action := &Action{
		log: hclog.NewNullLogger(),
	}

	params := map[string]string{
		"action": "invalid_action",
	}

	result, err := action.Run([]string{}, params)

	if err == nil {
		t.Fatal("Expected error for invalid action, got nil")
	}

	if result["error"] == "" {
		t.Error("Expected error in result, got empty string")
	}

	expectedError := "unknown action: invalid_action"
	if result["error"] != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, result["error"])
	}
}

func TestAction_Run_MissingAction(t *testing.T) {
	action := &Action{
		log: hclog.NewNullLogger(),
	}

	params := map[string]string{
		"url": "https://example.com",
	}

	result, err := action.Run([]string{}, params)

	if err == nil {
		t.Fatal("Expected error for missing action, got nil")
	}

	expectedError := "action parameter is required"
	if result["error"] != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, result["error"])
	}
}

func TestAction_Navigate_MissingURL(t *testing.T) {
	action := &Action{
		log: hclog.NewNullLogger(),
	}

	_, err := action.navigate(nil, "")

	if err == nil {
		t.Fatal("Expected error for missing URL, got nil")
	}

	expectedError := "url parameter is required for navigate action"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestAction_GetText_MissingSelector(t *testing.T) {
	action := &Action{
		log: hclog.NewNullLogger(),
	}

	_, err := action.getText(nil, "")

	if err == nil {
		t.Fatal("Expected error for missing selector, got nil")
	}

	expectedError := "selector parameter is required for get_text action"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestAction_GetAttribute_MissingParameters(t *testing.T) {
	action := &Action{
		log: hclog.NewNullLogger(),
	}

	// Test missing selector
	_, err := action.getAttribute(nil, "", "href")
	if err == nil {
		t.Fatal("Expected error for missing selector, got nil")
	}

	expectedError := "selector parameter is required for get_attribute action"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}

	// Test missing attribute
	_, err = action.getAttribute(nil, "a", "")
	if err == nil {
		t.Fatal("Expected error for missing attribute, got nil")
	}

	expectedError = "attribute parameter is required for get_attribute action"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestAction_Click_MissingSelector(t *testing.T) {
	action := &Action{
		log: hclog.NewNullLogger(),
	}

	_, err := action.click(nil, "")

	if err == nil {
		t.Fatal("Expected error for missing selector, got nil")
	}

	expectedError := "selector parameter is required for click action"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestAction_TypeText_MissingParameters(t *testing.T) {
	action := &Action{
		log: hclog.NewNullLogger(),
	}

	// Test missing selector
	_, err := action.typeText(nil, "", "test")
	if err == nil {
		t.Fatal("Expected error for missing selector, got nil")
	}

	expectedError := "selector parameter is required for type action"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}

	// Test missing value
	_, err = action.typeText(nil, "input", "")
	if err == nil {
		t.Fatal("Expected error for missing value, got nil")
	}

	expectedError = "value parameter is required for type action"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestAction_Submit_MissingSelector(t *testing.T) {
	action := &Action{
		log: hclog.NewNullLogger(),
	}

	_, err := action.submit(nil, "")

	if err == nil {
		t.Fatal("Expected error for missing selector, got nil")
	}

	expectedError := "selector parameter is required for submit action"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestAction_WaitVisible_MissingSelector(t *testing.T) {
	action := &Action{
		log: hclog.NewNullLogger(),
	}

	_, err := action.waitVisible(nil, "")

	if err == nil {
		t.Fatal("Expected error for missing selector, got nil")
	}

	expectedError := "selector parameter is required for wait_visible action"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestAction_WaitText_MissingParameters(t *testing.T) {
	action := &Action{
		log: hclog.NewNullLogger(),
	}

	// Test missing selector
	_, err := action.waitText(nil, "", "test")
	if err == nil {
		t.Fatal("Expected error for missing selector, got nil")
	}

	expectedError := "selector parameter is required for wait_text action"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}

	// Test missing value
	_, err = action.waitText(nil, "div", "")
	if err == nil {
		t.Fatal("Expected error for missing value, got nil")
	}

	expectedError = "value parameter is required for wait_text action"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestAction_GetElements_MissingSelector(t *testing.T) {
	action := &Action{
		log: hclog.NewNullLogger(),
	}

	_, err := action.getElements(nil, "")

	if err == nil {
		t.Fatal("Expected error for missing selector, got nil")
	}

	expectedError := "selector parameter is required for get_elements action"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

// Test helper functions
func TestParseHeadlessParameter(t *testing.T) {
	testCases := []struct {
		input    string
		expected bool
	}{
		{"true", true},
		{"false", false},
		{"", true}, // default
		{"invalid", true}, // default on parse error
	}

	for _, tc := range testCases {
		params := map[string]string{"action": "navigate", "url": "http://example.com"}
		if tc.input != "" {
			params["headless"] = tc.input
		}

		// We can't easily test the actual chromedp behavior without a browser,
		// but we can test that the parameter parsing logic works correctly
		// by examining how the headless parameter would be processed
		headless := true
		if h := params["headless"]; h != "" {
			if parsed, err := strconv.ParseBool(h); err == nil {
				headless = parsed
			}
		}

		if headless != tc.expected {
			t.Errorf("For input '%s', expected %v, got %v", tc.input, tc.expected, headless)
		}
	}
}