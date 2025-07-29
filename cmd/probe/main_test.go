package main

import (
	"flag"
	"os"
	"testing"
)

// Test that main function doesn't panic and handles basic cases
func TestMain(t *testing.T) {
	// We can't easily test main() directly since it calls os.Exit()
	// But we can test the components that main() uses

	// Test that version and commit variables are accessible
	if version == "" {
		version = "test-version" // Set for testing
	}
	if commit == "" {
		commit = "test-commit" // Set for testing
	}

	// Test newCmd with different argument scenarios
	tests := []struct {
		name      string
		args      []string
		expectNil bool
	}{
		{
			name:      "help argument",
			args:      []string{"probe", "--help"},
			expectNil: false,
		},
		{
			name:      "no arguments",
			args:      []string{"probe"},
			expectNil: false,
		},
		// Note: Builtin command test commented out because it tries to start actual servers
		// {
		// 	name: "builtin command",
		// 	args: []string{"probe", "action", "hello"},
		// 	expectNil: true,
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore original flag state
			originalCommandLine := flag.CommandLine
			originalArgs := os.Args
			defer func() {
				flag.CommandLine = originalCommandLine
				os.Args = originalArgs
			}()

			// Reset flag.CommandLine for each test
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
			flag.CommandLine.SetOutput(os.Stderr)
			os.Args = tt.args

			cmd := newCmd(tt.args)

			if tt.expectNil && cmd != nil {
				t.Errorf("newCmd(%v) should return nil", tt.args)
			}

			if !tt.expectNil && cmd == nil {
				t.Errorf("newCmd(%v) should not return nil", tt.args)
			}
		})
	}
}

// Test that the main function's logic flow works correctly
func TestMainLogic(t *testing.T) {
	// Save original os.Args
	originalArgs := os.Args
	defer func() {
		os.Args = originalArgs
	}()

	// Test help case
	t.Run("help case", func(t *testing.T) {
		// Save and restore original flag state
		originalCommandLine := flag.CommandLine
		originalArgs := os.Args
		defer func() {
			flag.CommandLine = originalCommandLine
			os.Args = originalArgs
		}()

		// Reset flag.CommandLine for each test
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
		flag.CommandLine.SetOutput(os.Stderr)

		os.Args = []string{"probe", "--help"}
		cmd := newCmd(os.Args)

		if cmd == nil {
			t.Fatal("newCmd should return a command for help")
		}

		// Test that start() can be called (it will return 1 for help)
		exitCode := cmd.start()
		if exitCode != 1 {
			t.Errorf("Help command should return exit code 1, got %d", exitCode)
		}
	})

	// Test no workflow case
	t.Run("no workflow case", func(t *testing.T) {
		// Save and restore original flag state
		originalCommandLine := flag.CommandLine
		originalArgs := os.Args
		defer func() {
			flag.CommandLine = originalCommandLine
			os.Args = originalArgs
		}()

		// Reset flag.CommandLine for each test
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
		flag.CommandLine.SetOutput(os.Stderr)

		os.Args = []string{"probe"}
		cmd := newCmd(os.Args)

		if cmd == nil {
			t.Fatal("newCmd should return a command")
		}

		// Test that start() returns 1 for missing workflow
		exitCode := cmd.start()
		if exitCode != 1 {
			t.Errorf("Missing workflow should return exit code 1, got %d", exitCode)
		}
	})

	// Note: Builtin command test commented out because it tries to start actual servers
	// t.Run("builtin command case", func(t *testing.T) {
	// 	os.Args = []string{"probe", "action", "hello"}
	// 	cmd := newCmd(os.Args)
	//
	// 	// Should return nil for builtin commands
	// 	if cmd != nil {
	// 		t.Error("Builtin commands should return nil from newCmd")
	// 	}
	// })
}

// Test global variables
func TestGlobalVariables(t *testing.T) {
	// Test that version and commit are defined
	if version != "dev" && version == "" {
		t.Error("version should be set to 'dev' or another value")
	}

	if commit != "unknown" && commit == "" {
		t.Error("commit should be set to 'unknown' or another value")
	}
}

// Test that we can simulate the main function logic without calling os.Exit
func TestMainSimulation(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectExitCall bool
	}{
		{
			name:           "valid help command",
			args:           []string{"probe", "--help"},
			expectExitCall: true,
		},
		// Note: Builtin command test commented out
		// {
		// 	name:     "builtin command",
		// 	args:     []string{"probe", "action", "hello"},
		// 	expectExitCall: false,
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore original flag state
			originalCommandLine := flag.CommandLine
			originalArgs := os.Args
			defer func() {
				flag.CommandLine = originalCommandLine
				os.Args = originalArgs
			}()

			// Reset flag.CommandLine for each test
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
			flag.CommandLine.SetOutput(os.Stderr)
			os.Args = tt.args

			cmd := newCmd(tt.args)

			if tt.expectExitCall {
				if cmd == nil {
					t.Error("Expected command to be created, but got nil")
				} else {
					// If cmd is not nil, main would call os.Exit(cmd.start())
					exitCode := cmd.start()
					if exitCode < 0 {
						t.Errorf("start() should return non-negative exit code, got %d", exitCode)
					}
				}
			} else {
				// If cmd is nil, main would not call os.Exit
				if cmd != nil {
					t.Error("Expected nil command, but got non-nil")
				}
			}
		})
	}
}
