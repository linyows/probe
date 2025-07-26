package probe

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/linyows/probe/testserver"
)

func TestEndToEndExitCodes(t *testing.T) {
	// Start local test server with dynamic port
	server := testserver.NewTestServer(0) // 0 means use any available port
	if err := server.Start(); err != nil {
		t.Fatalf("failed to start test server: %v", err)
	}
	defer func() {
		if err := server.Stop(); err != nil {
			t.Logf("Error stopping server: %v", err)
		}
	}()
	
	// Give server time to start
	time.Sleep(100 * time.Millisecond)
	
	serverURL := server.URL()
	t.Logf("Test server started at: %s", serverURL)

	// Get server port for creating test workflow files
	port := strings.Split(serverURL, ":")[2]

	tests := []struct {
		name         string
		templatePath string
		expectedCode int
	}{
		{
			name:         "success workflow returns exit code 0",
			templatePath: "testdata/success-localhost.yml",
			expectedCode: 0,
		},
		{
			name:         "failure workflow returns exit code 1",
			templatePath: "testdata/failure-localhost.yml",
			expectedCode: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary workflow file with correct port
			templateContent, err := os.ReadFile(tt.templatePath)
			if err != nil {
				t.Fatalf("failed to read template file: %v", err)
			}
			
			workflowContent := strings.ReplaceAll(string(templateContent), "PORT_PLACEHOLDER", port)
			
			// Create temporary file
			tmpFile, err := os.CreateTemp("", "test-workflow-*.yml")
			if err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())
			
			if _, err := tmpFile.WriteString(workflowContent); err != nil {
				t.Fatalf("failed to write temp file: %v", err)
			}
			tmpFile.Close()

			cmd := exec.Command("go", "run", "./cmd/probe", "--workflow", tmpFile.Name())
			output, err := cmd.CombinedOutput()
			t.Logf("Command output: %s", string(output))

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
