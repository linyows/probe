package main

import (
	"fmt"
	"strings"
	"testing"
)

func TestCmd_isValid(t *testing.T) {
	c := newBufferCmd()

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
	c := newBufferCmd()
	c.ver = "test-version"
	c.rev = "test-commit"
	c.usage()
	output := fmt.Sprintf("%s", c.errWriter)

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

func TestCmd_start(t *testing.T) {
	help := " __  __  __  __  __\n|  ||  ||  ||  || _|\n|  ||  /| |||  /|  |\n| | |  \\| |||  \\| _|\n|_| |_\\_|__||__||__|\n\nProbe - A YAML-based workflow automation tool.\nhttps://github.com/linyows/probe (ver: dev, rev: unknown)\n\nUsage: probe [options] <workflow-file>\n\nArguments:\n  workflow-file    Path to YAML workflow file(s). Multiple files can be \n                   specified with comma-separated paths (e.g., \"base.yml,override.yml\")\n                   to merge configurations.\n\nOptions:\n  -h, --help       Show command usage\n      --version    Show version information\n      --rt         Show response time\n  -v, --verbose    Show verbose log\n      --dag-ascii  Show job dependency graph as ASCII art\n"

	tests := []struct {
		name           string
		args           []string
		expectCode     int
		expectWorkflow string
		expectVerbose  bool
		expectRT       bool
		expectHelp     bool
		expectOutput   string
	}{
		{
			name:           "help flag",
			args:           []string{"probe", "--help"},
			expectCode:     1,
			expectHelp:     true,
			expectWorkflow: "",
			expectVerbose:  false,
			expectRT:       false,
			expectOutput:   help,
		},
		{
			name:           "version command",
			args:           []string{"probe", "--version"},
			expectCode:     0,
			expectHelp:     false,
			expectWorkflow: "",
			expectVerbose:  false,
			expectRT:       false,
			expectOutput:   "",
		},
		{
			name:           "no workflow specified",
			args:           []string{"probe"},
			expectCode:     1,
			expectHelp:     false,
			expectWorkflow: "",
			expectVerbose:  false,
			expectRT:       false,
			expectOutput:   "[ERROR] workflow is required\n",
		},
		{
			name:           "workflow argument",
			args:           []string{"probe", "test.yml"},
			expectCode:     0,
			expectHelp:     false,
			expectWorkflow: "test.yml",
			expectVerbose:  false,
			expectRT:       false,
			expectOutput:   "",
		},
		{
			name:           "verbose flag without workflow",
			args:           []string{"probe", "--verbose"},
			expectCode:     1,
			expectHelp:     false,
			expectWorkflow: "",
			expectVerbose:  true,
			expectRT:       false,
			expectOutput:   "[ERROR] workflow is required\n",
		},
		{
			name:           "rt flag without workflow",
			args:           []string{"probe", "--rt"},
			expectCode:     1,
			expectHelp:     false,
			expectWorkflow: "",
			expectVerbose:  false,
			expectRT:       true,
			expectOutput:   "[ERROR] workflow is required\n",
		},
		{
			name:           "multiple flags with workflow argument",
			args:           []string{"probe", "--verbose", "--rt", "test.yml"},
			expectCode:     0,
			expectHelp:     false,
			expectWorkflow: "test.yml",
			expectVerbose:  true,
			expectRT:       true,
			expectOutput:   "",
		},
		{
			name:           "v shorthand flag with workflow argument",
			args:           []string{"probe", "-v", "test.yml"},
			expectCode:     0,
			expectHelp:     false,
			expectWorkflow: "test.yml",
			expectVerbose:  true,
			expectRT:       false,
			expectOutput:   "",
		},
		{
			name:           "h shorthand flag",
			args:           []string{"probe", "-h"},
			expectCode:     1,
			expectHelp:     true,
			expectWorkflow: "",
			expectVerbose:  false,
			expectRT:       false,
			expectOutput:   help,
		},
		{
			name:           "options after argument",
			args:           []string{"probe", "test.yml", "--verbose"},
			expectCode:     0,
			expectHelp:     false,
			expectWorkflow: "test.yml",
			expectVerbose:  true,
			expectRT:       false,
			expectOutput:   "",
		},
		{
			name:           "options after argument with multiple flags",
			args:           []string{"probe", "test.yml", "--verbose", "--rt"},
			expectCode:     0,
			expectHelp:     false,
			expectWorkflow: "test.yml",
			expectVerbose:  true,
			expectRT:       true,
			expectOutput:   "",
		},
		{
			name:           "mixed options before and after argument",
			args:           []string{"probe", "--verbose", "test.yml", "--rt"},
			expectCode:     0,
			expectHelp:     false,
			expectWorkflow: "test.yml",
			expectVerbose:  true,
			expectRT:       true,
			expectOutput:   "",
		},
		{
			name:           "shorthand option after argument",
			args:           []string{"probe", "test.yml", "-v"},
			expectCode:     0,
			expectHelp:     false,
			expectWorkflow: "test.yml",
			expectVerbose:  true,
			expectRT:       false,
			expectOutput:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newBufferCmd()
			code := c.start(tt.args)
			output := fmt.Sprintf("%s", c.errWriter)

			if code != tt.expectCode {
				t.Errorf("start(%v) = %d, want %d", tt.args, code, tt.expectCode)
			}

			if c.WorkflowPath != tt.expectWorkflow {
				t.Errorf("start(%v).WorkflowPath = %q, want %q", tt.args, c.WorkflowPath, tt.expectWorkflow)
			}

			if c.Verbose != tt.expectVerbose {
				t.Errorf("start(%v).Verbose = %v, want %v", tt.args, c.Verbose, tt.expectVerbose)
			}

			if c.RT != tt.expectRT {
				t.Errorf("start(%v).RT = %v, want %v", tt.args, c.RT, tt.expectRT)
			}

			if c.Help != tt.expectHelp {
				t.Errorf("start(%v).Help = %v, want %v", tt.args, c.Help, tt.expectHelp)
			}

			// Check that version and commit are set
			if c.ver != version {
				t.Errorf("start(%v).ver = %q, want %q", tt.args, c.ver, version)
			}

			if c.rev != commit {
				t.Errorf("start(%v).rev = %q, want %q", tt.args, c.rev, commit)
			}

			if output != tt.expectOutput {
				t.Errorf("start(%v) output to %q, want %q", tt.args, output, tt.expectOutput)
			}

			// Check validFlags
			expectedFlags := []string{"help", "h", "version", "rt", "verbose", "v", "dag-ascii"}
			if len(c.validFlags) != len(expectedFlags) {
				t.Errorf("start(%v) validFlags length = %d, want %d", tt.args, len(c.validFlags), len(expectedFlags))
			}
		})
	}
}

