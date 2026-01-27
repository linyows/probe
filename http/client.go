package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	hp "net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/linyows/probe"
)

type TransportOptions struct {
	Timeout      int `map:"timeout"`
	MaxIdleConns int `map:"max_idle_conns"`
}

type Req struct {
	URL    string            `map:"url" validate:"required"`
	Method string            `map:"method" validate:"required"`
	Proto  string            `map:"ver"`
	Header map[string]string `map:"headers"`
	Body   string            `map:"body"` // Changed from []byte to string for text data
	cb     *Callback
}

type Res struct {
	Status   string            `map:"status"`
	Code     int               `map:"code"`
	Header   map[string]string `map:"headers"`
	Body     string            `map:"body"`     // Changed from []byte to string for text data
	FilePath string            `map:"filepath"` // New field for binary file paths
}

type Result struct {
	Req    Req           `map:"req"`
	Res    Res           `map:"res"`
	RT     time.Duration `map:"rt"`
	Status int           `map:"status"`
}

func NewReq() *Req {
	return &Req{
		Method: "GET",
		Proto:  "HTTP/1.1",
		Header: map[string]string{
			"Accept":     "*/*",
			"User-Agent": "probe-http/1.0.0",
		},
	}
}

// mergeHeaders merges custom headers with default headers, handling case-insensitive duplicates
// Custom headers override defaults when header names match (case-insensitive)
func mergeHeaders(defaultHeaders, customHeaders map[string]string) map[string]string {
	if customHeaders == nil {
		return defaultHeaders
	}

	result := make(map[string]string)

	// First, copy all default headers
	for key, value := range defaultHeaders {
		result[key] = value
	}

	// Then, add/override with custom headers, removing case-insensitive duplicates
	for customKey, customValue := range customHeaders {
		// Check if this custom header should override a default header
		var keyToRemove string
		for existingKey := range result {
			if strings.EqualFold(existingKey, customKey) {
				keyToRemove = existingKey
				break
			}
		}

		// Remove the existing header if found
		if keyToRemove != "" {
			delete(result, keyToRemove)
		}

		// Add the custom header
		result[customKey] = customValue
	}

	return result
}

func (r *Req) Do() (*Result, error) {
	if r.URL == "" {
		return nil, errors.New("Req.URL is required")
	}

	// Debug: log the body content and content-type
	log := hclog.Default()
	log.Info("DEBUG: HTTP Request Body", "body", r.Body, "headers", r.Header)

	req, err := hp.NewRequest(r.Method, r.URL, strings.NewReader(r.Body))
	if err != nil {
		return nil, err
	}

	for k, v := range r.Header {
		// Clean header value by removing newlines and other invalid characters
		cleanValue := strings.ReplaceAll(strings.ReplaceAll(v, "\n", ""), "\r", "")
		req.Header.Set(probe.TitleCase(k, "-"), cleanValue)
	}

	// callback
	if r.cb != nil && r.cb.before != nil {
		r.cb.before(req)
	}

	result := &Result{Req: *r}

	cl := &hp.Client{}
	start := time.Now()
	res, err := cl.Do(req)
	result.RT = time.Since(start)
	if err != nil {
		return result, err
	}
	defer func() { _ = res.Body.Close() }()

	// callback
	if r.cb != nil && r.cb.after != nil {
		r.cb.after(res)
	}

	result.Res = Res{
		Status: res.Status,
		Code:   res.StatusCode,
	}

	// Determine status based on HTTP status code (200-299 = success, others = failure)
	status := 1 // default to failure
	if res.StatusCode >= 200 && res.StatusCode < 300 {
		status = 0 // success
	}
	result.Status = status

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return result, err
	}

	// Process body based on Content-Type
	contentType := res.Header.Get("Content-Type")
	bodyString, filePath, err := probe.ProcessHttpBody(body, contentType)
	if err != nil {
		return result, err
	}
	result.Res.Body = bodyString
	result.Res.FilePath = filePath

	header := make(map[string]string)
	for k, v := range res.Header {
		// examples:
		//   Set-Cookie: sessionid=abc123; Path=/; HttpOnly
		//   Accept: text/html, application/xhtml+xml, application/xml;q=0.9
		header[k] = strings.Join(v, ", ")
	}
	result.Res.Header = header

	return result, nil
}

type Option func(*Callback)

type Callback struct {
	before func(req *hp.Request)
	after  func(res *hp.Response)
}

var httpMethods = []string{
	hp.MethodGet,
	hp.MethodHead,
	hp.MethodPost,
	hp.MethodPut,
	hp.MethodPatch,
	hp.MethodDelete,
	hp.MethodConnect,
	hp.MethodOptions,
	hp.MethodTrace,
}

