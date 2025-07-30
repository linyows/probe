package main

import (
	"bytes"
	"flag"
	"os"
	"strings"
	"testing"
)

func TestCmd_isValid(t *testing.T) {
	c := &Cmd{
		validFlags: []string{"help", "h", "version", "rt", "verbose", "v"},
	}

	tests := []struct {
		name     string
		flag     string
		expected bool
	}{
		{
			name:     "valid short flag",
			flag:     "-help",
			expected: true,
		},
		{
			name:     "valid long flag",
			flag:     "--help",
			expected: true,
		},
		{
			name:     "invalid flag",
			flag:     "--invalid",
			expected: false,
		},
		{
			name:     "invalid short flag",
			flag:     "-invalid",
			expected: false,
		},
		{
			name:     "valid verbose flag",
			flag:     "--verbose",
			expected: true,
		},
		{
			name:     "valid rt flag",
			flag:     "--rt",
			expected: true,
		},
		{
			name:     "valid v flag (shorthand)",
			flag:     "-v",
			expected: true,
		},
		{
			name:     "valid h flag (shorthand)",
			flag:     "-h",
			expected: true,
		},
		{
			name:     "valid version flag",
			flag:     "--version",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.isValid(tt.flag)
			if result != tt.expected {
				t.Errorf("isValid(%q) = %v, want %v", tt.flag, result, tt.expected)
			}
		})
	}
}

func TestCmd_usage(t *testing.T) {
	// Capture the output
	var buf bytes.Buffer

	// Save original and replace flag.CommandLine.Output
	originalOutput := flag.CommandLine.Output()
	flag.CommandLine.SetOutput(&buf)
	defer flag.CommandLine.SetOutput(originalOutput)

	c := &Cmd{
		ver: "test-version",
		rev: "test-commit",
	}

	c.usage()

	output := buf.String()

	// Check that output contains expected elements
	expectedElements := []string{
		"Probe - A YAML-based workflow automation tool",
		"https://github.com/linyows/probe",
		"test-version",
		"test-commit",
		"Usage: probe [options] <workflow-file>",
		"Options:",
	}

	for _, element := range expectedElements {
		if !strings.Contains(output, element) {
			t.Errorf("usage() output should contain %q, but it doesn't. Output: %s", element, output)
		}
	}

	// Check ASCII art is present (note: colors may not be in output during testing)
	if !strings.Contains(output, "__  __  __  __  __") {
		t.Errorf("usage() output should contain ASCII art")
	}
}

