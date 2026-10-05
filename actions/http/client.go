package http

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"mime"
	hp "net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/linyows/probe/binary"
	"github.com/linyows/probe/mapping"
)

// DefaultTimeout bounds a request when the step does not set `timeout`. The
// zero value of http.Client means "wait forever", which lets a single
// unresponsive endpoint hold a workflow open until the step timeout fires.
const DefaultTimeout = 30 * time.Second

type Req struct {
	URL     string            `map:"url" validate:"required"`
	Method  string            `map:"method" validate:"required"`
	Header  map[string]string `map:"headers"`
	Body    string            `map:"body"` // Changed from []byte to string for text data
	Timeout string            `map:"timeout"`
	cb      *Callback
	// payload is sent in place of Body when it is set. It holds a multipart
	// body, which may carry the bytes of files, and is never shown.
	payload []byte
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
		Header: map[string]string{
			"Accept":     "*/*",
			"User-Agent": "probe-http/1.0.0",
		},
		Timeout: DefaultTimeout.String(),
	}
}

// parseTimeout turns the request's timeout field into a duration. It accepts a
// Go duration string ("10s", "1m30s") and, like the workflow-level interval
// fields, a bare number of seconds. An empty value falls back to
// DefaultTimeout, and zero disables the limit.
func parseTimeout(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return DefaultTimeout, nil
	}

	d, err := time.ParseDuration(s)
	if err != nil {
		sec, ferr := strconv.ParseFloat(s, 64)
		if ferr != nil {
			return 0, fmt.Errorf("invalid timeout: %q", s)
		}
		d = time.Duration(sec * float64(time.Second))
	}

	if d < 0 {
		return 0, fmt.Errorf("timeout must not be negative: %q", s)
	}

	return d, nil
}

// mergeHeaders merges custom headers with default headers, handling case-insensitive duplicates
// Custom headers override defaults when header names match (case-insensitive)
func mergeHeaders(defaultHeaders, customHeaders map[string]string) map[string]string {
	if customHeaders == nil {
		return defaultHeaders
	}

	result := make(map[string]string)

	// First, copy all default headers
	maps.Copy(result, defaultHeaders)

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

	timeout, err := parseTimeout(r.Timeout)
	if err != nil {
		return nil, err
	}

	var reqBody io.Reader = strings.NewReader(r.Body)
	if r.payload != nil {
		reqBody = bytes.NewReader(r.payload)
	}
	req, err := hp.NewRequest(r.Method, r.URL, reqBody)
	if err != nil {
		return nil, err
	}

	for k, v := range r.Header {
		// Clean header value by removing newlines and other invalid characters
		cleanValue := strings.ReplaceAll(strings.ReplaceAll(v, "\n", ""), "\r", "")
		req.Header.Set(mapping.TitleCase(k, "-"), cleanValue)
	}

	// callback
	if r.cb != nil && r.cb.before != nil {
		r.cb.before(req)
	}

	result := &Result{Req: *r}

	cl := &hp.Client{Timeout: timeout}
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
	bodyString, filePath, err := binary.ProcessHttpBody(body, contentType)
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
			routeURL, err := url.Parse(route)
			if err != nil {
				return fmt.Errorf("invalid route: %w", err)
			}
			u.Path = path.Join(u.Path, routeURL.Path)
			if routeURL.RawQuery != "" {
				u.RawQuery = routeURL.RawQuery
			}
			data["url"] = u.String()
		}

		break
	}

	return nil
}

// hasJSONContentType checks if headers declare a JSON Content-Type, such as
// application/json; charset=utf-8 or application/problem+json.
// Supports multiple header map types and case-insensitive matching
func hasJSONContentType(headers any) bool {
	checkHeader := func(k string, v any) bool {
		s, ok := v.(string)
		return ok && strings.EqualFold(k, "content-type") && isJSONMediaType(s)
	}

	switch h := headers.(type) {
	case map[string]any: // also matches map[string]interface{}
		for k, v := range h {
			if checkHeader(k, v) {
				return true
			}
		}
	case map[string]string:
		for k, v := range h {
			if checkHeader(k, v) {
				return true
			}
		}
	}
	return false
}

// isJSONMediaType reports whether a Content-Type value names JSON, ignoring
// parameters such as charset.
func isJSONMediaType(v string) bool {
	mediaType, _, err := mime.ParseMediaType(v)
	if err != nil {
		return false
	}
	return mediaType == "application/json" || strings.HasSuffix(mediaType, "+json")
}

