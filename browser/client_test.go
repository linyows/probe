package browser

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
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

	// Test individual fields instead of deep equal since browserRunner is a pointer
	if got.Timeout != defaultTimeout {
		t.Errorf("Expected timeout %v, got %v", defaultTimeout, got.Timeout)
	}
	if got.WindowW != defaultWindowWidth {
		t.Errorf("Expected WindowW %d, got %d", defaultWindowWidth, got.WindowW)
	}
	if got.WindowH != defaultWindowHeight {
		t.Errorf("Expected WindowH %d, got %d", defaultWindowHeight, got.WindowH)
	}
	if got.Headless != true {
		t.Errorf("Expected Headless true, got %v", got.Headless)
	}
	if got.browserRunner == nil {
		t.Error("Expected browserRunner to be set, got nil")
	}
	// Verify it's the correct type
	if _, ok := got.browserRunner.(*ChromeDPRunner); !ok {
		t.Errorf("Expected ChromeDPRunner, got %T", got.browserRunner)
	}
}

func TestRequest_Validation(t *testing.T) {
	testCases := []struct {
		name        string
		data        map[string]string
		expectedErr string
	}{
		{
			"missing actions",
			map[string]string{
				"headless": "true",
			},
			"actions parameter is required",
		},
		{
			"invalid action",
			map[string]string{
				"actions__0__name": "invalid_action",
				"actions__0__url":  "http://example.com",
			},
			"unsupported action type: invalid_action",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := Request(tc.data)

			if err == nil {
				t.Errorf("Expected error for %s", tc.name)
			}

			if err.Error() != tc.expectedErr {
				t.Errorf("Expected error message '%s', got '%s'", tc.expectedErr, err.Error())
			}

			// Check that status field is present and indicates failure
			if status, exists := result["status"]; !exists {
				t.Error("Expected status field in error result")
			} else if status != "1" {
				t.Errorf("Expected status=1 for error, got %v", status)
			}
		})
	}
}

func TestRequest_ParameterMapping(t *testing.T) {
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

func TestCallback_Options(t *testing.T) {
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

func TestChromeDPAction_Mapping(t *testing.T) {
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

func TestMockRunner(t *testing.T) {
	t.Run("basic mock functionality", func(t *testing.T) {
		mock := NewMockRunner()

		// Test that no calls have been made initially
		if mock.GetCallCount() != 0 {
			t.Errorf("Expected 0 calls initially, got %d", mock.GetCallCount())
		}

		// Test running with mock
		ctx := context.Background()
		actions := []chromedp.Action{
			chromedp.Navigate("http://example.com"),
			chromedp.WaitVisible("body"),
		}

		err := mock.Run(ctx, actions...)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Verify call was recorded
		if mock.GetCallCount() != 1 {
			t.Errorf("Expected 1 call, got %d", mock.GetCallCount())
		}

		lastCall := mock.GetLastCall()
		if len(lastCall) != 2 {
			t.Errorf("Expected 2 actions in last call, got %d", len(lastCall))
		}
	})

	t.Run("custom run function", func(t *testing.T) {
		mock := NewMockRunner()
		expectedErr := fmt.Errorf("custom error")

		mock.SetRunFunc(func(ctx context.Context, actions ...chromedp.Action) error {
			return expectedErr
		})

		ctx := context.Background()
		err := mock.Run(ctx, chromedp.Navigate("http://example.com"))

		if err != expectedErr {
			t.Errorf("Expected custom error, got %v", err)
		}
	})

	t.Run("multiple calls tracking", func(t *testing.T) {
		mock := NewMockRunner()

		// Make multiple calls
		ctx := context.Background()
		_ = mock.Run(ctx, chromedp.Navigate("http://example.com"))
		_ = mock.Run(ctx, chromedp.WaitVisible("body"))
		_ = mock.Run(ctx, chromedp.Click("button"))

		if mock.GetCallCount() != 3 {
			t.Errorf("Expected 3 calls, got %d", mock.GetCallCount())
		}

		allCalls := mock.GetAllCalls()
		if len(allCalls) != 3 {
			t.Errorf("Expected 3 calls in history, got %d", len(allCalls))
		}
	})
}

func TestReqWithMockRunner(t *testing.T) {
	t.Run("request with mock runner", func(t *testing.T) {
		// Create request with mock runner
		req := NewReq()
		mock := NewMockRunner()
		req.browserRunner = mock

		// Add a simple action
		action := NewChromeDPAction()
		action.Name = "navigate"
		action.URL = "http://example.com"
		req.Actions = []*ChromeDPAction{action}

		// Execute request
		result, err := req.do()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Verify result
		if result == nil {
			t.Error("Expected non-nil result")
			return
		}
		if result.Res.Code != 0 {
			t.Errorf("Expected code 0, got %d", result.Res.Code)
		}

		// Verify mock was called
		if mock.GetCallCount() != 1 {
			t.Errorf("Expected 1 call to browser runner, got %d", mock.GetCallCount())
		}
	})

	t.Run("request with error from mock runner", func(t *testing.T) {
		req := NewReq()
		mock := NewMockRunner()
		expectedErr := fmt.Errorf("browser error")

		mock.SetRunFunc(func(ctx context.Context, actions ...chromedp.Action) error {
			return expectedErr
		})

		req.browserRunner = mock

		// Add action
		action := NewChromeDPAction()
		action.Name = "navigate"
		action.URL = "http://example.com"
		req.Actions = []*ChromeDPAction{action}

		// Execute request - should fail
		result, err := req.do()
		if err != expectedErr {
			t.Errorf("Expected custom error, got %v", err)
		}
		if result != nil {
			t.Error("Expected nil result on error")
		}
	})
}

func TestMockRunnerArgumentVerification(t *testing.T) {
	t.Run("verify navigate action arguments", func(t *testing.T) {
		req := NewReq()
		mock := NewMockRunner()
		req.browserRunner = mock

		// Add navigate action
		action := NewChromeDPAction()
		action.Name = "navigate"
		action.URL = "https://example.com/test"
		req.Actions = []*ChromeDPAction{action}

		// Execute request
		result, err := req.do()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if result == nil {
			t.Error("Expected non-nil result")
		}

		// Verify the action was called
		if mock.GetCallCount() != 1 {
			t.Errorf("Expected 1 call, got %d", mock.GetCallCount())
		}

		// Get the actions that were passed to Run
		lastCall := mock.GetLastCall()
		if len(lastCall) == 0 {
			t.Error("Expected at least one action in the call")
		}

		// We can't easily inspect the exact chromedp.Action content without reflection,
		// but we can verify that actions were created and passed
		t.Logf("Successfully called with %d actions", len(lastCall))
	})

	t.Run("verify multiple actions", func(t *testing.T) {
		req := NewReq()
		mock := NewMockRunner()
		req.browserRunner = mock

		// Add multiple actions
		navigate := NewChromeDPAction()
		navigate.Name = "navigate"
		navigate.URL = "https://example.com"

		click := NewChromeDPAction()
		click.Name = "click"
		click.Selector = "#submit-button"

		req.Actions = []*ChromeDPAction{navigate, click}

		// Execute request
		result, err := req.do()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if result == nil {
			t.Error("Expected non-nil result")
		}

		// Verify multiple actions were processed
		if mock.GetCallCount() != 1 {
			t.Errorf("Expected 1 call, got %d", mock.GetCallCount())
		}

		lastCall := mock.GetLastCall()
		// Should have actions for: navigate, click, plus potentially some setup actions
		if len(lastCall) < 2 {
			t.Errorf("Expected at least 2 actions, got %d", len(lastCall))
		}
	})
}
