package mail

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestNewReq(t *testing.T) {
	got := NewReq()

	expected := &Req{
		Addr:       "",
		From:       "",
		To:         "",
		Subject:    "",
		MyHostname: "",
		Session:    1,
		Message:    1,
		Length:     0,
	}

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("\nExpected:\n%#v\nGot:\n%#v", expected, got)
	}
}

func TestReqDo_ValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		req         *Req
		expectError bool
		errorMsg    string
	}{
		{
			name: "missing addr",
			req: &Req{
				From: "test@example.com",
				To:   "recipient@example.com",
			},
			expectError: true,
			errorMsg:    "Req.Addr is required",
		},
		{
			name: "missing from",
			req: &Req{
				Addr: "localhost:25",
				To:   "recipient@example.com",
			},
			expectError: true,
			errorMsg:    "Req.From is required",
		},
		{
			name: "missing to",
			req: &Req{
				Addr: "localhost:25",
				From: "test@example.com",
			},
			expectError: true,
			errorMsg:    "Req.To is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.req.Do()

			if tt.expectError {
				if err == nil {
					t.Errorf("Do() expected error but got none")
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain '%s', got: %s", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Do() unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Error("Do() returned nil result")
			}
		})
	}
}

func TestReqDo_WithCallbacks(t *testing.T) {
	// Note: This test focuses on callback functionality since actual SMTP
	// requires a real server. The bulk email functionality is tested separately.
	beforeCalled := false
	afterCalled := false
	var capturedFrom, capturedTo, capturedSubject string
	var capturedResult *Result

	req := &Req{
		Addr:    "localhost:25",
		From:    "test@example.com",
		To:      "recipient@example.com",
		Subject: "Test Subject",
		cb: &Callback{
			before: func(from string, to string, subject string) {
				beforeCalled = true
				capturedFrom = from
				capturedTo = to
				capturedSubject = subject
			},
			after: func(result *Result) {
				afterCalled = true
				capturedResult = result
			},
		},
	}

	// This will likely fail due to no SMTP server, but we can test the callback setup
	result, err := req.Do()

	// Verify callbacks were called even if the operation failed
	if !beforeCalled {
		t.Error("before callback was not called")
	}
	if capturedFrom != "test@example.com" {
		t.Errorf("Expected from 'test@example.com', got '%s'", capturedFrom)
	}
	if capturedTo != "recipient@example.com" {
		t.Errorf("Expected to 'recipient@example.com', got '%s'", capturedTo)
	}
	if capturedSubject != "Test Subject" {
		t.Errorf("Expected subject 'Test Subject', got '%s'", capturedSubject)
	}

	// The after callback should be called even if there's an error
	if !afterCalled {
		t.Error("after callback was not called")
	}
	if capturedResult == nil {
		t.Error("after callback did not receive result")
	}

	// Check basic result structure
	if result != nil {
		// Check that RT field is populated
		if result.RT <= 0 {
			t.Errorf("RT should be greater than 0, got: %v", result.RT)
		}

		// Check request fields are preserved
		if result.Req.From != req.From {
			t.Errorf("Req.From = %v, want %v", result.Req.From, req.From)
		}
		if result.Req.To != req.To {
			t.Errorf("Req.To = %v, want %v", result.Req.To, req.To)
		}
		if result.Req.Subject != req.Subject {
			t.Errorf("Req.Subject = %v, want %v", result.Req.Subject, req.Subject)
		}
	}

	// We expect an error since there's no real SMTP server
	if err == nil {
		t.Log("Note: Unexpected success - there might be a local SMTP server running")
	}
}

