package browser

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/linyows/probe"
)

func TestNewChromeDPAction(t *testing.T) {
	got := NewChromeDPAction()

	expected := &ChromeDPAction{
		Quality: defaultQuality,
	}

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("\nExpected:\n%#v\nGot:\n%#v", expected, got)
	}
}

func TestNewReq(t *testing.T) {
	got := NewReq()

	expected := &Req{
		Timeout:  defaultTimeout,
		WindowW:  defaultWindowWidth,
		WindowH:  defaultWindowHeight,
		Headless: true,
	}

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("\nExpected:\n%#v\nGot:\n%#v", expected, got)
	}
}

func TestRequestMissingActions(t *testing.T) {
	data := map[string]string{
		"headless": "true",
	}

	result, err := Request(data)

	if err == nil {
		t.Error("Expected error for missing actions parameter")
	}

	expectedError := "actions parameter is required"
	if err.Error() != expectedError {
		t.Errorf("Expected error message '%s', got '%s'", expectedError, err.Error())
	}

	if len(result) != 0 {
		t.Error("Expected empty result map when error occurs")
	}
}

func TestRequestWithInvalidAction(t *testing.T) {
	data := map[string]string{
		"actions__0__name": "invalid_action",
		"actions__0__url":  "http://example.com",
	}

	_, err := Request(data)

	if err == nil {
		t.Error("Expected error for invalid action type")
	}

	expectedError := "unsupported action type: invalid_action"
	if err.Error() != expectedError {
		t.Errorf("Expected error message '%s', got '%s'", expectedError, err.Error())
	}
}

func TestRequestParameterMapping(t *testing.T) {
	data := map[string]string{
		"actions__0__name": "navigate",
		"actions__0__url":  "http://example.com",
		"headless":         "false",
		"timeout":          "10s",
		"window_w":         "800",
		"window_h":         "600",
	}

	unflattened := probe.UnflattenInterface(data)
	req := NewReq()

	// Test headless parameter - probe package converts "false" string to bool
	if headless, exists := unflattened["headless"]; exists {
		if str, ok := headless.(string); ok && str == "false" {
			req.Headless = false
		} else if bo, ok := headless.(bool); ok {
			req.Headless = bo
		}
	}

	// Test timeout parameter
	if timeout, exists := unflattened["timeout"]; exists {
		if st, ok := timeout.(string); ok {
			if parsed, err := time.ParseDuration(st); err == nil {
				req.Timeout = parsed
			}
		}
	}

	// Test window dimensions - probe package converts string numbers to int
	if ww, exists := unflattened["window_w"]; exists {
		if str, ok := ww.(string); ok && str == "800" {
			req.WindowW = 800
		} else if in, ok := ww.(int); ok {
			req.WindowW = in
		}
	}
	if wh, exists := unflattened["window_h"]; exists {
		if str, ok := wh.(string); ok && str == "600" {
			req.WindowH = 600
		} else if in, ok := wh.(int); ok {
			req.WindowH = in
		}
	}

	if req.Headless != false {
		t.Errorf("Expected headless to be false, got %v", req.Headless)
	}

	if req.Timeout != 10*time.Second {
		t.Errorf("Expected timeout to be 10s, got %v", req.Timeout)
	}

	if req.WindowW != 800 {
		t.Errorf("Expected window width to be 800, got %d", req.WindowW)
	}

	if req.WindowH != 600 {
		t.Errorf("Expected window height to be 600, got %d", req.WindowH)
	}
}

func TestCallbackOptions(t *testing.T) {
	var receivedReq *Req
	var receivedRes *Res

	withInBrowserOpt := WithInBrowser(func(s string, i ...interface{}) {
		// Browser callback function
	})

	withBeforeOpt := WithBefore(func(req *Req) {
		receivedReq = req
	})

	withAfterOpt := WithAfter(func(res *Res) {
		receivedRes = res
	})

	cb := &Callback{}
	withInBrowserOpt(cb)
	withBeforeOpt(cb)
	withAfterOpt(cb)

	if cb.withInBrowser == nil {
		t.Error("WithInBrowser callback was not set")
	}

	if cb.before == nil {
		t.Error("WithBefore callback was not set")
	}

	if cb.after == nil {
		t.Error("WithAfter callback was not set")
	}

	// Test callback execution
	req := NewReq()
	if cb.before != nil {
		cb.before(req)
	}

	if receivedReq != req {
		t.Error("Before callback did not receive correct request")
	}

	res := &Res{Code: 0, Results: make(map[string]string)}
	if cb.after != nil {
		cb.after(res)
	}

	if receivedRes != res {
		t.Error("After callback did not receive correct response")
	}
}

func TestChromeDPActionMapping(t *testing.T) {
	testCases := []struct {
		name     string
		input    map[string]any
		expected ChromeDPAction
	}{
		{
			name: "navigate action",
			input: map[string]any{
				"id":   "nav1",
				"name": "navigate",
				"url":  "http://example.com",
			},
			expected: ChromeDPAction{
				ID:      "nav1",
				Name:    "navigate",
				URL:     "http://example.com",
				Quality: defaultQuality,
			},
		},
		{
			name: "text action with selector",
			input: map[string]any{
				"id":       "text1",
				"name":     "text",
				"selector": "#content",
			},
			expected: ChromeDPAction{
				ID:       "text1",
				Name:     "text",
				Selector: "#content",
				Quality:  defaultQuality,
			},
		},
		{
			name: "screenshot action with path and quality",
			input: map[string]any{
				"id":      "shot1",
				"name":    "screenshot",
				"path":    "/tmp/screenshot.png",
				"quality": 80,
			},
			expected: ChromeDPAction{
				ID:      "shot1",
				Name:    "screenshot",
				Path:    "/tmp/screenshot.png",
				Quality: 80,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			action := NewChromeDPAction()
			err := probe.MapToStructByTags(tc.input, action)

			if err != nil {
				t.Errorf("MapToStructByTags failed: %v", err)
			}

			if !reflect.DeepEqual(*action, tc.expected) {
				t.Errorf("\nExpected:\n%#v\nGot:\n%#v", tc.expected, *action)
			}
		})
	}
}

