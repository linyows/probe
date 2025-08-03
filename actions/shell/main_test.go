package shell

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
)

func TestParseParams(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]string
		expected *shellParams
		wantErr  bool
	}{
		{
			name: "basic command only",
			input: map[string]string{
				"cmd": "echo hello",
			},
			expected: &shellParams{
				cmd:     "echo hello",
				shell:   "/bin/sh",
				timeout: 30 * time.Second,
				env:     map[string]string{},
			},
			wantErr: false,
		},
		{
			name: "full configuration",
			input: map[string]string{
				"cmd":       "pwd",
				"workdir":   "/tmp",
				"shell":     "/bin/bash",
				"timeout":   "5m",
				"env__VAR1": "value1",
				"env__VAR2": "value2",
			},
			expected: &shellParams{
				cmd:     "pwd",
				workdir: "/tmp",
				shell:   "/bin/bash",
				timeout: 5 * time.Minute,
				env: map[string]string{
					"VAR1": "value1",
					"VAR2": "value2",
				},
			},
			wantErr: false,
		},
		{
			name: "timeout in seconds",
			input: map[string]string{
				"cmd":     "echo test",
				"timeout": "30s",
			},
			expected: &shellParams{
				cmd:     "echo test",
				shell:   "/bin/sh",
				timeout: 30 * time.Second,
				env:     map[string]string{},
			},
			wantErr: false,
		},
		{
			name: "timeout as plain number",
			input: map[string]string{
				"cmd":     "echo test",
				"timeout": "45",
			},
			expected: &shellParams{
				cmd:     "echo test",
				shell:   "/bin/sh",
				timeout: 45 * time.Second,
				env:     map[string]string{},
			},
			wantErr: false,
		},
		{
			name: "missing cmd parameter",
			input: map[string]string{
				"shell": "/bin/bash",
			},
			expected: nil,
			wantErr:  true,
		},
		{
			name: "invalid shell path",
			input: map[string]string{
				"cmd":   "echo test",
				"shell": "/usr/bin/evil_shell",
			},
			expected: nil,
			wantErr:  true,
		},
		{
			name: "invalid timeout format",
			input: map[string]string{
				"cmd":     "echo test",
				"timeout": "invalid",
			},
			expected: nil,
			wantErr:  true,
		},
		{
			name: "relative workdir path",
			input: map[string]string{
				"cmd":     "pwd",
				"workdir": "relative/path",
			},
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseParams(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("parseParams() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("parseParams() unexpected error = %v", err)
				return
			}

			if result.cmd != tt.expected.cmd {
				t.Errorf("cmd: expected %v, got %v", tt.expected.cmd, result.cmd)
			}

			if result.shell != tt.expected.shell {
				t.Errorf("shell: expected %v, got %v", tt.expected.shell, result.shell)
			}

			if result.workdir != tt.expected.workdir {
				t.Errorf("workdir: expected %v, got %v", tt.expected.workdir, result.workdir)
			}

			if result.timeout != tt.expected.timeout {
				t.Errorf("timeout: expected %v, got %v", tt.expected.timeout, result.timeout)
			}

			if len(result.env) != len(tt.expected.env) {
				t.Errorf("env length: expected %v, got %v", len(tt.expected.env), len(result.env))
			}

			for k, v := range tt.expected.env {
				if result.env[k] != v {
					t.Errorf("env[%s]: expected %v, got %v", k, v, result.env[k])
				}
			}
		})
	}
}

