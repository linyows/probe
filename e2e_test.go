package probe

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
)

func TestEndToEndExitCodes(t *testing.T) {
	tests := []struct {
		name         string
		workflowPath string
		expectedCode int
	}{
		{
			name:         "success workflow with hello action returns exit code 0",
			workflowPath: "testdata/success-hello.yml",
			expectedCode: 0,
		},
		{
			name:         "failure workflow with hello action returns exit code 1",
			workflowPath: "testdata/failure-hello.yml",
			expectedCode: 1,
		},
		{
			name:         "success workflow with embedded action returns exit code 0",
			workflowPath: "testdata/embedded-success-workflow.yml",
			expectedCode: 0,
		},
		{
			name:         "failure workflow with embedded action returns exit code 1",
			workflowPath: "testdata/embedded-failure-workflow.yml",
			expectedCode: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("go", "run", "./cmd/probe", tt.workflowPath)
			_, err := cmd.CombinedOutput()

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

func TestEndToEndBackgroundStoppedAfterWorkflow(t *testing.T) {
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "pid")
	logFile := filepath.Join(dir, "log")
	workflow := filepath.Join(dir, "workflow.yml")
	yml := fmt.Sprintf(`name: background
jobs:
- name: server
  steps:
  - name: start a command that never ends
    id: server
    uses: shell
    with:
      cmd: "echo $$ > %s; exec sleep 300"
      background: true
    outputs:
      log: res.log
  - name: record the log path
    uses: shell
    with:
      cmd: "echo '{{outputs.server.log}}' > %s; sleep 0.2"
    test: res.code == 0
`, pidFile, logFile)
	if err := os.WriteFile(workflow, []byte(yml), 0o600); err != nil {
		t.Fatal(err)
	}

	out, err := exec.Command("go", "run", "./cmd/probe", workflow).CombinedOutput()
	if err != nil {
		t.Fatalf("probe failed: %v\n%s", err, out)
	}

	pidData, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("background command did not start: %v\n%s", err, out)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(pidData)))
	if err != nil {
		t.Fatalf("pid file = %q: %v", pidData, err)
	}
	t.Cleanup(func() { _ = syscall.Kill(-pid, syscall.SIGKILL) })
	if syscall.Kill(pid, 0) == nil {
		t.Errorf("background process %d is still running after the workflow", pid)
	}

	logData, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("log path was not recorded: %v\n%s", err, out)
	}
	log := strings.TrimSpace(string(logData))
	if !strings.Contains(log, "probe-shell-action.") {
		t.Fatalf("recorded log path = %q", log)
	}
	if _, err := os.Stat(log); !os.IsNotExist(err) {
		_ = os.Remove(log)
		t.Errorf("log %s was left behind: %v", log, err)
	}
}
