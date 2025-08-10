package browser

import (
	"context"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/linyows/probe"
)

const (
	defaultTimeout = 5 * time.Second
	// Full HD
	defaultWindowWidth  = 1920
	defaultWindowHeight = 1080
	// Actions
	defaultQuality = 90
)

type Option func(*Callback)

type Callback struct {
	withInBrowser func(s string, i ...interface{})
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
	Path      string   `map:"path"`
	Quality   int      `map:"quality"`
	reText    *string
	reBuf     *[]byte
}

func NewChromeDPAction() *ChromeDPAction {
	return &ChromeDPAction{
		Quality: defaultQuality,
	}
}

type Req struct {
	Actions  []*ChromeDPAction
	Headless bool `map:"headless"`
	WindowW  int  `map:"window_w"`
	WindowH  int  `map:"window_h"`
	Timeout  time.Duration
	cb       *Callback
}

func NewReq() *Req {
	return &Req{
		Timeout:  defaultTimeout,
		WindowW:  defaultWindowWidth,
		WindowH:  defaultWindowHeight,
		Headless: true,
	}
}

type Res struct {
	Code    int               `map:"code"`
	Results map[string]string `map:"results"`
}

type Result struct {
	Req Req           `map:"req"`
	Res Res           `map:"res"`
	RT  time.Duration `map:"rt"`
}

func Request(data map[string]string, opts ...Option) (map[string]string, error) {
	unflattened := probe.UnflattenInterface(data)

	req := NewReq()
	cb := &Callback{}
	for _, opt := range opts {
		opt(cb)
	}
	req.cb = cb

	actions, exists := unflattened["actions"]
	if !exists || actions == "" {
		return map[string]string{}, fmt.Errorf("actions parameter is required")
	}
	if sl, ok := actions.([]interface{}); ok {
		for _, cdpa := range sl {
			if ma, ok := cdpa.(map[string]any); ok {
				a := NewChromeDPAction()
				err := probe.MapToStructByTags(ma, a)
				if err != nil {
					return map[string]string{"error": "MapToStructByTags failed"}, err
				}
				req.Actions = append(req.Actions, a)
			}
		}
	}

	headless, exists := unflattened["headless"]
	if bo, ok := headless.(bool); ok {
		req.Headless = bo
	}

	timeout, exists := unflattened["timeout"]
	if st, ok := timeout.(string); ok {
		if parsed, err := time.ParseDuration(st); err == nil {
			req.Timeout = parsed
		}
	}

	ww, exists := unflattened["window_w"]
	if in, ok := ww.(int); ok {
		req.WindowW = in
	}
	wh, exists := unflattened["window_h"]
	if in, ok := wh.(int); ok {
		req.WindowH = in
	}

	start := time.Now()

	// Setup chromedp options
	cdpOpts := []chromedp.ExecAllocatorOption{
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.DisableGPU,
		chromedp.NoSandbox,
		chromedp.WindowSize(req.WindowW, req.WindowH),
		chromedp.Flag("headless", req.Headless),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-background-timer-throttling", true),
		chromedp.Flag("disable-backgrounding-occluded-windows", true),
		chromedp.Flag("disable-renderer-backgrounding", true),
	}

	// Create allocator context
	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), cdpOpts...)
	defer allocCancel()
	// Create context with timeout
	ctx, cancel := chromedp.NewContext(allocCtx, chromedp.WithDebugf(func(s string, i ...interface{}) {
		// Callback
		if req.cb != nil && req.cb.withInBrowser != nil {
			req.cb.withInBrowser(s, i)
		}
	}))
	defer cancel()
	ctx, timeoutCancel := context.WithTimeout(ctx, req.Timeout)
	defer timeoutCancel()

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

	// Callback
	if req.cb != nil && req.cb.before != nil {
		req.cb.before(req)
	}

	if err := chromedp.Run(ctx, tasks); err != nil {
		return map[string]string{}, err
	}

	results := make(map[string]string)

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
				if err := ioutil.WriteFile(action.Path, *action.reBuf, 0644); err != nil {
					return map[string]string{}, err
				}
			}
		}
	}

	res := &Res{
		Code:    0,
		Results: results,
	}
	ret := Result{
		Req: *req,
		Res: *res,
		RT:  time.Since(start),
	}

	// Callback
	if req.cb != nil && req.cb.after != nil {
		req.cb.after(res)
	}

	mapRet, err := probe.StructToMapByTags(ret)
	if err != nil {
		return map[string]string{}, err
	}

	return probe.FlattenInterface(mapRet), nil
}

func WithInBrowser(f func(s string, i ...interface{})) Option {
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
