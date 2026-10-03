package browser

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/linyows/probe/mapping"
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
		data        map[string]any
		expectedErr string
	}{
		{
			"missing actions",
			map[string]any{
				"headless": "true",
			},
			"actions parameter is required",
		},
		{
			"invalid action",
			map[string]any{
				"actions": []any{
					map[string]any{
						"name": "invalid_action",
						"url":  "http://example.com",
					},
				},
			},
			"unsupported action type: invalid_action",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock runner that will fail for unsupported actions
			mockRunner := NewMockRunner()
			mockRunner.SetRunFunc(func(ctx context.Context, actions ...chromedp.Action) error {
				return fmt.Errorf("mock error")
			})

			// Create req and set mock runner
			req := NewReq()
			req.browserRunner = mockRunner

			// Parse data
			err := req.parseData(tc.data, nil)
			if err != nil {
				// Check if it's the expected parsing error
				if err.Error() != tc.expectedErr {
					t.Errorf("Expected error message '%s', got '%s'", tc.expectedErr, err.Error())
				}
				return
			}

			// If parsing succeeded, try to build action tasks (this should fail for invalid actions)
			_, err = req.buildActionTasks()
			if err != nil {
				if err.Error() != tc.expectedErr {
					t.Errorf("Expected error message '%s', got '%s'", tc.expectedErr, err.Error())
				}
				return
			}

			// If we get here, the test should have failed but didn't
			t.Errorf("Expected error for %s, but got none", tc.name)
		})
	}
}

func TestRequest_ParameterMapping(t *testing.T) {
	// Use the new map[string]any format directly
	unflattened := map[string]any{
		"actions": []any{
			map[string]any{
				"name": "navigate",
				"url":  "http://example.com",
			},
		},
		"headless": false,
		"timeout":  "10s",
		"window_w": 800,
		"window_h": 600,
	}
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

	withInBrowserOpt := WithInBrowser(func(s string, i ...any) {
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
			err := mapping.MapToStructByTags(tc.input, action)

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

	err := mapping.MapToStructByTags(testData, action)
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

	err := mapping.MapToStructByTags(testData, action)
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

	err := mapping.MapToStructByTags(testData, action)
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

			err := mapping.MapToStructByTags(testData, action)
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
		if !errors.Is(err, expectedErr) {
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

// stubCapture returns a capture function that hands back a fixed page and
// records whether the browser context was still alive when it was called.
func stubCapture(alive *bool) func(context.Context) ([]byte, string, string, error) {
	return func(ctx context.Context) ([]byte, string, string, error) {
		*alive = ctx.Err() == nil
		return []byte("\x89PNG fake"), "<html><body>at failure</body></html>", "http://app.test/checkout", nil
	}
}

func TestReq_FailureEvidence(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "evidence")

	req := NewReq()
	req.Timeout = 50 * time.Millisecond
	req.EvidenceDir = dir
	mock := NewMockRunner()
	// The action waits out its deadline, as wait_visible on a missing node does.
	mock.SetRunFunc(func(ctx context.Context, actions ...chromedp.Action) error {
		<-ctx.Done()
		return ctx.Err()
	})
	req.browserRunner = mock
	alive := false
	req.capture = stubCapture(&alive)

	action := NewChromeDPAction()
	action.Name = "wait_visible"
	action.Selector = "#missing"
	req.Actions = []*ChromeDPAction{action}

	_, err := req.do()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want it to wrap the action's deadline", err)
	}
	if !alive {
		t.Error("the page must be captured on a context that outlives the actions' deadline")
	}

	msg := err.Error()
	if !strings.Contains(msg, "page at failure: url http://app.test/checkout") {
		t.Errorf("message should give the URL: %s", msg)
	}
	shots, _ := filepath.Glob(filepath.Join(dir, "probe-browser-failure-*.png"))
	htmls, _ := filepath.Glob(filepath.Join(dir, "probe-browser-failure-*.html"))
	if len(shots) != 1 || len(htmls) != 1 {
		t.Fatalf("files = %v %v, want one screenshot and one HTML", shots, htmls)
	}
	if strings.TrimSuffix(shots[0], ".png") != strings.TrimSuffix(htmls[0], ".html") {
		t.Errorf("the two files should share a name: %s %s", shots[0], htmls[0])
	}
	if !strings.Contains(msg, "screenshot "+shots[0]) || !strings.Contains(msg, "html "+htmls[0]) {
		t.Errorf("message should name both files: %s", msg)
	}
	if data, _ := os.ReadFile(htmls[0]); string(data) != "<html><body>at failure</body></html>" {
		t.Errorf("html = %q", data)
	}
	for _, f := range []string{shots[0], htmls[0]} {
		if info, _ := os.Stat(f); info.Mode().Perm() != 0o600 {
			t.Errorf("%s mode = %v, want 0600: the page can hold private data", filepath.Base(f), info.Mode().Perm())
		}
	}
}

func TestReq_FailureEvidence_DefaultDir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)

	req := NewReq()
	alive := false
	req.capture = stubCapture(&alive)
	err := req.withEvidence(context.Background(), errors.New("boom"))

	if !strings.Contains(err.Error(), "screenshot "+tmp) {
		t.Errorf("without evidence_dir the files should go to the temporary directory: %v", err)
	}
}

func TestReq_FailureEvidence_CaptureFails(t *testing.T) {
	req := NewReq()
	req.EvidenceDir = t.TempDir()
	req.capture = func(context.Context) ([]byte, string, string, error) {
		return nil, "", "", errors.New("browser is gone")
	}
	original := errors.New("navigation failed")

	err := req.withEvidence(context.Background(), original)
	if !errors.Is(err, original) {
		t.Errorf("the action's error must be kept: %v", err)
	}
	if !strings.Contains(err.Error(), "the page could not be captured: browser is gone") {
		t.Errorf("message should say why there is no evidence: %v", err)
	}
	if files, _ := os.ReadDir(req.EvidenceDir); len(files) != 0 {
		t.Errorf("nothing should be written, got %d files", len(files))
	}
}

