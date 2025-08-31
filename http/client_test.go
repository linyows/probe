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
		input    map[string]any
		expected map[string]any
		wantErr  bool
	}{
		{
			name: "GET method with relative path",
			input: map[string]any{
				"get": "/users",
				"url": "https://api.example.com",
			},
			expected: map[string]any{
				"method": "GET",
				"url":    "https://api.example.com/users",
			},
			wantErr: false,
		},
		{
			name: "POST method with complete HTTPS URL",
			input: map[string]any{
				"post": "https://api.example.com/users",
			},
			expected: map[string]any{
				"method": "POST",
				"url":    "https://api.example.com/users",
			},
			wantErr: false,
		},
		{
			name: "Missing URL with relative path",
			input: map[string]any{
				"get": "/users",
			},
			expected: map[string]any{
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