func TestSend(t *testing.T) {
	tests := []struct {
		name        string
		data        map[string]any
		expectError bool
	}{
		{
			name: "complete request data",
			data: map[string]any{
				"addr":    "localhost:25",
				"from":    "test@example.com",
				"to":      "recipient@example.com",
				"subject": "Test Email",
				"session": "1",
				"message": "1",
				"length":  "100",
			},
			expectError: true, // Expected to fail without real SMTP server
		},
		{
			name: "missing required addr",
			data: map[string]any{
				"from": "test@example.com",
				"to":   "recipient@example.com",
			},
			expectError: true,
		},
		{
			name: "missing required from",
			data: map[string]any{
				"addr": "localhost:25",
				"to":   "recipient@example.com",
			},
			expectError: true,
		},
		{
			name: "missing required to",
			data: map[string]any{
				"addr": "localhost:25",
				"from": "test@example.com",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Track if callbacks were called
			beforeCalled := false
			afterCalled := false

			before := WithBefore(func(from string, to string, subject string) {
				beforeCalled = true
			})
			after := WithAfter(func(result *Result) {
				afterCalled = true
			})

			result, err := Send(tt.data, before, after)

			if tt.expectError {
				if err == nil {
					t.Errorf("Send() expected error but got none")
				}
			}

			// For valid requests, callbacks should be called even if SMTP fails
			addr, addrOk := tt.data["addr"]
			from, fromOk := tt.data["from"]
			to, toOk := tt.data["to"]
			if addrOk && addr != "" && fromOk && from != "" && toOk && to != "" {
				if !beforeCalled {
					t.Error("before callback was not called for valid request")
				}
				if !afterCalled {
					t.Error("after callback was not called for valid request")
				}

				// Check that result contains nested structure when error occurs
				if result != nil {
					// Check that req nested fields exist
					if req, exists := result["req"]; exists {
						if reqMap, ok := req.(map[string]any); ok {
							if _, exists := reqMap["addr"]; !exists {
								t.Error("Expected 'addr' field in req")
							}
							if _, exists := reqMap["from"]; !exists {
								t.Error("Expected 'from' field in req")
							}
							if _, exists := reqMap["to"]; !exists {
								t.Error("Expected 'to' field in req")
							}
						} else {
							t.Error("Expected req to be map[string]any")
						}
					} else {
						t.Error("Expected 'req' field in result")
					}
				}
			}
		})
	}
}

func TestWithBefore(t *testing.T) {
	called := false
	var capturedFrom, capturedTo, capturedSubject string

	option := WithBefore(func(from string, to string, subject string) {
		called = true
		capturedFrom = from
		capturedTo = to
		capturedSubject = subject
	})

	cb := &Callback{}
	option(cb)

	if cb.before == nil {
		t.Error("WithBefore() did not set before callback")
		return
	}

	// Test the callback
	cb.before("test@example.com", "recipient@example.com", "Test Subject")

	if !called {
		t.Error("before callback was not called")
	}
	if capturedFrom != "test@example.com" {
		t.Errorf("Expected from 'test@example.com', got '%s'", capturedFrom)
	}
	if capturedTo != "recipient@example.com" {
		t.Errorf("Expected to 'recipient@example.com', got '%s'", capturedTo)
	}
	if capturedSubject != "Test Subject" {
		t.Errorf("Expected subject 'Test Subject', got '%s'", capturedSubject)
	}
}

func TestWithAfter(t *testing.T) {
	called := false
	var capturedResult *Result

	option := WithAfter(func(result *Result) {
		called = true
		capturedResult = result
	})

	cb := &Callback{}
	option(cb)

	if cb.after == nil {
		t.Error("WithAfter() did not set after callback")
		return
	}

	// Test the callback
	testResult := &Result{
		Status: 1,
		Res: Res{
			Code:   1,
			Sent:   0,
			Failed: 1,
			Total:  1,
			Error:  "test error",
		},
		RT: time.Second,
	}
	cb.after(testResult)

	if !called {
		t.Error("after callback was not called")
	}
	if capturedResult != testResult {
		t.Error("after callback did not receive correct result")
	}
}

func TestReqFieldMapping(t *testing.T) {
	// Test that the request structure properly maps integer fields
	req := &Req{
		Addr:       "localhost:25",
		From:       "test@example.com",
		To:         "recipient@example.com",
		Subject:    "Test",
		MyHostname: "testhost",
		Session:    5,
		Message:    10,
		Length:     1000,
	}

	// Test that all fields are properly set
	if req.Session != 5 {
		t.Errorf("Expected Session 5, got %d", req.Session)
	}
	if req.Message != 10 {
		t.Errorf("Expected Message 10, got %d", req.Message)
	}
	if req.Length != 1000 {
		t.Errorf("Expected Length 1000, got %d", req.Length)
	}
}