// ResolveMethodAndURL resolves HTTP method fields and updates the data map
// Converts method fields like "get", "post" to "method" and "url" fields
func ResolveMethodAndURL(data map[string]any) error {
	for _, method := range httpMethods {
		lowerMethod := strings.ToLower(method)
		routeValue, ok := data[lowerMethod]
		if !ok {
			continue
		}

		route, ok := routeValue.(string)
		if !ok {
			return fmt.Errorf("method field %s must be a string", lowerMethod)
		}

		data["method"] = method
		delete(data, lowerMethod)

		// If route is a complete URL (starts with http:// or https://), use it directly
		if strings.HasPrefix(route, "http://") || strings.HasPrefix(route, "https://") {
			data["url"] = route
		} else {
			// If route is a relative path, combine with base URL
			baseURLValue, ok := data["url"]
			if !ok {
				return errors.New("url is missing for relative path")
			}

			baseURL, ok := baseURLValue.(string)
			if !ok {
				return fmt.Errorf("base URL must be a string")
			}

			// renew url as full-url
			u, err := url.Parse(baseURL)
			if err != nil {
				return err
			}
			u.Path = path.Join(u.Path, route)
			data["url"] = u.String()
		}

		break
	}

	return nil
}

func Request(data map[string]any, opts ...Option) (map[string]any, error) {
	// Create a copy to avoid modifying the original data
	m := make(map[string]any)
	for k, v := range data {
		m[k] = v
	}

	// Resolve HTTP method fields (get, post, etc.) to method and url
	if err := ResolveMethodAndURL(m); err != nil {
		return map[string]any{}, err
	}

	// Handle body conversion for JSON content-type by checking headers
	if bodyData, bodyExists := data["body"]; bodyExists {
		isJSON := false
		if headers, ok := data["headers"].(map[string]any); ok {
			for k, v := range headers {
				if strings.EqualFold(k, "content-type") && v == "application/json" {
					isJSON = true
					break
				}
			}
		} else if headers, ok := data["headers"].(map[string]string); ok {
			for k, v := range headers {
				if strings.EqualFold(k, "content-type") && v == "application/json" {
					isJSON = true
					break
				}
			}
		} else if headers, ok := data["headers"].(map[string]interface{}); ok {
			for k, v := range headers {
				if strings.EqualFold(k, "content-type") && v == "application/json" {
					isJSON = true
					break
				}
			}
		}
		if isJSON {
			// Convert body to JSON string (supports both map and slice)
			switch body := bodyData.(type) {
			case map[string]any:
				if jsonBytes, err := json.Marshal(body); err == nil {
					m["body"] = string(jsonBytes)
				}
			case []any:
				if jsonBytes, err := json.Marshal(body); err == nil {
					m["body"] = string(jsonBytes)
				}
			}
		}
	}

	m = probe.HeaderToStringValue(m)

	// Extract custom headers and merge with defaults before MapToStructByTags
	var customHeaders map[string]string
	if headersInterface, exists := m["headers"]; exists {
		hclog.Default().Info("DEBUG: Processing headers", "headers", headersInterface, "type", fmt.Sprintf("%T", headersInterface))
		if headers, ok := headersInterface.(map[string]string); ok {
			customHeaders = headers
			hclog.Default().Info("DEBUG: Headers matched map[string]string")
		} else if headersInterfaceMap, ok := headersInterface.(map[string]interface{}); ok {
			// Convert map[string]interface{} to map[string]string
			customHeaders = make(map[string]string)
			for k, v := range headersInterfaceMap {
				if strVal, ok := v.(string); ok {
					customHeaders[k] = strVal
				}
			}
			hclog.Default().Info("DEBUG: Headers matched map[string]interface{}")
		} else if headersAnyMap, ok := headersInterface.(map[string]any); ok {
			// Convert map[string]any to map[string]string
			customHeaders = make(map[string]string)
			for k, v := range headersAnyMap {
				if strVal, ok := v.(string); ok {
					customHeaders[k] = strVal
				}
			}
			hclog.Default().Info("DEBUG: Headers matched map[string]any")
		} else {
			hclog.Default().Info("DEBUG: Headers type not matched")
		}
		hclog.Default().Info("DEBUG: Extracted custom headers", "customHeaders", customHeaders)
	}

	// Create new request with merged headers
	r := NewReq()
	r.Header = mergeHeaders(r.Header, customHeaders)

	// Update the map with merged headers to avoid duplication in MapToStructByTags
	m["headers"] = r.Header

	cb := &Callback{}
	for _, opt := range opts {
		opt(cb)
	}
	r.cb = cb

	if err := probe.MapToStructByTags(m, r); err != nil {
		return map[string]any{}, err
	}

	ret, err := r.Do()
	if err != nil {
		return map[string]any{}, err
	}

	mapRet, err := probe.StructToMapByTags(ret)
	if err != nil {
		return map[string]any{}, err
	}

	// Return the result directly without flattening
	return mapRet, nil
}

func WithBefore(f func(req *hp.Request)) Option {
	return func(c *Callback) {
		c.before = f
	}
}

func WithAfter(f func(res *hp.Response)) Option {
	return func(c *Callback) {
		c.after = f
	}
}
