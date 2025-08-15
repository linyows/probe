package shell

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestNewReq(t *testing.T) {
	got := NewReq()

	expected := &Req{
		Cmd:     "",
		Shell:   "/bin/sh",
		Workdir: "",
		Timeout: "30s",
		Env:     map[string]string{},
	}

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("\nExpected:\n%#v\nGot:\n%#v", expected, got)
	}
}

func TestValidateShellPath(t *testing.T) {
	tests := []struct {
		name    string
		shell   string
		wantErr bool
	}{
		{"valid /bin/sh", "/bin/sh", false},
		{"valid /bin/bash", "/bin/bash", false},
		{"valid /bin/zsh", "/bin/zsh", false},
		{"valid /usr/bin/bash", "/usr/bin/bash", false},
		{"invalid shell", "/usr/local/bin/fish", true},
		{"empty shell", "", true},
		{"relative path", "bash", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateShellPath(tt.shell)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateShellPath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseTimeout(t *testing.T) {
	tests := []struct {
		name        string
		timeoutStr  string
		expected    time.Duration
		expectError bool
	}{
		{"plain seconds", "30", 30 * time.Second, false},
		{"duration format", "30s", 30 * time.Second, false},
		{"minutes", "5m", 5 * time.Minute, false},
		{"hours", "1h", 1 * time.Hour, false},
		{"invalid format", "invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTimeout(tt.timeoutStr)
			if tt.expectError {
				if err == nil {
					t.Errorf("parseTimeout() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("parseTimeout() unexpected error: %v", err)
				return
			}

			if got != tt.expected {
				t.Errorf("parseTimeout() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDo(t *testing.T) {
	tests := []struct {
		name        string
		req         *Req
		expectError bool
		checkOutput bool
	}{
		{
			name: "simple echo command",
			req: &Req{
				Cmd:     "echo 'Hello World'",
				Shell:   "/bin/sh",
				Timeout: "5s",
			},
			expectError: false,
			checkOutput: true,
		},
		{
			name: "command with exit code 1",
			req: &Req{
				Cmd:     "exit 1",
				Shell:   "/bin/sh",
				Timeout: "5s",
			},
			expectError: false, // Exit code 1 should not cause error, just status = 1
			checkOutput: false,
		},
		{
			name: "empty command",
			req: &Req{
				Cmd:   "",
				Shell: "/bin/sh",
			},
			expectError: true,
			checkOutput: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Track if callbacks were called
			beforeCalled := false
			afterCalled := false

			// Set up callbacks for testing
			tt.req.cb = &Callback{
				before: func(cmd string, shell string, workdir string) {
					beforeCalled = true
				},
				after: func(result *Result) {
					afterCalled = true
				},
			}

			result, err := tt.req.Do()

			if tt.expectError {
				if err == nil {
					t.Errorf("Do() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Do() unexpected error: %v", err)
				return
			}

			// Verify callbacks were called
			if !beforeCalled {
				t.Error("before callback was not called")
			}
			if !afterCalled {
				t.Error("after callback was not called")
			}

			// Check basic result structure
			if result == nil {
				t.Error("Do() returned nil result")
				return
			}

			// Check that RT field is populated
			if result.RT <= 0 {
				t.Errorf("RT should be greater than 0, got: %v", result.RT)
			}

			// Check request fields are preserved
			if result.Req.Cmd != tt.req.Cmd {
				t.Errorf("Req.Cmd = %v, want %v", result.Req.Cmd, tt.req.Cmd)
			}

			if tt.checkOutput {
				// For echo command, stdout should contain "Hello World"
				if !strings.Contains(result.Res.Stdout, "Hello World") {
					t.Errorf("Expected stdout to contain 'Hello World', got: %s", result.Res.Stdout)
				}

				// Successful command should have status 0
				if result.Status != 0 {
					t.Errorf("Expected status 0 for successful command, got: %d", result.Status)
				}

				// Exit code should be 0
				if result.Res.Code != 0 {
					t.Errorf("Expected exit code 0, got: %d", result.Res.Code)
				}
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
			name: "extract environment variables",
			input: map[string]string{
				"cmd":           "echo $TEST_VAR",
				"env__TEST_VAR": "hello",
				"env__PATH":     "/usr/bin",
				"shell":         "/bin/bash",
			},
			expected: map[string]string{
				"cmd":           "echo $TEST_VAR",
				"shell":         "/bin/bash",
				"env__TEST_VAR": "hello",
				"env__PATH":     "/usr/bin",
			},
		},
		{
			name: "no environment variables",
			input: map[string]string{
				"cmd":   "echo hello",
				"shell": "/bin/sh",
			},
			expected: map[string]string{
				"cmd":   "echo hello",
				"shell": "/bin/sh",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy of input to avoid modifying the test case
			data := make(map[string]string)
			for k, v := range tt.input {
				data[k] = v
			}

			err := PrepareRequestData(data)
			if err != nil {
				t.Errorf("PrepareRequestData() error = %v", err)
				return
			}

			if !reflect.DeepEqual(data, tt.expected) {
				t.Errorf("PrepareRequestData() = %v, want %v", data, tt.expected)
			}
		})
	}
}

func TestExecute(t *testing.T) {
	tests := []struct {
		name        string
		data        map[string]string
		expectError bool
		checkStatus bool
	}{
		{
			name: "simple command execution",
			data: map[string]string{
				"cmd":   "echo 'test output'",
				"shell": "/bin/sh",
			},
			expectError: false,
			checkStatus: true,
		},
		{
			name: "command with environment variable",
			data: map[string]string{
				"cmd":           "echo $TEST_VAR",
				"shell":         "/bin/sh",
				"env__TEST_VAR": "hello_world",
			},
			expectError: false,
			checkStatus: true,
		},
		{
			name: "missing required cmd parameter",
			data: map[string]string{
				"shell": "/bin/sh",
			},
			expectError: true,
			checkStatus: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Track if callbacks were called
			beforeCalled := false
			afterCalled := false

			before := WithBefore(func(cmd string, shell string, workdir string) {
				beforeCalled = true
			})
			after := WithAfter(func(result *Result) {
				afterCalled = true
			})

			result, err := Execute(tt.data, before, after)

			if tt.expectError {
				if err == nil {
					t.Errorf("Execute() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Execute() unexpected error: %v", err)
				return
			}

			// Verify callbacks were called
			if !beforeCalled {
				t.Error("before callback was not called")
			}
			if !afterCalled {
				t.Error("after callback was not called")
			}

			// Check that result is flattened map
			if result == nil {
				t.Error("Execute() returned nil result")
				return
			}

			if tt.checkStatus {
				// Check that basic fields exist in flattened result
				if _, exists := result["req__cmd"]; !exists {
					t.Error("Expected 'req__cmd' field in result")
				}
				if _, exists := result["res__code"]; !exists {
					t.Error("Expected 'res__code' field in result")
				}
				if _, exists := result["status"]; !exists {
					t.Error("Expected 'status' field in result")
				}
			}
		})
	}
}

func TestWithBefore(t *testing.T) {
	called := false
	var capturedCmd, capturedShell, capturedWorkdir string

	option := WithBefore(func(cmd string, shell string, workdir string) {
		called = true
		capturedCmd = cmd
		capturedShell = shell
		capturedWorkdir = workdir
	})

	cb := &Callback{}
	option(cb)

	if cb.before == nil {
		t.Error("WithBefore() did not set before callback")
		return
	}

	// Test the callback
	cb.before("test-cmd", "/bin/bash", "/tmp")

	if !called {
		t.Error("before callback was not called")
	}
	if capturedCmd != "test-cmd" {
		t.Errorf("Expected cmd 'test-cmd', got '%s'", capturedCmd)
	}
	if capturedShell != "/bin/bash" {
		t.Errorf("Expected shell '/bin/bash', got '%s'", capturedShell)
	}
	if capturedWorkdir != "/tmp" {
		t.Errorf("Expected workdir '/tmp', got '%s'", capturedWorkdir)
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
		Status: 0,
		Res: Res{
			Code:   0,
			Stdout: "test output",
		},
	}
	cb.after(testResult)

	if !called {
		t.Error("after callback was not called")
	}
	if capturedResult != testResult {
		t.Error("after callback did not receive correct result")
	}
}
