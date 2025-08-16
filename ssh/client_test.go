package ssh

import (
	"testing"
	"time"
)

func TestNewReq(t *testing.T) {
	req := NewReq()

	// Test default values
	if req.Port != 22 {
		t.Errorf("Expected default port 22, got %d", req.Port)
	}
	if req.Timeout != "30s" {
		t.Errorf("Expected default timeout '30s', got %s", req.Timeout)
	}
	if !req.StrictHostCheck {
		t.Errorf("Expected default StrictHostCheck true, got %v", req.StrictHostCheck)
	}
	if req.Env == nil {
		t.Errorf("Expected Env map to be initialized")
	}
}

func TestParseParams(t *testing.T) {
	tests := []struct {
		name      string
		req       *Req
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid request with password",
			req: &Req{
				Host:     "example.com",
				Port:     22,
				User:     "testuser",
				Cmd:      "ls -la",
				Password: "testpass",
				Timeout:  "30s",
			},
			wantError: false,
		},
		{
			name: "valid request with key file (but file validation skipped in this test)",
			req: &Req{
				Host:     "example.com",
				Port:     22,
				User:     "testuser",
				Cmd:      "ls -la",
				KeyFile:  "",         // Set empty to skip key file validation in parseParams test
				Password: "testpass", // Use password instead for this test
				Timeout:  "30s",
			},
			wantError: false,
		},
		{
			name: "missing host",
			req: &Req{
				Port:     22,
				User:     "testuser",
				Cmd:      "ls -la",
				Password: "testpass",
				Timeout:  "30s",
			},
			wantError: true,
			errorMsg:  "host parameter is required",
		},
		{
			name: "missing user",
			req: &Req{
				Host:     "example.com",
				Port:     22,
				Cmd:      "ls -la",
				Password: "testpass",
				Timeout:  "30s",
			},
			wantError: true,
			errorMsg:  "user parameter is required",
		},
		{
			name: "missing cmd",
			req: &Req{
				Host:     "example.com",
				Port:     22,
				User:     "testuser",
				Password: "testpass",
				Timeout:  "30s",
			},
			wantError: true,
			errorMsg:  "cmd parameter is required",
		},
		{
			name: "missing authentication",
			req: &Req{
				Host:    "example.com",
				Port:    22,
				User:    "testuser",
				Cmd:     "ls -la",
				Timeout: "30s",
			},
			wantError: true,
			errorMsg:  "either password or key_file must be provided for authentication",
		},
		{
			name: "invalid port",
			req: &Req{
				Host:     "example.com",
				Port:     70000,
				User:     "testuser",
				Cmd:      "ls -la",
				Password: "testpass",
				Timeout:  "30s",
			},
			wantError: true,
			errorMsg:  "invalid port number: 70000",
		},
		{
			name: "invalid timeout",
			req: &Req{
				Host:     "example.com",
				Port:     22,
				User:     "testuser",
				Cmd:      "ls -la",
				Password: "testpass",
				Timeout:  "invalid",
			},
			wantError: true,
			errorMsg:  "invalid timeout format: invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseParams(tt.req)
			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if err.Error() != tt.errorMsg {
					t.Errorf("Expected error '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidateKeyFile(t *testing.T) {
	tests := []struct {
		name      string
		keyFile   string
		wantError bool
	}{
		{
			name:      "non-existent file",
			keyFile:   "/tmp/non_existent_key",
			wantError: true,
		},
		{
			name:      "empty path",
			keyFile:   "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateKeyFile(tt.keyFile)
			if tt.wantError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.wantError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestPrepareRequestData(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]string
		expected map[string]string
	}{
		{
			name: "with environment variables",
			input: map[string]string{
				"host":      "example.com",
				"user":      "testuser",
				"env__PATH": "/usr/bin",
				"env__HOME": "/home/user",
				"cmd":       "echo $PATH",
			},
			expected: map[string]string{
				"host":      "example.com",
				"user":      "testuser",
				"env__PATH": "/usr/bin",
				"env__HOME": "/home/user",
				"cmd":       "echo $PATH",
			},
		},
		{
			name: "without environment variables",
			input: map[string]string{
				"host": "example.com",
				"user": "testuser",
				"cmd":  "ls -la",
			},
			expected: map[string]string{
				"host": "example.com",
				"user": "testuser",
				"cmd":  "ls -la",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy of input since PrepareRequestData modifies the map
			data := make(map[string]string)
			for k, v := range tt.input {
				data[k] = v
			}

			err := PrepareRequestData(data)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Verify the result
			for key, expectedValue := range tt.expected {
				if actualValue, exists := data[key]; !exists {
					t.Errorf("Expected key '%s' not found in result", key)
				} else if actualValue != expectedValue {
					t.Errorf("Expected value '%s' for key '%s', got '%s'", expectedValue, key, actualValue)
				}
			}
		})
	}
}

func TestWithBefore(t *testing.T) {
	called := false
	var capturedHost string
	var capturedPort int
	var capturedUser string
	var capturedCmd string

	beforeFunc := func(host string, port int, user string, cmd string) {
		called = true
		capturedHost = host
		capturedPort = port
		capturedUser = user
		capturedCmd = cmd
	}

	cb := &Callback{}
	option := WithBefore(beforeFunc)
	option(cb)

	if cb.before == nil {
		t.Errorf("Expected before callback to be set")
	}

	// Test the callback
	cb.before("test.com", 2222, "testuser", "test command")

	if !called {
		t.Errorf("Expected before callback to be called")
	}
	if capturedHost != "test.com" {
		t.Errorf("Expected host 'test.com', got '%s'", capturedHost)
	}
	if capturedPort != 2222 {
		t.Errorf("Expected port 2222, got %d", capturedPort)
	}
	if capturedUser != "testuser" {
		t.Errorf("Expected user 'testuser', got '%s'", capturedUser)
	}
	if capturedCmd != "test command" {
		t.Errorf("Expected cmd 'test command', got '%s'", capturedCmd)
	}
}

func TestWithAfter(t *testing.T) {
	called := false
	var capturedResult *Result

	afterFunc := func(result *Result) {
		called = true
		capturedResult = result
	}

	cb := &Callback{}
	option := WithAfter(afterFunc)
	option(cb)

	if cb.after == nil {
		t.Errorf("Expected after callback to be set")
	}

	// Test the callback
	testResult := &Result{
		Res:    Res{Code: 0, Stdout: "test output", Stderr: ""},
		RT:     time.Second,
		Status: 0,
	}
	cb.after(testResult)

	if !called {
		t.Errorf("Expected after callback to be called")
	}
	if capturedResult != testResult {
		t.Errorf("Expected captured result to match test result")
	}
}