func TestValidateShellPath(t *testing.T) {
	tests := []struct {
		name    string
		shell   string
		wantErr bool
	}{
		{
			name:    "valid /bin/sh",
			shell:   "/bin/sh",
			wantErr: false,
		},
		{
			name:    "valid /bin/bash",
			shell:   "/bin/bash",
			wantErr: false,
		},
		{
			name:    "valid /bin/zsh",
			shell:   "/bin/zsh",
			wantErr: false,
		},
		{
			name:    "valid /usr/bin/bash",
			shell:   "/usr/bin/bash",
			wantErr: false,
		},
		{
			name:    "invalid shell path",
			shell:   "/usr/bin/evil",
			wantErr: true,
		},
		{
			name:    "relative shell path",
			shell:   "bash",
			wantErr: true,
		},
		{
			name:    "empty shell path",
			shell:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateShellPath(tt.shell)

			if tt.wantErr {
				if err == nil {
					t.Errorf("validateShellPath() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("validateShellPath() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestValidateWorkdir(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	tests := []struct {
		name    string
		workdir string
		wantErr bool
	}{
		{
			name:    "valid absolute path (temp dir)",
			workdir: tempDir,
			wantErr: false,
		},
		{
			name:    "valid /tmp directory",
			workdir: "/tmp",
			wantErr: false,
		},
		{
			name:    "relative path",
			workdir: "relative/path",
			wantErr: true,
		},
		{
			name:    "non-existent directory",
			workdir: "/nonexistent/directory",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWorkdir(tt.workdir)

			if tt.wantErr {
				if err == nil {
					t.Errorf("validateWorkdir() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("validateWorkdir() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestParseTimeout(t *testing.T) {
	tests := []struct {
		name     string
		timeout  string
		expected time.Duration
		wantErr  bool
	}{
		{
			name:     "plain number (seconds)",
			timeout:  "30",
			expected: 30 * time.Second,
			wantErr:  false,
		},
		{
			name:     "seconds format",
			timeout:  "45s",
			expected: 45 * time.Second,
			wantErr:  false,
		},
		{
			name:     "minutes format",
			timeout:  "5m",
			expected: 5 * time.Minute,
			wantErr:  false,
		},
		{
			name:     "hours format",
			timeout:  "2h",
			expected: 2 * time.Hour,
			wantErr:  false,
		},
		{
			name:     "milliseconds format",
			timeout:  "500ms",
			expected: 500 * time.Millisecond,
			wantErr:  false,
		},
		{
			name:    "invalid format",
			timeout: "invalid",
			wantErr: true,
		},
		{
			name:    "empty string",
			timeout: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseTimeout(tt.timeout)

			if tt.wantErr {
				if err == nil {
					t.Errorf("parseTimeout() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("parseTimeout() unexpected error = %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("parseTimeout() expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestExecuteShellCommand(t *testing.T) {
	// Use a null logger for testing to avoid log output
	logger := hclog.NewNullLogger()

	tests := []struct {
		name        string
		params      *shellParams
		expectCode  string
		expectError bool
	}{
		{
			name: "simple echo command",
			params: &shellParams{
				cmd:     "echo 'hello world'",
				shell:   "/bin/sh",
				timeout: 5 * time.Second,
				env:     map[string]string{},
			},
			expectCode:  "0",
			expectError: false,
		},
		{
			name: "command with environment variable",
			params: &shellParams{
				cmd:     "echo $TEST_VAR",
				shell:   "/bin/sh",
				timeout: 5 * time.Second,
				env:     map[string]string{"TEST_VAR": "test_value"},
			},
			expectCode:  "0",
			expectError: false,
		},
		{
			name: "command that fails",
			params: &shellParams{
				cmd:     "exit 1",
				shell:   "/bin/sh",
				timeout: 5 * time.Second,
				env:     map[string]string{},
			},
			expectCode:  "1",
			expectError: false,
		},
		{
			name: "command with working directory",
			params: &shellParams{
				cmd:     "pwd",
				workdir: "/tmp",
				shell:   "/bin/sh",
				timeout: 5 * time.Second,
				env:     map[string]string{},
			},
			expectCode:  "0",
			expectError: false,
		},
		{
			name: "command that outputs to stderr",
			params: &shellParams{
				cmd:     "echo 'error message' >&2",
				shell:   "/bin/sh",
				timeout: 5 * time.Second,
				env:     map[string]string{},
			},
			expectCode:  "0",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executeShellCommand(tt.params, logger)

			if tt.expectError {
				if err == nil {
					t.Errorf("executeShellCommand() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("executeShellCommand() unexpected error = %v", err)
				return
			}

			// Check that required fields are present
			if result["req__cmd"] != tt.params.cmd {
				t.Errorf("req__cmd: expected %v, got %v", tt.params.cmd, result["req__cmd"])
			}

			if result["req__shell"] != tt.params.shell {
				t.Errorf("req__shell: expected %v, got %v", tt.params.shell, result["req__shell"])
			}

			if result["res__code"] != tt.expectCode {
				t.Errorf("res__code: expected %v, got %v", tt.expectCode, result["res__code"])
			}

			// Check that stdout and stderr fields exist
			if _, exists := result["res__stdout"]; !exists {
				t.Errorf("res__stdout field missing")
			}

			if _, exists := result["res__stderr"]; !exists {
				t.Errorf("res__stderr field missing")
			}

			// Check that rt (response time) field exists
			if _, exists := result["rt"]; !exists {
				t.Errorf("rt field missing")
			}

			// Verify specific outputs for certain tests
			switch tt.name {
			case "simple echo command":
				if !strings.Contains(result["res__stdout"], "hello world") {
					t.Errorf("stdout should contain 'hello world', got: %v", result["res__stdout"])
				}
			case "command with environment variable":
				if !strings.Contains(result["res__stdout"], "test_value") {
					t.Errorf("stdout should contain 'test_value', got: %v", result["res__stdout"])
				}
			case "command with working directory":
				if !strings.Contains(result["res__stdout"], "/tmp") {
					t.Errorf("stdout should contain '/tmp', got: %v", result["res__stdout"])
				}
			case "command that outputs to stderr":
				if !strings.Contains(result["res__stderr"], "error message") {
					t.Errorf("stderr should contain 'error message', got: %v", result["res__stderr"])
				}
			}
		})
	}
}

func TestActionRun(t *testing.T) {
	// Use a null logger for testing
	logger := hclog.NewNullLogger()
	action := &Action{log: logger}

	tests := []struct {
		name        string
		args        []string
		with        map[string]string
		expectError bool
	}{
		{
			name: "successful command",
			args: []string{},
			with: map[string]string{
				"cmd": "echo 'test'",
			},
			expectError: false,
		},
		{
			name: "missing cmd parameter",
			args: []string{},
			with: map[string]string{
				"shell": "/bin/sh",
			},
			expectError: true,
		},
		{
			name: "empty cmd parameter",
			args: []string{},
			with: map[string]string{
				"cmd": "",
			},
			expectError: true,
		},
		{
			name: "invalid shell",
			args: []string{},
			with: map[string]string{
				"cmd":   "echo test",
				"shell": "/invalid/shell",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := action.Run(tt.args, tt.with)

			if tt.expectError {
				if err == nil {
					t.Errorf("Action.Run() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Action.Run() unexpected error = %v", err)
				return
			}

			// For successful cases, verify basic structure
			if !tt.expectError {
				// Check that basic fields are present
				requiredFields := []string{"req__cmd", "req__shell", "res__code", "res__stdout", "res__stderr", "rt"}
				for _, field := range requiredFields {
					if _, exists := result[field]; !exists {
						t.Errorf("required field %s missing from result", field)
					}
				}
			}
		})
	}
}

func TestActionRunIntegration(t *testing.T) {
	// Skip this test if we're in a restricted environment
	if os.Getenv("CI") != "" {
		t.Skip("Skipping integration test in CI environment")
	}

	logger := hclog.NewNullLogger()
	action := &Action{log: logger}

	// Test a more complex scenario
	with := map[string]string{
		"cmd":       "echo 'Hello' && echo 'Error' >&2 && exit 0",
		"shell":     "/bin/bash",
		"timeout":   "10s",
		"env__VAR1": "value1",
		"env__VAR2": "value2",
	}

	result, err := action.Run([]string{}, with)
	if err != nil {
		t.Errorf("Integration test failed with error: %v", err)
		return
	}

	// Verify the result structure
	if result["res__code"] != "0" {
		t.Errorf("Expected exit code 0, got %v", result["res__code"])
	}

	if !strings.Contains(result["res__stdout"], "Hello") {
		t.Errorf("Expected stdout to contain 'Hello', got: %v", result["res__stdout"])
	}

	if !strings.Contains(result["res__stderr"], "Error") {
		t.Errorf("Expected stderr to contain 'Error', got: %v", result["res__stderr"])
	}

	// Check environment variables are set in request
	if result["req__env__VAR1"] != "value1" {
		t.Errorf("Expected env VAR1 to be 'value1', got: %v", result["req__env__VAR1"])
	}

	if result["req__env__VAR2"] != "value2" {
		t.Errorf("Expected env VAR2 to be 'value2', got: %v", result["req__env__VAR2"])
	}
}
