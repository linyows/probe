package browser

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/linyows/probe/binary"
	"github.com/linyows/probe/mapping"
)

const (
	defaultTimeout = 5 * time.Second
	// evidenceGrace is how long the browser is kept after the actions'
	// deadline, so that the page can still be captured when they time out.
	evidenceGrace = 10 * time.Second
	// Full HD
	defaultWindowWidth  = 1920
	defaultWindowHeight = 1080
	// Actions
	defaultQuality = 90
)

// BrowserRunner defines the interface for running browser actions
type BrowserRunner interface {
	// Start launches the browser. chromedp ties the browser process to the
	// context of the first Run, so the browser is started on a context
	// without the actions' deadline and outlives a timed-out action.
	Start(ctx context.Context) error
	Run(ctx context.Context, actions ...chromedp.Action) error
}

// ChromeDPRunner implements BrowserRunner using the actual ChromeDP
type ChromeDPRunner struct{}

// Start launches the browser by running no actions on ctx.
func (r *ChromeDPRunner) Start(ctx context.Context) error {
	return chromedp.Run(ctx)
}

// Run executes actions using ChromeDP
func (r *ChromeDPRunner) Run(ctx context.Context, actions ...chromedp.Action) error {
	return chromedp.Run(ctx, actions...)
}

// MockRunner implements BrowserRunner for testing
type MockRunner struct {
	RunFunc     func(ctx context.Context, actions ...chromedp.Action) error
	CallHistory [][]chromedp.Action
	mu          sync.Mutex
}

// NewMockRunner creates a new mock browser runner
func NewMockRunner() *MockRunner {
	return &MockRunner{
		CallHistory: make([][]chromedp.Action, 0),
	}
}

// Start does nothing: there is no browser to launch. It is not recorded, so
// the call history holds only the actions that were run.
func (m *MockRunner) Start(ctx context.Context) error {
	return nil
}

// Run records the call and optionally executes a custom function
func (m *MockRunner) Run(ctx context.Context, actions ...chromedp.Action) error {
	m.mu.Lock()
	m.CallHistory = append(m.CallHistory, actions)
	m.mu.Unlock()

	if m.RunFunc != nil {
		return m.RunFunc(ctx, actions...)
	}
	return nil // Default success
}

// GetCallCount returns the number of Run calls made
func (m *MockRunner) GetCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.CallHistory)
}

// GetLastCall returns the actions from the most recent Run call
func (m *MockRunner) GetLastCall() []chromedp.Action {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.CallHistory) == 0 {
		return nil
	}
	return m.CallHistory[len(m.CallHistory)-1]
}

// GetAllCalls returns all recorded Run calls
func (m *MockRunner) GetAllCalls() [][]chromedp.Action {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([][]chromedp.Action, len(m.CallHistory))
	copy(result, m.CallHistory)
	return result
}

// SetRunFunc sets a custom function to execute when Run is called
func (m *MockRunner) SetRunFunc(fn func(ctx context.Context, actions ...chromedp.Action) error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.RunFunc = fn
}

type Option func(*Callback)

type Callback struct {
	withInBrowser func(s string, i ...any)
	before        func(req *Req)
	after         func(res *Res)
}

type ChromeDPAction struct {
	ID        string   `map:"id"`
	Name      string   `map:"name"`
	URL       string   `map:"url"`
	Selector  string   `map:"selector"`
	Attribute []string `map:"attribute"`
	Value     string   `map:"value"`
	Path      string   `map:"path"` // Deprecated: use FilePath from results
	Quality   int      `map:"quality"`
	reText    *string
	reBuf     *[]byte
	filePath  *string // New: for binary file path
}

func NewChromeDPAction() *ChromeDPAction {
	return &ChromeDPAction{
		Quality: defaultQuality,
	}
}

type Req struct {
	Actions  []*ChromeDPAction `map:"actions"`
	Headless bool              `map:"headless"`
	WindowW  int               `map:"window_w"`
	WindowH  int               `map:"window_h"`
	// EvidenceDir is where the page is saved when the actions fail. Empty
	// means the system's temporary directory.
	EvidenceDir   string `map:"evidence_dir"`
	Timeout       time.Duration
	cb            *Callback
	browserRunner BrowserRunner
	// capture takes the page as it is when the actions fail. Nil means
	// capturing it through browserRunner; tests replace it.
	capture func(ctx context.Context) (screenshot []byte, html, url string, err error)
}

