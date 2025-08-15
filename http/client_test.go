package http

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/linyows/probe"
)

func TestNewReq(t *testing.T) {
	got := NewReq()

	expects := &Req{
		URL:    "",
		Method: "GET",
		Proto:  "HTTP/1.1",
		Header: map[string]string{
			"Accept":     "*/*",
			"User-Agent": "probe-http/1.0.0",
		},
	}

	if !reflect.DeepEqual(got, expects) {
		t.Errorf("\nExpected:\n%#v\nGot:\n%#v", expects, got)
	}
}

func TestDo(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	res := httpmock.NewStringResponder(200, "Hello World\n")
	httpmock.RegisterResponder("GET", "http://localhost:8080/foo/bar", res)

	req := NewReq()
	req.URL = "http://localhost:8080/foo/bar"

	got, err := req.Do()
	if err != nil {
		t.Errorf("got error %s", err)
	}

	expects := "Hello World\n"

	if string(got.Res.Body) != expects {
		t.Errorf("\nExpected:\n%s\nGot:\n%s", expects, got.Res.Body)
	}

	// Check that RT field is populated
	if got.RT <= 0 {
		t.Errorf("RT should be greater than 0, got: %v", got.RT)
	}
}

func TestConvertBodyToJson(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]string
		expected string
		hasBody  bool
	}{
		{
			name: "simple flat structure",
			input: map[string]string{
				"body__name": "test",
				"body__age":  "25",
				"method":     "POST",
			},
			expected: `{"age":25,"name":"test"}`,
			hasBody:  true,
		},
		{
			name: "nested structure",
			input: map[string]string{
				"body__foo__name": "aaa",
				"body__foo__role": "bbb",
				"body__bar":       "xyz",
				"method":          "POST",
			},
			expected: `{"bar":"xyz","foo":{"name":"aaa","role":"bbb"}}`,
			hasBody:  true,
		},
		{
			name: "array structure",
			input: map[string]string{
				"body__0__foo": "1",
				"body__0__bar": "2",
				"body__0__baz": "3",
				"method":       "POST",
			},
			expected: `[{"bar":2,"baz":3,"foo":1}]`,
			hasBody:  true,
		},
		{
			name: "multiple array items",
			input: map[string]string{
				"body__0__name": "item1",
				"body__1__name": "item2",
				"body__2__name": "item3",
				"method":        "POST",
			},
			expected: `[{"name":"item1"},{"name":"item2"},{"name":"item3"}]`,
			hasBody:  true,
		},
		{
			name: "deeply nested structure",
			input: map[string]string{
				"body__user__profile__name":    "John",
				"body__user__profile__age":     "30",
				"body__user__settings__theme":  "dark",
				"body__user__settings__notify": "true",
				"method":                       "POST",
			},
			expected: `{"user":{"profile":{"age":30,"name":"John"},"settings":{"notify":"true","theme":"dark"}}}`,
			hasBody:  true,
		},
		{
			name: "mixed data types",
			input: map[string]string{
				"body__count":   "42",
				"body__price":   "19.99",
				"body__active":  "true",
				"body__message": "hello world",
				"method":        "POST",
			},
			expected: `{"active":"true","count":42,"message":"hello world","price":19.99}`,
			hasBody:  true,
		},
		{
			name: "no body data",
			input: map[string]string{
				"method": "GET",
				"url":    "http://example.com",
			},
			expected: "",
			hasBody:  false,
		},
		{
			name: "empty body prefix",
			input: map[string]string{
				"method":  "POST",
				"headers": "application/json",
			},
			expected: "",
			hasBody:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy of input to avoid modifying the test case
			data := make(map[string]string)
			for k, v := range tt.input {
				data[k] = v
			}

			err := probe.ConvertBodyToJson(data)
			if err != nil {
				t.Errorf("ConvertBodyToJson() error = %v", err)
				return
			}

			if tt.hasBody {
				body, exists := data["body"]
				if !exists {
					t.Errorf("expected body key to exist in result")
					return
				}

				// Parse both JSON strings to compare structure, not formatting
				var expectedJSON, actualJSON interface{}

				if err := json.Unmarshal([]byte(tt.expected), &expectedJSON); err != nil {
					t.Errorf("failed to parse expected JSON: %v", err)
					return
				}

				if err := json.Unmarshal([]byte(body), &actualJSON); err != nil {
					t.Errorf("failed to parse actual JSON: %v", err)
					return
				}

				expectedBytes, _ := json.Marshal(expectedJSON)
				actualBytes, _ := json.Marshal(actualJSON)

				if string(expectedBytes) != string(actualBytes) {
					t.Errorf("ConvertBodyToJson() body = %v, want %v", body, tt.expected)
				}

				// Verify body__ keys are removed
				for key := range data {
					if key != "body" && key != "method" && key != "url" && key != "headers" {
						t.Errorf("unexpected key remaining: %s", key)
					}
				}
			} else {
				if _, exists := data["body"]; exists {
					t.Errorf("expected no body key when hasBody is false")
				}
			}
		})
	}
}