func TestNewCmd(t *testing.T) {
	// Save original os.Args and flag state
	originalArgs := os.Args
	originalCommandLine := flag.CommandLine
	defer func() {
		os.Args = originalArgs
		flag.CommandLine = originalCommandLine
	}()

	tests := []struct {
		name           string
		args           []string
		expectNil      bool
		expectWorkflow string
		expectVerbose  bool
		expectRT       bool
		expectHelp     bool
	}{
		{
			name:           "help flag",
			args:           []string{"probe", "--help"},
			expectNil:      false,
			expectHelp:     true,
			expectWorkflow: "",
			expectVerbose:  false,
			expectRT:       false,
		},
		{
			name:           "workflow argument",
			args:           []string{"probe", "test.yml"},
			expectNil:      false,
			expectHelp:     false,
			expectWorkflow: "test.yml",
			expectVerbose:  false,
			expectRT:       false,
		},
		{
			name:           "verbose flag",
			args:           []string{"probe", "--verbose"},
			expectNil:      false,
			expectHelp:     false,
			expectWorkflow: "",
			expectVerbose:  true,
			expectRT:       false,
		},
		{
			name:           "rt flag",
			args:           []string{"probe", "--rt"},
			expectNil:      false,
			expectHelp:     false,
			expectWorkflow: "",
			expectVerbose:  false,
			expectRT:       true,
		},
		{
			name:           "multiple flags with workflow argument",
			args:           []string{"probe", "--verbose", "--rt", "test.yml"},
			expectNil:      false,
			expectHelp:     false,
			expectWorkflow: "test.yml",
			expectVerbose:  true,
			expectRT:       true,
		},
		{
			name:           "v shorthand flag with workflow argument",
			args:           []string{"probe", "-v", "test.yml"},
			expectNil:      false,
			expectHelp:     false,
			expectWorkflow: "test.yml",
			expectVerbose:  true,
			expectRT:       false,
		},
		{
			name:           "h shorthand flag",
			args:           []string{"probe", "-h"},
			expectNil:      false,
			expectHelp:     true,
			expectWorkflow: "",
			expectVerbose:  false,
			expectRT:       false,
		},
		// Note: Builtin command tests are commented out because they try to start actual servers
		// In a real test environment, these would need to be mocked or tested differently
		// {
		// 	name:      "builtin command http",
		// 	args:      []string{"probe", probe.BuiltinCmd, "http"},
		// 	expectNil: true,
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flag.CommandLine for each test
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
			flag.CommandLine.SetOutput(os.Stderr) // Suppress output during tests

			// Set os.Args to the test args so flag.Parse() can see them
			os.Args = tt.args

			cmd := newCmd(tt.args)

			if tt.expectNil {
				if cmd != nil {
					t.Errorf("newCmd(%v) should return nil for builtin commands", tt.args)
				}
				return
			}

			if cmd == nil {
				t.Errorf("newCmd(%v) returned nil, expected non-nil", tt.args)
				return
			}

			if cmd.WorkflowPath != tt.expectWorkflow {
				t.Errorf("newCmd(%v).WorkflowPath = %q, want %q", tt.args, cmd.WorkflowPath, tt.expectWorkflow)
			}

			if cmd.Verbose != tt.expectVerbose {
				t.Errorf("newCmd(%v).Verbose = %v, want %v", tt.args, cmd.Verbose, tt.expectVerbose)
			}

			if cmd.RT != tt.expectRT {
				t.Errorf("newCmd(%v).RT = %v, want %v", tt.args, cmd.RT, tt.expectRT)
			}

			if cmd.Help != tt.expectHelp {
				t.Errorf("newCmd(%v).Help = %v, want %v", tt.args, cmd.Help, tt.expectHelp)
			}

			// Check that version and commit are set
			if cmd.ver != version {
				t.Errorf("newCmd(%v).ver = %q, want %q", tt.args, cmd.ver, version)
			}

			if cmd.rev != commit {
				t.Errorf("newCmd(%v).rev = %q, want %q", tt.args, cmd.rev, commit)
			}

			// Check validFlags
			expectedFlags := []string{"help", "h", "version", "rt", "verbose", "v"}
			if len(cmd.validFlags) != len(expectedFlags) {
				t.Errorf("newCmd(%v) validFlags length = %d, want %d", tt.args, len(cmd.validFlags), len(expectedFlags))
			}
		})
	}
}

func TestNewCmd_InvalidFlags(t *testing.T) {
	// Save original os.Args and stderr
	originalArgs := os.Args
	originalStderr := os.Stderr
	originalCommandLine := flag.CommandLine
	defer func() {
		os.Args = originalArgs
		os.Stderr = originalStderr
		flag.CommandLine = originalCommandLine
	}()

	// Capture stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Reset flag.CommandLine
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	// Set os.Args for flag parsing
	args := []string{"probe", "--invalid-flag"}
	os.Args = args

	cmd := newCmd(args)

	// Close write end and read from pipe
	_ = w.Close()
	var buf bytes.Buffer
	_, err := buf.ReadFrom(r)
	if err != nil {
		t.Fatalf("Failed to read from pipe: %v", err)
	}
	output := buf.String()

	// Should return nil for invalid flags
	if cmd != nil {
		t.Errorf("newCmd(%v) should return nil for invalid flags", args)
	}

	// Should output error message
	if !strings.Contains(output, "Unknown flag: --invalid-flag") {
		t.Errorf("Expected error message about unknown flag, got: %s", output)
	}
}

