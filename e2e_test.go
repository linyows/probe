package probe

import (
	"os"
	"os/exec"
	"testing"
)

func TestEndToEndExitCodes(t *testing.T) {
	tests := []struct {
		name         string
		workflowPath string
		expectedCode int
	}{
		{
			name:         "success workflow returns exit code 0",
			workflowPath: "testdata/success.yml",
			expectedCode: 0,
		},
		{
			name:         "failure workflow returns exit code 1",
			workflowPath: "testdata/failure.yml",
			expectedCode: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("go", "run", "./cmd/probe", "--workflow", tt.workflowPath)
			cmd.Env = os.Environ()

			err := cmd.Run()

			var exitCode int
			if err != nil {
				if exitError, ok := err.(*exec.ExitError); ok {
					exitCode = exitError.ExitCode()
				} else {
					t.Fatalf("unexpected error type: %v", err)
				}
			} else {
				exitCode = 0
			}

			if exitCode != tt.expectedCode {
				t.Errorf("expected exit code %d, got %d", tt.expectedCode, exitCode)
			}
		})
	}
}
