package http

import (
	"bytes"
	"errors"
	"io"
	hp "net/http"
	"net/url"
	"path"
	"strings"
	"time"

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
	Body   []byte            `map:"body"`
	cb     *Callback
}

type Res struct {
	Status string            `map:"status"`
	Code   int               `map:"code"`
	Header map[string]string `map:"headers"`
	Body   []byte            `map:"body"`
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

	req, err := hp.NewRequest(r.Method, r.URL, bytes.NewBuffer(r.Body))
	if err != nil {
		return nil, err
	}

	for k, v := range r.Header {
		req.Header.Set(probe.TitleCase(k, "-"), v)
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
	result.Res.Body = body

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
func ResolveMethodAndURL(data map[string]string) error {
	for _, method := range httpMethods {
		lowerMethod := strings.ToLower(method)
		route, ok := data[lowerMethod]
		if !ok {
			continue
		}

		data["method"] = method
		delete(data, lowerMethod)

		// If route is a complete URL (starts with http:// or https://), use it directly
		if strings.HasPrefix(route, "http://") || strings.HasPrefix(route, "https://") {
			data["url"] = route
		} else {
			// If route is a relative path, combine with base URL
			baseURL, ok := data["url"]
			if !ok {
				return errors.New("url is missing for relative path")
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

// ConvertBodyToFormEncoded converts body__ prefixed fields to form-encoded body
func ConvertBodyToFormEncoded(data map[string]string) error {
	values := url.Values{}

	for key, value := range data {
		if strings.HasPrefix(key, "body__") {
			newKey := strings.TrimPrefix(key, "body__")
			values.Add(newKey, value)
			delete(data, key)
		}
	}

	if len(values) > 0 {
		data["body"] = values.Encode()
		data["headers__content-type"] = "application/x-www-form-urlencoded"
	}

	return nil
}

// PrepareRequestData prepares all request data including method fields and body conversion
func PrepareRequestData(data map[string]string) error {
	// Resolve HTTP method fields first
	if err := ResolveMethodAndURL(data); err != nil {
		return err
	}

	// Handle body conversion based on content-type
	contentType, exists := data["headers__content-type"]
	if exists && contentType == "application/json" {
		if err := probe.ConvertBodyToJson(data); err != nil {
			return err
		}
	} else {
		if err := ConvertBodyToFormEncoded(data); err != nil {
			return err
		}
	}

	return nil
}

func Request(data map[string]string, opts ...Option) (map[string]string, error) {
	// Create a copy to avoid modifying the original data
	dataCopy := make(map[string]string)
	for k, v := range data {
		dataCopy[k] = v
	}

	// Prepare request data (resolve method fields and convert body)
	if err := PrepareRequestData(dataCopy); err != nil {
		return map[string]string{}, err
	}

	m := probe.HeaderToStringValue(probe.UnflattenInterface(dataCopy))

	// Extract custom headers and merge with defaults before MapToStructByTags
	var customHeaders map[string]string
	if headersInterface, exists := m["headers"]; exists {
		if headers, ok := headersInterface.(map[string]string); ok {
			customHeaders = headers
		} else if headersInterfaceMap, ok := headersInterface.(map[string]interface{}); ok {
			// Convert map[string]interface{} to map[string]string
			customHeaders = make(map[string]string)
			for k, v := range headersInterfaceMap {
				if strVal, ok := v.(string); ok {
					customHeaders[k] = strVal
				}
			}
		}
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
		return map[string]string{}, err
	}

	ret, err := r.Do()
	if err != nil {
		return map[string]string{}, err
	}

	mapRet, err := probe.StructToMapByTags(ret)
	if err != nil {
		return map[string]string{}, err
	}

	return probe.FlattenInterface(mapRet), nil
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