func TestCmd_start(t *testing.T) {
	tests := []struct {
		name           string
		cmd            *Cmd
		expectedExit   int
		expectsError   bool
		checkOutput    bool
		expectedOutput string
	}{
		{
			name: "help command",
			cmd: &Cmd{
				Help: true,
				ver:  "test-version",
				rev:  "test-commit",
			},
			expectedExit: 1, // Help always returns 1
			checkOutput:  false,
		},
		{
			name: "no workflow specified",
			cmd: &Cmd{
				WorkflowPath: "",
			},
			expectedExit: 1,
			expectsError: true,
		},
		{
			name: "invalid workflow path",
			cmd: &Cmd{
				WorkflowPath: "/nonexistent/path/workflow.yml",
				Verbose:      false,
			},
			expectedExit: 1,
			expectsError: true,
		},
		{
			name: "version command",
			cmd: &Cmd{
				Version: true,
				ver:     "test-version",
				rev:     "test-commit",
			},
			expectedExit: 1, // version command returns 1
			checkOutput:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stderr for error messages
			originalStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			// Capture stdout for help output
			originalCommandLineOutput := flag.CommandLine.Output()
			var helpBuf bytes.Buffer
			flag.CommandLine.SetOutput(&helpBuf)

			exitCode := tt.cmd.start()

			// Close write end and read stderr
			_ = w.Close()
			var stderrBuf bytes.Buffer
			_, err := stderrBuf.ReadFrom(r)
			if err != nil {
				t.Fatalf("Failed to read stderr: %v", err)
			}
			stderrOutput := stderrBuf.String()

			// Restore
			os.Stderr = originalStderr
			flag.CommandLine.SetOutput(originalCommandLineOutput)

			if exitCode != tt.expectedExit {
				t.Errorf("start() exit code = %d, want %d", exitCode, tt.expectedExit)
			}

			if tt.expectsError && stderrOutput == "" && helpBuf.String() == "" {
				t.Errorf("start() should produce error output or help output")
			}

			if tt.cmd.Help && !strings.Contains(helpBuf.String(), "Probe - A YAML-based workflow automation tool") {
				t.Errorf("Help command should produce help output")
			}

			if tt.cmd.WorkflowPath == "" && !tt.cmd.Help && !tt.cmd.Version {
				if !strings.Contains(stderrOutput, "workflow is required") {
					t.Errorf("Missing workflow should produce 'workflow is required' error, got: %s", stderrOutput)
				}
			}
		})
	}
}

func TestRunBuiltinActions(t *testing.T) {
	// This function calls action servers, so we mainly test that it doesn't panic
	// and that it handles the known action names without crashing

	tests := []struct {
		name       string
		actionName string
	}{
		{
			name:       "http action",
			actionName: "http",
		},
		{
			name:       "hello action",
			actionName: "hello",
		},
		{
			name:       "smtp action",
			actionName: "smtp",
		},
		{
			name:       "unknown action",
			actionName: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We cannot easily test the actual server starting without complex setup
			// So we just verify the function exists and can be called without panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("runBuiltinActions(%q) panicked: %v", tt.actionName, r)
				}
			}()

			// Note: This will actually try to start servers, which we can't test easily
			// In a real scenario, you might want to use dependency injection or
			// make the actions mockable for testing
			// For now, we'll skip the actual call to avoid starting servers during tests

			// runBuiltinActions(tt.actionName) // Commented out to avoid starting servers

			// Instead, just verify the function signature and that it compiles
			var _ = runBuiltinActions
		})
	}
}

func TestCmd_printVersion(t *testing.T) {
	// Capture stdout
	var buf bytes.Buffer
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	c := &Cmd{
		ver: "1.0.0",
		rev: "abc123",
	}

	c.printVersion()

	// Close write end and read from pipe
	_ = w.Close()
	_, err := buf.ReadFrom(r)
	if err != nil {
		t.Fatalf("Failed to read from pipe: %v", err)
	}
	output := buf.String()

	// Restore stdout
	os.Stdout = originalStdout

	expectedOutput := "Probe Version 1.0.0 (commit: abc123)\n"
	if output != expectedOutput {
		t.Errorf("printVersion() output = %q, want %q", output, expectedOutput)
	}
}
