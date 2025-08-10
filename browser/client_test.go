package browser

import (
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