func TestReq_FailureEvidence_SaveFails(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	req := NewReq()
	req.EvidenceDir = filepath.Join(file, "evidence")
	alive := false
	req.capture = stubCapture(&alive)
	original := errors.New("click failed")

	err := req.withEvidence(context.Background(), original)
	if !errors.Is(err, original) || !strings.Contains(err.Error(), "the page could not be saved") {
		t.Errorf("error = %v", err)
	}
}

func TestReq_NoEvidenceOnSuccess(t *testing.T) {
	req := NewReq()
	req.EvidenceDir = t.TempDir()
	req.browserRunner = NewMockRunner()
	called := false
	req.capture = func(context.Context) ([]byte, string, string, error) {
		called = true
		return nil, "", "", nil
	}
	action := NewChromeDPAction()
	action.Name = "navigate"
	action.URL = "http://app.test/"
	req.Actions = []*ChromeDPAction{action}

	if _, err := req.do(); err != nil {
		t.Fatal(err)
	}
	if called {
		t.Error("a successful run should not capture the page")
	}
}

func TestRequest_EvidenceDirMapping(t *testing.T) {
	req := NewReq()
	err := req.parseData(map[string]any{
		"evidence_dir": "out/browser",
		"actions":      []any{map[string]any{"name": "navigate", "url": "http://app.test/"}},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if req.EvidenceDir != "out/browser" {
		t.Errorf("EvidenceDir = %q", req.EvidenceDir)
	}
}

// failingStartRunner is a runner whose browser cannot be launched.
type failingStartRunner struct{ MockRunner }

func (r *failingStartRunner) Start(context.Context) error { return errors.New("chrome not found") }

func TestReq_FailureEvidence_BrowserDidNotStart(t *testing.T) {
	req := NewReq()
	req.EvidenceDir = t.TempDir()
	req.browserRunner = &failingStartRunner{}
	called := false
	req.capture = func(context.Context) ([]byte, string, string, error) {
		called = true
		return nil, "", "", nil
	}
	action := NewChromeDPAction()
	action.Name = "navigate"
	action.URL = "http://app.test/"
	req.Actions = []*ChromeDPAction{action}

	_, err := req.do()
	if err == nil || !strings.Contains(err.Error(), "chrome not found (the page could not be captured: the browser did not start)") {
		t.Errorf("error = %v, want the start failure with a note that no page was saved", err)
	}
	if called {
		t.Error("there is no browser to capture the page from")
	}
}

func TestFullScreenshotQuality(t *testing.T) {
	tests := []struct{ in, want int }{
		{0, 100},   // not given in YAML: PNG, not a quality-0 JPEG
		{-5, 100},  // out of range
		{101, 100}, // out of range
		{100, 100}, // PNG
		{1, 1},
		{80, 80}, // JPEG at that quality
		{99, 99},
	}
	for _, tt := range tests {
		if got := fullScreenshotQuality(tt.in); got != tt.want {
			t.Errorf("fullScreenshotQuality(%d) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

// TestFullScreenshot_IsSaved covers full_screenshot, which captured an image
// and then dropped it: the buffer was never attached to the action.
func TestFullScreenshot_IsSaved(t *testing.T) {
	req := NewReq()
	action := &ChromeDPAction{Name: "full_screenshot", ID: "page"}
	req.Actions = []*ChromeDPAction{action}

	if _, err := req.buildActionTasks(); err != nil {
		t.Fatal(err)
	}
	if action.reBuf == nil {
		t.Fatal("full_screenshot must keep the buffer it captures into")
	}

	*action.reBuf = []byte("\x89PNG\r\n\x1a\n" + strings.Repeat("\x00", 16))
	_, filePaths, err := req.collectResults()
	if err != nil {
		t.Fatal(err)
	}
	path := filePaths["page"]
	t.Cleanup(func() { _ = os.Remove(path) })
	if !strings.HasSuffix(path, ".png") {
		t.Errorf("path = %q, want a saved .png", path)
	}
}

// TestScreenshot_ExtensionFollowsContent checks the saved file is named for
// what it holds: full_screenshot below quality 100 is JPEG.
func TestScreenshot_ExtensionFollowsContent(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		ext  string
	}{
		{"png", []byte("\x89PNG\r\n\x1a\n" + strings.Repeat("\x00", 16)), ".png"},
		{"jpeg", []byte("\xff\xd8\xff\xe0\x00\x10JFIF\x00" + strings.Repeat("\x00", 16)), ".jpg"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := tt.data
			req := NewReq()
			req.Actions = []*ChromeDPAction{{Name: "full_screenshot", ID: "shot", reBuf: &buf}}

			_, filePaths, err := req.collectResults()
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Remove(filePaths["shot"]) })
			if !strings.HasSuffix(filePaths["shot"], tt.ext) {
				t.Errorf("path = %q, want %s", filePaths["shot"], tt.ext)
			}
		})
	}
}

func TestJSString(t *testing.T) {
	tests := map[string]string{
		`#menu`:               `"#menu"`,
		`a[href='/checkout']`: `"a[href='/checkout']"`,
		`input[name="q"]`:     `"input[name=\"q\"]"`,
		"a\\b":                `"a\\b"`,
		"</script>":           `"\u003c/script\u003e"`,
	}
	for in, want := range tests {
		if got := jsString(in); got != want {
			t.Errorf("jsString(%q) = %s, want %s", in, got, want)
		}
	}
}