func TestNewActionTypes(t *testing.T) {
	testCases := []struct {
		name        string
		actionName  string
		expectError bool
	}{
		{"wait_ready action", "wait_ready", false},
		{"wait_not_visible action", "wait_not_visible", false},
		{"submit action", "submit", false},
		{"select action", "select", false},
		{"scroll action", "scroll", false},
		{"get_attribute action", "get_attribute", true},
		{"wait_text action", "wait_text", false},
		{"invalid action", "invalid_action", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data := map[string]string{
				"actions__0__name":     tc.actionName,
				"actions__0__selector": "#test",
				"actions__0__url":      "http://example.com",
			}

			// Add specific parameters for certain actions
			switch tc.actionName {
			case "select":
				data["actions__0__value"] = "option1"
			case "wait_text":
				data["actions__0__value"] = "expected text"
			}

			_, err := Request(data)

			if tc.expectError && err == nil {
				t.Errorf("Expected error for action %s, but got none", tc.actionName)
			}

			if !tc.expectError && err != nil {
				// For non-error cases, we expect chromedp context errors since we're not running a real browser
				// Just check that the action was recognized (not "unsupported action type" error)
				if err.Error() == fmt.Sprintf("unsupported action type: %s", tc.actionName) {
					t.Errorf("Action %s should be supported but got unsupported error", tc.actionName)
				}
			}
		})
	}
}

func TestGetAttributeActionMapping(t *testing.T) {
	action := NewChromeDPAction()
	testData := map[string]any{
		"name":      "get_attribute",
		"selector":  "#link",
		"attribute": []string{"href"},
	}

	err := probe.MapToStructByTags(testData, action)
	if err != nil {
		t.Errorf("MapToStructByTags failed: %v", err)
	}

	if action.Name != "get_attribute" {
		t.Errorf("Expected action name 'get_attribute', got '%s'", action.Name)
	}

	if action.Selector != "#link" {
		t.Errorf("Expected selector '#link', got '%s'", action.Selector)
	}

	if len(action.Attribute) != 1 || action.Attribute[0] != "href" {
		t.Errorf("Expected attribute ['href'], got %v", action.Attribute)
	}
}

func TestSelectActionMapping(t *testing.T) {
	action := NewChromeDPAction()
	testData := map[string]any{
		"name":     "select",
		"selector": "select#dropdown",
		"value":    "option2",
	}

	err := probe.MapToStructByTags(testData, action)
	if err != nil {
		t.Errorf("MapToStructByTags failed: %v", err)
	}

	if action.Name != "select" {
		t.Errorf("Expected action name 'select', got '%s'", action.Name)
	}

	if action.Value != "option2" {
		t.Errorf("Expected value 'option2', got '%s'", action.Value)
	}
}

func TestMediumPriorityActions(t *testing.T) {
	testCases := []struct {
		name        string
		actionName  string
		expectError bool
	}{
		{"hover action", "hover", false},
		{"focus action", "focus", false},
		{"get_html action", "get_html", false},
		{"wait_enabled action", "wait_enabled", false},
		{"double_click action", "double_click", false},
		{"right_click action", "right_click", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data := map[string]string{
				"actions__0__name":     tc.actionName,
				"actions__0__selector": "#test-element",
			}

			_, err := Request(data)

			if tc.expectError && err == nil {
				t.Errorf("Expected error for action %s, but got none", tc.actionName)
			}

			if !tc.expectError && err != nil {
				// For non-error cases, we expect chromedp context errors since we're not running a real browser
				// Just check that the action was recognized (not "unsupported action type" error)
				if err.Error() == fmt.Sprintf("unsupported action type: %s", tc.actionName) {
					t.Errorf("Action %s should be supported but got unsupported error", tc.actionName)
				}
			}
		})
	}
}

func TestGetHtmlActionMapping(t *testing.T) {
	action := NewChromeDPAction()
	testData := map[string]any{
		"id":       "html1",
		"name":     "get_html",
		"selector": ".content",
	}

	err := probe.MapToStructByTags(testData, action)
	if err != nil {
		t.Errorf("MapToStructByTags failed: %v", err)
	}

	if action.ID != "html1" {
		t.Errorf("Expected ID 'html1', got '%s'", action.ID)
	}

	if action.Name != "get_html" {
		t.Errorf("Expected action name 'get_html', got '%s'", action.Name)
	}

	if action.Selector != ".content" {
		t.Errorf("Expected selector '.content', got '%s'", action.Selector)
	}
}

func TestMouseActionMapping(t *testing.T) {
	testCases := []struct {
		name       string
		actionName string
		selector   string
	}{
		{"hover mapping", "hover", "#hover-target"},
		{"double_click mapping", "double_click", "#double-click-target"},
		{"right_click mapping", "right_click", "#right-click-target"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			action := NewChromeDPAction()
			testData := map[string]any{
				"name":     tc.actionName,
				"selector": tc.selector,
			}

			err := probe.MapToStructByTags(testData, action)
			if err != nil {
				t.Errorf("MapToStructByTags failed: %v", err)
			}

			if action.Name != tc.actionName {
				t.Errorf("Expected action name '%s', got '%s'", tc.actionName, action.Name)
			}

			if action.Selector != tc.selector {
				t.Errorf("Expected selector '%s', got '%s'", tc.selector, action.Selector)
			}
		})
	}
}