func NewReq() *Req {
	return &Req{
		Timeout:       defaultTimeout,
		WindowW:       defaultWindowWidth,
		WindowH:       defaultWindowHeight,
		Headless:      true,
		browserRunner: &ChromeDPRunner{},
	}
}

type Res struct {
	Code      int               `map:"code"`
	Results   map[string]string `map:"results"`
	FilePaths map[string]string `map:"filepaths"` // New: for binary file paths (action_id -> filepath)
}

type Result struct {
	Req    Req           `map:"req"`
	Res    Res           `map:"res"`
	RT     time.Duration `map:"rt"`
	Status int           `map:"status"`
}

func (req *Req) parseData(data map[string]any, opts []Option) error {
	cb := &Callback{}
	for _, opt := range opts {
		opt(cb)
	}
	req.cb = cb

	// Validate actions existence
	if _, exists := data["actions"]; !exists {
		return fmt.Errorf("actions parameter is required")
	}

	// Use MapToStructByTags to parse all fields including actions
	err := mapping.MapToStructByTags(data, req)
	if err != nil {
		return fmt.Errorf("MapToStructByTags failed: %w", err)
	}

	// Handle timeout separately as it requires special parsing
	if timeout, exists := data["timeout"]; exists {
		if st, ok := timeout.(string); ok {
			if parsed, err := time.ParseDuration(st); err == nil {
				req.Timeout = parsed
			}
		}
	}

	return nil
}

func (req *Req) buildChromeDPOptions() []chromedp.ExecAllocatorOption {
	// Start from DefaultExecAllocatorOptions to include all recommended flags
	// (disable-extensions, enable-automation, disable-sync, etc.)
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.DisableGPU,
		chromedp.NoSandbox,
		chromedp.WindowSize(req.WindowW, req.WindowH),
		chromedp.Flag("headless", req.Headless),
		chromedp.WSURLReadTimeout(req.Timeout),
	)
	return opts
}

// createBrowserContext returns the chromedp context the browser lives in.
// It carries no deadline of its own beyond browserDeadline, which bounds the
// browser process as a whole; the actions get their own, earlier deadline.
func (req *Req) createBrowserContext(browserDeadline time.Time) (context.Context, context.CancelFunc, error) {
	cdpOpts := req.buildChromeDPOptions()

	allocBaseCtx, allocBaseCancel := context.WithDeadline(context.Background(), browserDeadline)
	allocCtx, allocCancel := chromedp.NewExecAllocator(allocBaseCtx, cdpOpts...)
	ctx, cancel := chromedp.NewContext(allocCtx, chromedp.WithDebugf(func(s string, i ...any) {
		// Callback
		if req.cb != nil && req.cb.withInBrowser != nil {
			req.cb.withInBrowser(s, i)
		}
	}))

	// Combine cancellation functions
	cancelFunc := func() {
		cancel()
		allocCancel()
		allocBaseCancel()
	}

	return ctx, cancelFunc, nil
}