func TestConvertNumericStrings(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected map[string]any
	}{
		{
			name: "convert integers",
			input: map[string]any{
				"age":   "25",
				"count": "100",
				"name":  "test",
			},
			expected: map[string]any{
				"age":   25,
				"count": 100,
				"name":  "test",
			},
		},
		{
			name: "convert floats",
			input: map[string]any{
				"price":  "19.99",
				"weight": "2.5",
				"name":   "product",
			},
			expected: map[string]any{
				"price":  19.99,
				"weight": 2.5,
				"name":   "product",
			},
		},
		{
			name: "nested structures",
			input: map[string]any{
				"user": map[string]any{
					"age":  "30",
					"name": "John",
					"settings": map[string]any{
						"timeout": "5000",
						"enabled": "true",
					},
				},
				"count": "42",
			},
			expected: map[string]any{
				"user": map[string]any{
					"age":  30,
					"name": "John",
					"settings": map[string]any{
						"timeout": 5000,
						"enabled": "true",
					},
				},
				"count": 42,
			},
		},
		{
			name: "preserve non-numeric strings",
			input: map[string]any{
				"message": "hello123",
				"code":    "abc123",
				"mixed":   "123abc",
			},
			expected: map[string]any{
				"message": "hello123",
				"code":    "abc123",
				"mixed":   "123abc",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := probe.ConvertNumericStrings(tt.input)

			// Convert to JSON for easy comparison
			expectedJSON, _ := json.Marshal(tt.expected)
			actualJSON, _ := json.Marshal(result)

			if string(expectedJSON) != string(actualJSON) {
				t.Errorf("ConvertNumericStrings() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestResolveMethodAndURL(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]string
		expected map[string]string
		wantErr  bool
	}{
		{
			name: "GET method with relative path",
			input: map[string]string{
				"get": "/users",
				"url": "https://api.example.com",
			},
			expected: map[string]string{
				"method": "GET",
				"url":    "https://api.example.com/users",
			},
			wantErr: false,
		},
		{
			name: "POST method with complete HTTPS URL",
			input: map[string]string{
				"post": "https://api.example.com/users",
			},
			expected: map[string]string{
				"method": "POST",
				"url":    "https://api.example.com/users",
			},
			wantErr: false,
		},
		{
			name: "Missing URL with relative path",
			input: map[string]string{
				"get": "/users",
			},
			expected: map[string]string{
				"get": "/users",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ResolveMethodAndURL(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			for key, expectedValue := range tt.expected {
				actualValue, exists := tt.input[key]
				if !exists {
					t.Errorf("Expected key %s not found in result", key)
					continue
				}
				if actualValue != expectedValue {
					t.Errorf("Expected %s=%s, got %s=%s", key, expectedValue, key, actualValue)
				}
			}
		})
	}
}

func TestMergeHeaders(t *testing.T) {
	tests := []struct {
		name           string
		defaultHeaders map[string]string
		customHeaders  map[string]string
		expected       map[string]string
	}{
		{
			name: "no custom headers",
			defaultHeaders: map[string]string{
				"Accept":     "*/*",
				"User-Agent": "probe-http/1.0.0",
			},
			customHeaders: nil,
			expected: map[string]string{
				"Accept":     "*/*",
				"User-Agent": "probe-http/1.0.0",
			},
		},
		{
			name: "override user-agent lowercase",
			defaultHeaders: map[string]string{
				"Accept":     "*/*",
				"User-Agent": "probe-http/1.0.0",
			},
			customHeaders: map[string]string{
				"user-agent": "custom-agent/1.0",
			},
			expected: map[string]string{
				"Accept":     "*/*",
				"user-agent": "custom-agent/1.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeHeaders(tt.defaultHeaders, tt.customHeaders)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d headers, got %d", len(tt.expected), len(result))
			}

			for expectedKey, expectedValue := range tt.expected {
				actualValue, exists := result[expectedKey]
				if !exists {
					t.Errorf("Expected header %s not found", expectedKey)
					continue
				}
				if actualValue != expectedValue {
					t.Errorf("Expected %s = %s, got %s", expectedKey, expectedValue, actualValue)
				}
			}

			// Check for duplicates (case-insensitive)
			userAgentCount := 0
			for key := range result {
				if strings.ToLower(key) == "user-agent" {
					userAgentCount++
				}
			}
			if userAgentCount > 1 {
				t.Errorf("Found %d User-Agent headers, expected only 1. Headers: %v", userAgentCount, result)
			}
		})
	}
}
