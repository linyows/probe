package mail

import (
	"net"
	"net/textproto"
	"reflect"
	"strings"
	"sync/atomic"
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
		StartTLS:   StartTLSAuto,
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

// TestSend_MyHostname follows the myhostname parameter from the action's
// input through Bulk to the EHLO line, the path that dropped it before.
func TestSend_MyHostname(t *testing.T) {
	stubHostname(t, "machine-a", nil)
	srv := newGreetingRecorder(t)

	_, err := Send(map[string]any{
		"addr":       srv.addr(),
		"from":       "from@example.com",
		"to":         "to@example.com",
		"subject":    "hi",
		"myhostname": "probe-client.local",
		"session":    2,
		"message":    2,
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	got := srv.greetings()
	if len(got) != 2 {
		t.Fatalf("greetings = %q, want one per session", got)
	}
	for _, g := range got {
		if g != "EHLO probe-client.local" {
			t.Errorf("greeting = %q, want EHLO probe-client.local", g)
		}
	}
}

// startRejectingServer runs an SMTP server that refuses every recipient.
func startRejectingServer(t *testing.T) string {
	return startSMTPServer(t, func(int) bool { return true })
}

// startSMTPServer runs an SMTP server that refuses every recipient on the
// connections answer reports true for, counting from 0. The others are
// dropped without a word after a moment, so that they finish last.
func startSMTPServer(t *testing.T, answer func(n int) bool) string {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = lis.Close() })

	go func() {
		for n := 0; ; n++ {
			conn, err := lis.Accept()
			if err != nil {
				return
			}
			if !answer(n) {
				go func(conn net.Conn) {
					time.Sleep(100 * time.Millisecond)
					_ = conn.Close()
				}(conn)
				continue
			}
			go func(conn net.Conn) {
				defer func() { _ = conn.Close() }()
				text := textproto.NewConn(conn)
				_ = text.PrintfLine("220 test ESMTP")
				for {
					line, err := text.ReadLine()
					if err != nil {
						return
					}
					switch cmd := strings.ToUpper(strings.SplitN(line, " ", 2)[0]); cmd {
					case "RCPT":
						_ = text.PrintfLine("550 5.1.1 No such user")
					case "QUIT":
						_ = text.PrintfLine("221 bye")
						return
					default:
						_ = text.PrintfLine("250 ok")
					}
				}
			}(conn)
		}
	}()

	return lis.Addr().String()
}

func TestReqDo_Rejected(t *testing.T) {
	// The server answered, so the step gets a result to test rather than an
	// error.
	req := &Req{
		Addr:    startRejectingServer(t),
		From:    "from@example.com",
		To:      "nobody@example.com",
		Subject: "test",
		Session: 1,
		Message: 1,
	}
	result, err := req.Do()
	if err != nil {
		t.Fatalf("Do() error: %v", err)
	}
	if result.Status != 1 || result.Res.Code != 1 {
		t.Errorf("status = %d, code = %d, want 1 and 1", result.Status, result.Res.Code)
	}
	if !strings.Contains(result.Res.Error, "550") {
		t.Errorf("error = %q, want the server's reply", result.Res.Error)
	}
}

func TestReqDo_Unreachable(t *testing.T) {
	// Nothing listens there, so there is no answer to test.
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := lis.Addr().String()
	_ = lis.Close()

	req := &Req{
		Addr:    addr,
		From:    "from@example.com",
		To:      "nobody@example.com",
		Subject: "test",
		Session: 1,
		Message: 1,
	}
	if _, err := req.Do(); err == nil {
		t.Fatal("Do() to a closed port succeeded")
	}
}

func TestReqDo_RejectedAndDropped(t *testing.T) {
	// One session is refused and the other loses its connection, after the
	// refusal. The server answered once, so the outcome is a result whichever
	// session finishes last.
	for i := 0; i < 5; i++ {
		addr := startSMTPServer(t, func(n int) bool { return n%2 == 0 })
		req := &Req{
			Addr:    addr,
			From:    "from@example.com",
			To:      "nobody@example.com",
			Subject: "test",
			Session: 2,
			Message: 2,
		}
		result, err := req.Do()
		if err != nil {
			t.Fatalf("run %d: Do() error: %v", i, err)
		}
		if result.Res.Failed != 2 {
			t.Errorf("run %d: failed = %d, want 2", i, result.Res.Failed)
		}
		if !strings.Contains(result.Res.Error, "550") {
			t.Errorf("run %d: error = %q, want the server's reply", i, result.Res.Error)
		}
	}
}

func TestSendStartTLS(t *testing.T) {
	// The smtp action's starttls and insecure_skip_tls, against a server that
	// offers STARTTLS with a self-signed certificate.
	addr := startMockServer(t, true)
	params := func(extra map[string]any) map[string]any {
		p := map[string]any{"addr": addr, "from": "from@example.com", "to": "to@example.com", "subject": "test"}
		for k, v := range extra {
			p[k] = v
		}
		return p
	}

	// auto is the default, so STARTTLS is taken up and the certificate
	// checked, which a self-signed one fails
	if _, err := Send(params(nil)); err == nil || !strings.Contains(err.Error(), "certificate") {
		t.Errorf("default: error = %v, want a certificate error", err)
	}

	for _, extra := range []map[string]any{
		{"starttls": "required", "insecure_skip_tls": true},
		{"starttls": "AUTO", "insecure_skip_tls": true},
		{"starttls": "off"},
	} {
		ret, err := Send(params(extra))
		if err != nil {
			t.Errorf("%v: Send() error: %v", extra, err)
			continue
		}
		res, _ := ret["res"].(map[string]any)
		if res["code"] != 0 || res["sent"] != 1 {
			t.Errorf("%v: res = %v, want one message sent", extra, res)
		}
	}

	if _, err := Send(params(map[string]any{"starttls": "sometimes"})); err == nil || !strings.Contains(err.Error(), "off, auto or required") {
		t.Errorf("invalid starttls: error = %v, want the values there are", err)
	}
}

func TestSendStartTLSRequiredNotOffered(t *testing.T) {
	// required fails a server that does not offer STARTTLS, rather than
	// sending in plain text
	addr := startMockServer(t, false)
	_, err := Send(map[string]any{"addr": addr, "from": "from@example.com", "to": "to@example.com", "starttls": "required"})
	if err == nil || !strings.Contains(err.Error(), "does not offer STARTTLS") {
		t.Errorf("error = %v, want the server not offering STARTTLS", err)
	}
}

func TestSendStopsOnUnreadableParameter(t *testing.T) {
	// A parameter that cannot be read fails the step before anything is
	// sent; it used to send with the parameter at its zero value and report
	// the error only afterwards.
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = lis.Close() })
	var accepted atomic.Int32
	go func() {
		for {
			conn, err := lis.Accept()
			if err != nil {
				return
			}
			accepted.Add(1)
			_ = conn.Close()
		}
	}()

	_, err = Send(map[string]any{
		"addr":              lis.Addr().String(),
		"from":              "from@example.com",
		"to":                "to@example.com",
		"insecure_skip_tls": "invalid",
	})
	if err == nil {
		t.Fatal("Send() succeeded with an unreadable insecure_skip_tls")
	}
	time.Sleep(100 * time.Millisecond)
	if n := accepted.Load(); n != 0 {
		t.Errorf("the server got %d connections, want none", n)
	}
}