func (req *Req) buildActionTasks() (chromedp.Tasks, error) {
	tasks := chromedp.Tasks{}

	for _, action := range req.Actions {
		switch action.Name {
		case "navigate":
			tasks = append(tasks, chromedp.Navigate(action.URL))
		case "wait_visible":
			tasks = append(tasks, chromedp.WaitVisible(action.Selector, chromedp.ByQuery))
		case "text":
			var text string
			tasks = append(tasks, chromedp.Text(action.Selector, &text, chromedp.ByQuery))
			action.reText = &text
		case "value":
			var text string
			tasks = append(tasks, chromedp.Value(action.Selector, &text))
			action.reText = &text
		case "click":
			tasks = append(tasks, chromedp.Click(action.Selector, chromedp.NodeVisible))
		case "send_keys", "type":
			tasks = append(tasks, chromedp.Clear(action.Selector))
			tasks = append(tasks, chromedp.SendKeys(action.Selector, action.Value, chromedp.NodeVisible))
		case "full_screenshot":
			var buf []byte
			tasks = append(tasks, chromedp.FullScreenshot(&buf, action.Quality))
		case "capture_screenshot":
			var buf []byte
			tasks = append(tasks, chromedp.CaptureScreenshot(&buf))
			action.reBuf = &buf
		case "screenshot":
			var buf []byte
			tasks = append(tasks, chromedp.Screenshot(action.Selector, &buf, chromedp.NodeVisible))
			action.reBuf = &buf
		case "wait_ready":
			tasks = append(tasks, chromedp.WaitReady("body"))
		case "wait_not_visible":
			tasks = append(tasks, chromedp.WaitNotVisible(action.Selector, chromedp.ByQuery))
		case "submit":
			tasks = append(tasks, chromedp.Submit(action.Selector, chromedp.NodeVisible))
		case "select":
			tasks = append(tasks, chromedp.SetAttributeValue(action.Selector, "value", action.Value, chromedp.NodeVisible))
		case "scroll":
			tasks = append(tasks, chromedp.ScrollIntoView(action.Selector, chromedp.NodeVisible))
		case "get_attribute":
			if len(action.Attribute) > 0 {
				var value string
				var ok bool
				tasks = append(tasks, chromedp.AttributeValue(action.Selector, action.Attribute[0], &value, &ok, chromedp.NodeVisible))
				action.reText = &value
			} else {
				return nil, fmt.Errorf("attribute parameter is required for get_attribute action")
			}
		case "wait_text":
			tasks = append(tasks, chromedp.WaitVisible(action.Selector, chromedp.ByQuery))
			var text string
			tasks = append(tasks, chromedp.Text(action.Selector, &text, chromedp.ByQuery))
			action.reText = &text
		case "hover":
			tasks = append(tasks, chromedp.EvaluateAsDevTools(fmt.Sprintf(`
				const el = document.querySelector('%s');
				if (el) {
					const event = new MouseEvent('mouseover', {
						view: window,
						bubbles: true,
						cancelable: true
					});
					el.dispatchEvent(event);
				}
			`, action.Selector), nil))
		case "focus":
			tasks = append(tasks, chromedp.Focus(action.Selector, chromedp.NodeVisible))
		case "get_html":
			var html string
			tasks = append(tasks, chromedp.OuterHTML(action.Selector, &html, chromedp.NodeVisible))
			action.reText = &html
		case "wait_enabled":
			tasks = append(tasks, chromedp.WaitEnabled(action.Selector, chromedp.NodeVisible))
		case "double_click":
			tasks = append(tasks, chromedp.DoubleClick(action.Selector, chromedp.NodeVisible))
		case "right_click":
			tasks = append(tasks, chromedp.EvaluateAsDevTools(fmt.Sprintf(`
				const el = document.querySelector('%s');
				if (el) {
					const event = new MouseEvent('contextmenu', {
						view: window,
						bubbles: true,
						cancelable: true,
						button: 2
					});
					el.dispatchEvent(event);
				}
			`, action.Selector), nil))
		default:
			return nil, fmt.Errorf("unsupported action type: %s", action.Name)
		}
	}

	return tasks, nil
}

func (req *Req) collectResults() (map[string]string, map[string]string, error) {
	results := make(map[string]string)
	filePaths := make(map[string]string)

	for _, action := range req.Actions {
		switch action.Name {
		case "text", "value", "get_attribute", "wait_text", "get_html":
			key := action.Name
			if action.ID != "" {
				key = action.ID
			}
			if action.reText != nil {
				results[key] = *action.reText
			}
		case "full_screenshot", "capture_screenshot", "screenshot":
			if action.reBuf != nil && len(*action.reBuf) > 0 {
				// Save screenshot to temporary file using our binary utility
				filePath, err := binary.SaveBinaryToTempFile(*action.reBuf, "image/png")
				if err != nil {
					return nil, nil, fmt.Errorf("failed to save screenshot: %w", err)
				}
				action.filePath = &filePath

				// Store file path in results
				key := action.Name
				if action.ID != "" {
					key = action.ID
				}
				filePaths[key] = filePath

				// Backward compatibility: also save to specified path if provided
				if action.Path != "" {
					if err := os.WriteFile(action.Path, *action.reBuf, 0644); err != nil {
						return nil, nil, err
					}
				}
			}
		}
	}

	return results, filePaths, nil
}

func (req *Req) do() (*Result, error) {
	start := time.Now()
	deadline := start.Add(req.Timeout)

	browserCtx, cancel, err := req.createBrowserContext(deadline.Add(evidenceGrace))
	if err != nil {
		return nil, err
	}
	defer cancel()

	tasks, err := req.buildActionTasks()
	if err != nil {
		return nil, err
	}

	// Callback
	if req.cb != nil && req.cb.before != nil {
		req.cb.before(req)
	}

	if err := req.browserRunner.Start(browserCtx); err != nil {
		// There is no page to capture, but say so, as every other failure
		// of the actions says whether the page was saved.
		return nil, fmt.Errorf("%w (the page could not be captured: the browser did not start)", err)
	}

	ctx, actionsCancel := context.WithDeadline(browserCtx, deadline)
	defer actionsCancel()

	if err := req.browserRunner.Run(ctx, tasks...); err != nil {
		return nil, req.withEvidence(browserCtx, err)
	}

	results, filePaths, err := req.collectResults()
	if err != nil {
		return nil, err
	}

	res := &Res{
		Code:      0,
		Results:   results,
		FilePaths: filePaths,
	}
	ret := &Result{
		Req:    *req,
		Res:    *res,
		RT:     time.Since(start),
		Status: 0, // success
	}

	// Callback
	if req.cb != nil && req.cb.after != nil {
		req.cb.after(res)
	}

	return ret, nil
}