// MarshalBodyIfJSON converts body to JSON string when Content-Type is application/json
// Supports map[string]any and []any body types
func MarshalBodyIfJSON(data, m map[string]any) {
	bodyData, bodyExists := data["body"]
	if !bodyExists {
		return
	}

	if !hasJSONContentType(data["headers"]) {
		return
	}

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

func Request(data map[string]any, opts ...Option) (map[string]any, error) {
	// Create a copy to avoid modifying the original data
	m := make(map[string]any)
	maps.Copy(m, data)

	// Resolve HTTP method fields (get, post, etc.) to method and url
	if err := ResolveMethodAndURL(m); err != nil {
		return map[string]any{}, err
	}

	// form and multipart are turned into the body they stand for.
	payload, multipartSpec, contentType, err := takeFormBody(m)
	if err != nil {
		return map[string]any{}, err
	}

	// Handle body conversion for JSON content-type
	MarshalBodyIfJSON(data, m)

	m = mapping.HeaderToStringValue(m)

	// Extract custom headers and merge with defaults before MapToStructByTags
	var customHeaders map[string]string
	if headersInterface, exists := m["headers"]; exists {
		if headers, ok := headersInterface.(map[string]string); ok {
			// A copy, since basic_auth, form and multipart change it.
			customHeaders = maps.Clone(headers)
		} else if headersInterfaceMap, ok := headersInterface.(map[string]any); ok {
			// Convert map[string]interface{} to map[string]string
			customHeaders = make(map[string]string)
			for k, v := range headersInterfaceMap {
				if strVal, ok := v.(string); ok {
					customHeaders[k] = strVal
				}
			}
		}
	}

	// basic_auth is turned into the Authorization header it stands for.
	if auth, exists := m["basic_auth"]; exists {
		delete(m, "basic_auth")
		value, err := basicAuthHeader(auth)
		if err != nil {
			return map[string]any{}, err
		}
		for k := range customHeaders {
			if strings.EqualFold(k, "authorization") {
				return map[string]any{}, errors.New("basic_auth and an authorization header cannot be given together")
			}
		}
		if customHeaders == nil {
			customHeaders = make(map[string]string)
		}
		customHeaders["authorization"] = value
	}

	// The body built from form or multipart is sent with its own Content-Type,
	// which replaces one set for the job's other requests, such as JSON in
	// defaults; a multipart one also carries the boundary only it knows.
	if contentType != "" {
		for k := range customHeaders {
			if strings.EqualFold(k, "content-type") {
				delete(customHeaders, k)
			}
		}
		if customHeaders == nil {
			customHeaders = make(map[string]string)
		}
		customHeaders["content-type"] = contentType
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

	if err := mapping.MapToStructByTags(m, r); err != nil {
		return map[string]any{}, err
	}
	r.payload = payload

	ret, err := r.Do()
	if err != nil {
		return map[string]any{}, err
	}

	mapRet, err := mapping.StructToMapByTags(ret)
	if err != nil {
		return map[string]any{}, err
	}

	// A multipart request shows what was written in place of the body sent.
	if multipartSpec != nil {
		if req, ok := mapRet["req"].(map[string]any); ok {
			req["multipart"] = multipartSpec
		}
	}

	// Return the result directly without flattening
	return mapRet, nil
}

// basicAuthHeader builds the value of an Authorization header for HTTP Basic
// authentication from basic_auth, a map of username and password, as RFC 7617
// defines it. A password may be a number, as a YAML value written without
// quotes is, and may be empty; a username may not hold a colon, which would
// end it early.
func basicAuthHeader(v any) (string, error) {
	auth, ok := v.(map[string]any)
	if !ok {
		return "", errors.New("basic_auth must be a map of username and password")
	}
	for k := range auth {
		if k != "username" && k != "password" {
			return "", fmt.Errorf("basic_auth takes username and password, not %s", k)
		}
	}
	username, err := basicAuthField(auth, "username")
	if err != nil {
		return "", err
	}
	if username == "" {
		return "", errors.New("basic_auth.username is required")
	}
	if strings.Contains(username, ":") {
		return "", errors.New("basic_auth.username must not contain a colon")
	}
	password, err := basicAuthField(auth, "password")
	if err != nil {
		return "", err
	}
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+password)), nil
}

// basicAuthField returns the field key of basic_auth as a string: a string
// as it is, a number or a boolean as written, and a missing one as empty.
func basicAuthField(auth map[string]any, key string) (string, error) {
	switch v := auth[key].(type) {
	case nil:
		return "", nil
	case string:
		return v, nil
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	case int, int64, uint64, bool:
		return fmt.Sprint(v), nil
	default:
		return "", fmt.Errorf("basic_auth.%s must be a string", key)
	}
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
