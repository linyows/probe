package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	hp "net/http"
	"strconv"
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
	Req Req           `map:"req"`
	Res Res           `map:"res"`
	RT  time.Duration `map:"rt"`
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
	defer res.Body.Close()

	// callback
	if r.cb != nil && r.cb.after != nil {
		r.cb.after(res)
	}

	result.Res = Res{
		Status: res.Status,
		Code:   res.StatusCode,
	}

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

func HeaderToStringValue(data map[string]any) map[string]any {
	v, exists := data["headers"]
	if !exists {
		return data
	}

	newHeaders := make(map[string]any)
	if headers, ok := v.(map[string]any); ok {
		for key, value := range headers {
			switch v := value.(type) {
			case string:
				newHeaders[key] = v
			case int:
				newHeaders[key] = strconv.Itoa(v)
			case float64:
				newHeaders[key] = strconv.FormatFloat(v, 'f', -1, 64)
			default:
				newHeaders[key] = fmt.Sprintf("%v", v)
			}
		}
	}

	if len(newHeaders) > 0 {
		data["headers"] = newHeaders
	}

	return data
}

func Request(data map[string]string, opts ...Option) (map[string]string, error) {
	// Debug: Check if expressions are already expanded
	for key, value := range data {
		if strings.Contains(value, "{{") && strings.Contains(value, "}}") {
			fmt.Printf("DEBUG: Found unexpanded expression in %s: %s\n", key, value)
		}
	}

	m := HeaderToStringValue(probe.UnflattenInterface(data))
	r := NewReq()

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

// ConvertBodyToJson converts flat body data to properly nested JSON structure
func ConvertBodyToJson(data map[string]string) error {
	bodyData := map[string]string{}

	// Extract all body__ prefixed keys
	for key, value := range data {
		if strings.HasPrefix(key, "body__") {
			newKey := strings.TrimPrefix(key, "body__")
			bodyData[newKey] = value
			delete(data, key)
		}
	}

	if len(bodyData) > 0 {
		// Note: Expression expansion should already be done by this point
		// Just restore nested structure from flat keys and convert to arrays
		unflattenedData := probe.UnflattenInterface(bodyData)

		// Convert numeric strings to numbers and maps with numeric keys to arrays
		convertedData := ConvertNumericStringsAndArrays(unflattenedData)

		j, err := json.Marshal(convertedData)
		if err != nil {
			return err
		}
		data["body"] = string(j)
	}

	return nil
}

// ConvertNumericStrings recursively converts numeric strings to numbers in nested structures
func ConvertNumericStrings(data map[string]any) map[string]any {
	result := make(map[string]any)

	for key, value := range data {
		switch v := value.(type) {
		case string:
			// Try to convert numeric strings to numbers
			if num, err := strconv.Atoi(v); err == nil {
				result[key] = num
			} else if floatNum, err := strconv.ParseFloat(v, 64); err == nil {
				result[key] = floatNum
			} else {
				result[key] = v
			}
		case map[string]any:
			// Recursively process nested maps
			result[key] = ConvertNumericStrings(v)
		default:
			result[key] = v
		}
	}

	return result
}

// ConvertNumericStringsAndArrays recursively converts numeric strings to numbers and maps with numeric keys to arrays
func ConvertNumericStringsAndArrays(data map[string]any) any {
	// Check if this map should be converted to an array
	if shouldConvertToArray(data) {
		return convertMapToArray(data)
	}

	result := make(map[string]any)

	for key, value := range data {
		switch v := value.(type) {
		case string:
			// Try to convert numeric strings to numbers
			if num, err := strconv.Atoi(v); err == nil {
				result[key] = num
			} else if floatNum, err := strconv.ParseFloat(v, 64); err == nil {
				result[key] = floatNum
			} else {
				result[key] = v
			}
		case map[string]any:
			// Recursively process nested maps
			result[key] = ConvertNumericStringsAndArrays(v)
		default:
			result[key] = v
		}
	}

	return result
}

// shouldConvertToArray checks if a map should be converted to an array
func shouldConvertToArray(m map[string]any) bool {
	if len(m) == 0 {
		return false
	}

	// Check if all keys are numeric and form a complete sequence from 0 to len-1
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}

	return isNumericSequence(keys)
}

// isNumericSequence checks if the keys form a numeric sequence starting from 0
func isNumericSequence(keys []string) bool {
	if len(keys) == 0 {
		return false
	}

	// Convert all keys to integers and check if they form a sequence
	nums := make([]int, len(keys))
	for i, key := range keys {
		num, err := strconv.Atoi(key)
		if err != nil {
			return false
		}
		nums[i] = num
	}

	// Check if it's a complete sequence from 0 to len-1
	for i := 0; i < len(nums); i++ {
		found := false
		for _, num := range nums {
			if num == i {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// convertMapToArray converts a map with numeric keys to an array
func convertMapToArray(m map[string]any) []any {
	result := make([]any, len(m))

	for key, value := range m {
		index, _ := strconv.Atoi(key)
		switch v := value.(type) {
		case map[string]any:
			result[index] = ConvertNumericStringsAndArrays(v)
		default:
			result[index] = value
		}
	}

	return result
}
