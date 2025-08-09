package browser

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/linyows/probe"
)

type Action struct {
	log hclog.Logger
}

func (a *Action) Run(args []string, with map[string]string) (map[string]string, error) {
	a.log.Debug("received browser action request", "params", with)

	action := with["action"]
	if action == "" {
		return map[string]string{"error": "action parameter is required"}, fmt.Errorf("action parameter is required")
	}

	// Parse common parameters
	url := with["url"]
	selector := with["selector"]
	value := with["value"]
	attribute := with["attribute"]
	
	// Parse headless mode (default true)
	headless := true
	if h := with["headless"]; h != "" {
		if parsed, err := strconv.ParseBool(h); err == nil {
			headless = parsed
		}
	}

	// Parse timeout (default 30s)
	timeout := 30 * time.Second
	if t := with["timeout"]; t != "" {
		if parsed, err := time.ParseDuration(t); err == nil {
			timeout = parsed
		}
	}

	// Setup chromedp options
	opts := []chromedp.ExecAllocatorOption{
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.DisableGPU,
	}

	if headless {
		opts = append(opts, chromedp.Headless)
	}

	// Create allocator context
	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer allocCancel()

	// Create context with timeout
	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	ctx, timeoutCancel := context.WithTimeout(ctx, timeout)
	defer timeoutCancel()

	var result map[string]string
	var err error

	switch action {
	case "navigate":
		result, err = a.navigate(ctx, url)
	case "get_text":
		result, err = a.getText(ctx, selector)
	case "get_attribute":
		result, err = a.getAttribute(ctx, selector, attribute)
	case "get_html":
		result, err = a.getHTML(ctx, selector)
	case "click":
		result, err = a.click(ctx, selector)
	case "type":
		result, err = a.typeText(ctx, selector, value)
	case "submit":
		result, err = a.submit(ctx, selector)
	case "screenshot":
		result, err = a.screenshot(ctx)
	case "wait_visible":
		result, err = a.waitVisible(ctx, selector)
	case "wait_text":
		result, err = a.waitText(ctx, selector, value)
	case "get_elements":
		result, err = a.getElements(ctx, selector)
	default:
		return map[string]string{"error": fmt.Sprintf("unknown action: %s", action)}, 
			fmt.Errorf("unknown action: %s", action)
	}

	if err != nil {
		a.log.Error("browser action failed", "action", action, "error", err)
		if result == nil {
			result = make(map[string]string)
		}
		result["error"] = err.Error()
		return result, err
	}

	a.log.Debug("browser action completed", "action", action)
	return result, nil
}

func (a *Action) navigate(ctx context.Context, url string) (map[string]string, error) {
	if url == "" {
		return nil, fmt.Errorf("url parameter is required for navigate action")
	}

	start := time.Now()
	err := chromedp.Run(ctx, chromedp.Navigate(url))
	duration := time.Since(start)

	result := map[string]string{
		"url":      url,
		"time_ms":  fmt.Sprintf("%d", duration.Milliseconds()),
		"success":  strconv.FormatBool(err == nil),
	}

	if err != nil {
		result["error"] = err.Error()
		return result, err
	}

	return result, nil
}

func (a *Action) getText(ctx context.Context, selector string) (map[string]string, error) {
	if selector == "" {
		return nil, fmt.Errorf("selector parameter is required for get_text action")
	}

	var text string
	err := chromedp.Run(ctx, chromedp.Text(selector, &text, chromedp.NodeVisible))

	result := map[string]string{
		"selector": selector,
		"text":     text,
		"success":  strconv.FormatBool(err == nil),
	}

	if err != nil {
		result["error"] = err.Error()
		return result, err
	}

	return result, nil
}

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

func (a *Action) screenshot(ctx context.Context) (map[string]string, error) {
	var buf []byte
	err := chromedp.Run(ctx, chromedp.CaptureScreenshot(&buf))

	result := map[string]string{
		"success": strconv.FormatBool(err == nil),
	}

	if err != nil {
		result["error"] = err.Error()
		return result, err
	}

	// Encode screenshot as base64
	encoded := base64.StdEncoding.EncodeToString(buf)
	result["screenshot"] = encoded
	result["size_bytes"] = strconv.Itoa(len(buf))

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

func main() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: probe.Handshake,
		Plugins: map[string]plugin.Plugin{
			"actions": &probe.ActionsPlugin{Impl: &Action{}},
		},
		GRPCServer: plugin.DefaultGRPCServer,
		Logger: hclog.New(&hclog.LoggerOptions{
			Name:   "browser-action",
			Output: os.Stderr,
			Level:  hclog.Debug,
		}),
	})
}