// evidenceTimeout bounds capturing the page after a failure.
const evidenceTimeout = 5 * time.Second

// withEvidence saves the page as it was when the actions failed, and names
// the files in the returned error. An action error is all that reaches the
// workflow runner, so the error message is where the report and the terminal
// can point at the evidence. A page that cannot be captured, such as a
// browser that never started, leaves the error as it was, with a note.
func (req *Req) withEvidence(browserCtx context.Context, err error) error {
	ctx, cancel := context.WithTimeout(browserCtx, evidenceTimeout)
	defer cancel()

	capture := req.capture
	if capture == nil {
		capture = req.captureWithRunner
	}
	screenshot, html, url, cerr := capture(ctx)
	if cerr != nil {
		return fmt.Errorf("%w (the page could not be captured: %v)", err, cerr)
	}

	dir := req.EvidenceDir
	if dir == "" {
		dir = os.TempDir()
	}
	shot, html2, serr := saveEvidence(dir, screenshot, html)
	if serr != nil {
		return fmt.Errorf("%w (the page could not be saved: %v)", err, serr)
	}
	return fmt.Errorf("%w (page at failure: url %s, screenshot %s, html %s)", err, url, shot, html2)
}

// captureWithRunner takes the URL, the HTML and a full-page screenshot of the
// current page. The HTML is read with a script rather than a node query,
// which would wait for the node to become visible.
func (req *Req) captureWithRunner(ctx context.Context) ([]byte, string, string, error) {
	var screenshot []byte
	var html, url string
	err := req.browserRunner.Run(ctx,
		chromedp.Location(&url),
		chromedp.Evaluate(`document.documentElement.outerHTML`, &html),
		// FullScreenshot returns PNG only at quality 100, and JPEG otherwise.
		chromedp.FullScreenshot(&screenshot, 100),
	)
	return screenshot, html, url, err
}

// saveEvidence writes the screenshot and the HTML into dir under one shared,
// unique name, and returns their paths.
func saveEvidence(dir string, screenshot []byte, html string) (string, string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", "", err
	}
	f, err := os.CreateTemp(dir, "probe-browser-failure-*.png")
	if err != nil {
		return "", "", err
	}
	shotPath := f.Name()
	_, err = f.Write(screenshot)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return "", "", err
	}

	htmlPath := strings.TrimSuffix(shotPath, ".png") + ".html"
	// The page can hold private data and the default directory is shared,
	// so the HTML stays private like the screenshot CreateTemp made.
	if err := os.WriteFile(htmlPath, []byte(html), 0o600); err != nil {
		return "", "", err
	}
	return shotPath, htmlPath, nil
}

func Request(data map[string]any, opts ...Option) (map[string]any, error) {
	start := time.Now()
	req := NewReq()

	if err := req.parseData(data, opts); err != nil {
		return createErrorResult(start, req, err)
	}

	result, err := req.do()
	if err != nil {
		return createErrorResult(start, req, err)
	}

	mapRet, err := mapping.StructToMapByTags(result)
	if err != nil {
		return createErrorResult(start, req, err)
	}

	return mapRet, nil
}

func createErrorResult(start time.Time, req *Req, err error) (map[string]any, error) {
	duration := time.Since(start)

	result := &Result{
		Req: *req,
		Res: Res{
			Code:    1,
			Results: map[string]string{},
		},
		RT:     duration,
		Status: 1, // failure
	}

	mapResult, mapErr := mapping.StructToMapByTags(result)
	if mapErr != nil {
		return map[string]any{}, mapErr
	}

	return mapResult, err
}

func WithInBrowser(f func(s string, i ...any)) Option {
	return func(c *Callback) {
		c.withInBrowser = f
	}
}

func WithBefore(f func(req *Req)) Option {
	return func(c *Callback) {
		c.before = f
	}
}

func WithAfter(f func(res *Res)) Option {
	return func(c *Callback) {
		c.after = f
	}
}