func TestCmd_InvalidFlags(t *testing.T) {
	args := []string{"probe", "--invalid-flag"}
	cmd := newBufferCmd()
	n := cmd.start(args)
	output := fmt.Sprintf("%s", cmd.errWriter)
	expected := strings.TrimLeft(`
[ERROR] unknown flag: --invalid-flag
try --help to know more
`, "\n")

	if n != 1 {
		t.Errorf("Cmd.start(%v) should return 1 for invalid flags", args)
	}

	if expected != output {
		t.Errorf("Expected error message about unknown flag, got: %s", output)
	}
}

func TestCmd_runBuiltinActions(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{name: "hello", expected: ""},
		{name: "http", expected: ""},
		{name: "smtp", expected: ""},
		{name: "db", expected: ""},
		{name: "shell", expected: ""},
		{name: "browser", expected: ""},
		{name: "embedded", expected: ""},
		{name: "unknown", expected: "[ERROR] not supported plugin: unknown\n"},
	}

	cmd := newBufferCmd()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd.runBuiltinActions(tt.name)
			got := fmt.Sprintf("%s", cmd.errWriter)
			if got != tt.expected {
				t.Errorf("runBuiltinActions got: %s, expected: %s", got, tt.expected)
			}
		})
	}
}

func TestCmd_printVersion(t *testing.T) {
	c := newBufferCmd()
	c.ver = "1.0.0"
	c.rev = "abc123"
	c.printVersion()
	got := fmt.Sprintf("%s", c.outWriter)
	expected := "Probe Version 1.0.0 (commit: abc123)\n"

	if got != expected {
		t.Errorf("printVersion() output = %q, want %q", got, expected)
	}
}
