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
		case "text", "value":
			key := action.Name
			if action.ID != "" {
				key = action.ID
			}
			results[key] = *action.reText
		case "full_screenshot", "capture_screenshot", "screenshot":
			if len(*action.reBuf) > 0 {
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

/*
func (a *Action) getAttribute(ctx context.Context, selector, attribute string) (map[string]string, error) {
	if selector == "" {
		return nil, fmt.Errorf("selector parameter is required for get_attribute action")
	}
	if attribute == "" {
		return nil, fmt.Errorf("attribute parameter is required for get_attribute action")
	}

	var value string
	var ok bool
	err := chromedp.Run(ctx, chromedp.AttributeValue(selector, attribute, &value, &ok, chromedp.NodeVisible))

	result := map[string]string{
		"selector":  selector,
		"attribute": attribute,
		"value":     value,
		"exists":    strconv.FormatBool(ok),
		"success":   strconv.FormatBool(err == nil),
	}

	if err != nil {
		result["error"] = err.Error()
		return result, err
	}

	return result, nil
}

func (a *Action) getHTML(ctx context.Context, selector string) (map[string]string, error) {
	if selector == "" {
		return nil, fmt.Errorf("selector parameter is required for get_html action")
	}

	var html string
	err := chromedp.Run(ctx, chromedp.OuterHTML(selector, &html, chromedp.NodeVisible))

	result := map[string]string{
		"selector": selector,
		"html":     html,
		"success":  strconv.FormatBool(err == nil),
	}

	if err != nil {
		result["error"] = err.Error()
		return result, err
	}

	return result, nil
}

func (a *Action) click(ctx context.Context, selector string) (map[string]string, error) {
	if selector == "" {
		return nil, fmt.Errorf("selector parameter is required for click action")
	}

	start := time.Now()
	err := chromedp.Run(ctx, chromedp.Click(selector, chromedp.NodeVisible))
	duration := time.Since(start)

	result := map[string]string{
		"selector": selector,
		"time_ms":  fmt.Sprintf("%d", duration.Milliseconds()),
		"success":  strconv.FormatBool(err == nil),
	}

	if err != nil {
		result["error"] = err.Error()
		return result, err
	}

	return result, nil
}

func (a *Action) typeText(ctx context.Context, selector, value string) (map[string]string, error) {
	if selector == "" {
		return nil, fmt.Errorf("selector parameter is required for type action")
	}
	if value == "" {
		return nil, fmt.Errorf("value parameter is required for type action")
	}

	err := chromedp.Run(ctx,
		chromedp.Clear(selector),
		chromedp.SendKeys(selector, value, chromedp.NodeVisible),
	)

	result := map[string]string{
		"selector": selector,
		"value":    value,
		"success":  strconv.FormatBool(err == nil),
	}

	if err != nil {
		result["error"] = err.Error()
		return result, err
	}

	return result, nil
}

func (a *Action) submit(ctx context.Context, selector string) (map[string]string, error) {
	if selector == "" {
		return nil, fmt.Errorf("selector parameter is required for submit action")
	}

	start := time.Now()
	err := chromedp.Run(ctx, chromedp.Submit(selector, chromedp.NodeVisible))
	duration := time.Since(start)

	result := map[string]string{
		"selector": selector,
		"time_ms":  fmt.Sprintf("%d", duration.Milliseconds()),
		"success":  strconv.FormatBool(err == nil),
	}

	if err != nil {
		result["error"] = err.Error()
		return result, err
	}

	return result, nil
}

func (a *Action) waitVisible(ctx context.Context, selector string) (map[string]string, error) {
	if selector == "" {
		return nil, fmt.Errorf("selector parameter is required for wait_visible action")
	}

	start := time.Now()
	err := chromedp.Run(ctx, chromedp.WaitVisible(selector))
	duration := time.Since(start)

	result := map[string]string{
		"selector": selector,
		"time_ms":  fmt.Sprintf("%d", duration.Milliseconds()),
		"success":  strconv.FormatBool(err == nil),
	}

	if err != nil {
		result["error"] = err.Error()
		return result, err
	}

	return result, nil
}

func (a *Action) waitText(ctx context.Context, selector, expectedText string) (map[string]string, error) {
	if selector == "" {
		return nil, fmt.Errorf("selector parameter is required for wait_text action")
	}
	if expectedText == "" {
		return nil, fmt.Errorf("value parameter is required for wait_text action")
	}

	start := time.Now()

	// Wait for element to be visible first, then check text content
	var text string
	err := chromedp.Run(ctx,
		chromedp.WaitVisible(selector),
		chromedp.Text(selector, &text, chromedp.NodeVisible),
	)

	// Check if text matches expected
	if err == nil && text != expectedText {
		err = fmt.Errorf("text mismatch: expected '%s', got '%s'", expectedText, text)
	}

	duration := time.Since(start)

	result := map[string]string{
		"selector":      selector,
		"expected_text": expectedText,
		"actual_text":   text,
		"time_ms":       fmt.Sprintf("%d", duration.Milliseconds()),
		"success":       strconv.FormatBool(err == nil),
	}

	if err != nil {
		result["error"] = err.Error()
		return result, err
	}

	return result, nil
}

func (a *Action) getElements(ctx context.Context, selector string) (map[string]string, error) {
	if selector == "" {
		return nil, fmt.Errorf("selector parameter is required for get_elements action")
	}

	var nodes []*runtime.RemoteObject
	err := chromedp.Run(ctx, chromedp.Evaluate(fmt.Sprintf(`
		Array.from(document.querySelectorAll('%s')).map((el, index) => ({
			index: index,
			tagName: el.tagName,
			text: el.textContent.trim(),
			html: el.outerHTML,
			id: el.id,
			className: el.className
		}))
	`, selector), &nodes))

	result := map[string]string{
		"selector": selector,
		"success":  strconv.FormatBool(err == nil),
	}

	if err != nil {
		result["error"] = err.Error()
		return result, err
	}

	// Convert nodes to string representation
	if len(nodes) > 0 {
		result["count"] = strconv.Itoa(len(nodes))
		// For simplicity, just return the count. In a real implementation,
		// you might want to return more detailed information about each element.
	} else {
		result["count"] = "0"
	}

	return result, nil
}
*/
