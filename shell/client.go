package shell

import (
	"context"
	"crypto/md5"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/linyows/probe"
)

type Req struct {
	Cmd        string            `map:"cmd" validate:"required"`
	Shell      string            `map:"shell"`
	Workdir    string            `map:"workdir"`
	Timeout    string            `map:"timeout"`
	Env        map[string]string `map:"env"`
	Background bool              `map:"background"`
	cb         *Callback
}

type Res struct {
	Code   int    `map:"code"`
	Stdout string `map:"stdout"`
	Stderr string `map:"stderr"`
	PID    int    `map:"pid"`
}

type Result struct {
	Req    Req           `map:"req"`
	Res    Res           `map:"res"`
	RT     time.Duration `map:"rt"`
	Status int           `map:"status"`
}

type shellParams struct {
	cmd     string
	workdir string
	shell   string
	timeout time.Duration
	env     map[string]string
}

type Option func(*Callback)

type Callback struct {
	before func(cmd string, shell string, workdir string)
	after  func(result *Result)
}

func NewReq() *Req {
	return &Req{
		Shell:   "/bin/sh",
		Timeout: "30s",
		Env:     make(map[string]string),
	}
}

func parseParams(req *Req) (*shellParams, error) {
	params := &shellParams{
		cmd:     req.Cmd,
		workdir: req.Workdir,
		shell:   req.Shell,
		env:     req.Env,
	}

	// Validate required parameters
	if params.cmd == "" {
		return nil, fmt.Errorf("cmd parameter is required")
	}

	// Set default shell
	if params.shell == "" {
		params.shell = "/bin/sh"
	}

	// Validate shell path for security
	if err := validateShellPath(params.shell); err != nil {
		return nil, err
	}

	// Parse timeout
	timeoutStr := req.Timeout
	if timeoutStr == "" {
		timeoutStr = "30s"
	}
	timeout, err := parseTimeout(timeoutStr)
	if err != nil {
		return nil, fmt.Errorf("invalid timeout format: %s", timeoutStr)
	}
	params.timeout = timeout

	// Validate working directory if provided
	if params.workdir != "" {
		if err := validateWorkdir(params.workdir); err != nil {
			return nil, err
		}
	}

	return params, nil
}

func validateShellPath(shell string) error {
	// Check if shell path is empty
	if shell == "" {
		return fmt.Errorf("shell path cannot be empty")
	}

	// Only allow common shell paths for security
	allowedShells := []string{
		"/bin/sh",
		"/bin/bash",
		"/bin/zsh",
		"/bin/dash",
		"/usr/bin/sh",
		"/usr/bin/bash",
		"/usr/bin/zsh",
		"/usr/bin/dash",
	}

	for _, allowed := range allowedShells {
		if shell == allowed {
			return nil
		}
	}

	return fmt.Errorf("shell path not allowed: %s", shell)
}

func validateWorkdir(workdir string) error {
	// Convert relative path to absolute path
	absPath, err := filepath.Abs(workdir)
	if err != nil {
		return fmt.Errorf("failed to resolve workdir path: %s", err)
	}

	// Check if directory exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("workdir does not exist: %s", absPath)
	}

	return nil
}

func parseTimeout(timeoutStr string) (time.Duration, error) {
	// Check if it's a plain number (treat as seconds)
	if matched, _ := regexp.MatchString(`^\d+$`, timeoutStr); matched {
		if seconds, err := strconv.Atoi(timeoutStr); err == nil {
			return time.Duration(seconds) * time.Second, nil
		}
	}

	// Parse as duration string (e.g., "30s", "5m", "1h")
	return time.ParseDuration(timeoutStr)
}

func (r *Req) Do() (*Result, error) {
	// Always create result with current request data, even if validation fails
	result := &Result{Req: *r}

	if r.Cmd == "" {
		return result, fmt.Errorf("Req.Cmd is required")
	}

	params, err := parseParams(r)
	if err != nil {
		return result, err
	}

	// callback before
	if r.cb != nil && r.cb.before != nil {
		r.cb.before(params.cmd, params.shell, params.workdir)
	}

	// Create command with appropriate context
	var cmd *exec.Cmd
	if r.Background {
		// For background execution, use context.Background() without timeout
		cmd = exec.CommandContext(context.Background(), params.shell, "-c", params.cmd)
	} else {
		// For synchronous execution, use timeout context
		ctx, cancel := context.WithTimeout(context.Background(), params.timeout)
		defer cancel()
		cmd = exec.CommandContext(ctx, params.shell, "-c", params.cmd)
	}

	// Set working directory
	if params.workdir != "" {
		absWorkdir, _ := filepath.Abs(params.workdir)
		cmd.Dir = absWorkdir
	}

	// Set environment variables
	cmd.Env = os.Environ()
	for key, value := range params.env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// For background execution, configure SysProcAttr to create new session
	if r.Background {
		if runtime.GOOS != "windows" {
			cmd.SysProcAttr = &syscall.SysProcAttr{
				Setsid: true, // Create new session (also creates new process group)
			}
		}
	}

	start := time.Now()

	// For background execution, setup log file and start process
	if r.Background {
		// Get current working directory for log file
		cwd, err := os.Getwd()
		if err != nil {
			return result, fmt.Errorf("failed to get current working directory: %w", err)
		}

		// Generate hash from command for unique log filename
		hash := fmt.Sprintf("%x", md5.Sum([]byte(params.cmd)))[:8]
		logPath := filepath.Join(cwd, fmt.Sprintf("bg-cmd-%s.log", hash))

		// Create log file
		logFile, err := os.Create(logPath)
		if err != nil {
			return result, fmt.Errorf("failed to create log file: %w", err)
		}

		// Redirect both stdout and stderr to the same log file
		cmd.Stdout = logFile
		cmd.Stderr = logFile

		// Start command
		if err := cmd.Start(); err != nil {
			_ = logFile.Close()
			_ = os.Remove(logPath)
			return result, fmt.Errorf("failed to start command: %w", err)
		}

		// Start a goroutine to close the log file when process exits
		go func() {
			_ = cmd.Wait()
			_ = logFile.Close()
		}()

		result.RT = time.Since(start)
		result.Res = Res{
			Code:   -1, // Indicate background process (not finished)
			Stdout: "",
			Stderr: "",
			PID:    cmd.Process.Pid,
		}
		result.Status = -1 // Indicate background execution
		
		// callback after
		if r.cb != nil && r.cb.after != nil {
			r.cb.after(result)
		}
		
		return result, nil
	}

	// Capture stdout and stderr for synchronous execution
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return result, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return result, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return result, fmt.Errorf("failed to start command: %w", err)
	}

	// Read stdout
	stdoutChan := make(chan []byte, 1)
	go func() {
		buf := make([]byte, 1024)
		var output []byte
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				output = append(output, buf[:n]...)
			}
			if err != nil {
				break
			}
		}
		stdoutChan <- output
	}()

	// Read stderr
	stderrChan := make(chan []byte, 1)
	go func() {
		buf := make([]byte, 1024)
		var output []byte
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				output = append(output, buf[:n]...)
			}
			if err != nil {
				break
			}
		}
		stderrChan <- output
	}()

	// Wait for command completion (synchronous execution)
	cmdErr := cmd.Wait()
	result.RT = time.Since(start)

	// Get output
	stdoutBytes := <-stdoutChan
	stderrBytes := <-stderrChan

	// Get exit code
	exitCode := 0
	if cmdErr != nil {
		if exitError, ok := cmdErr.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			// Non-exit error (e.g., timeout, process killed)
			return result, fmt.Errorf("command execution failed: %w", cmdErr)
		}
	}

	// Determine status based on exit code (0 = success, 1 = failure)
	status := 1 // default to failure
	if exitCode == 0 {
		status = 0 // success
	}

	result.Res = Res{
		Code:   exitCode,
		Stdout: string(stdoutBytes),
		Stderr: string(stderrBytes),
		PID:    cmd.Process.Pid,
	}
	result.Status = status

	// callback after
	if r.cb != nil && r.cb.after != nil {
		r.cb.after(result)
	}

	return result, nil
}

// PrepareRequestData prepares shell request data by extracting environment variables
func PrepareRequestData(data map[string]string) error {
	// Extract environment variables from env__ prefixed keys
	env := make(map[string]string)
	for key, value := range data {
		if strings.HasPrefix(key, "env__") {
			envKey := strings.TrimPrefix(key, "env__")
			env[envKey] = value
			delete(data, key)
		}
	}

	// Store env as a nested structure if any env vars were found
	if len(env) > 0 {
		for key, value := range env {
			data["env__"+key] = value
		}
	}

	return nil
}

func Execute(data map[string]any, opts ...Option) (map[string]any, error) {
	// Create a copy to avoid modifying the original data and convert to map[string]string for compatibility
	dataCopy := make(map[string]string)
	for k, v := range data {
		if str, ok := v.(string); ok {
			dataCopy[k] = str
		} else {
			dataCopy[k] = fmt.Sprintf("%v", v)
		}
	}

	// Prepare request data
	if err := PrepareRequestData(dataCopy); err != nil {
		return map[string]any{}, err
	}

	// Convert dataCopy (map[string]string) directly to map[string]any
	m := make(map[string]any)
	for k, v := range dataCopy {
		m[k] = v
	}
	m = probe.HeaderToStringValue(m)

	r := NewReq()

	cb := &Callback{}
	for _, opt := range opts {
		opt(cb)
	}
	r.cb = cb

	mapErr := probe.MapToStructByTags(m, r)

	result, err := r.Do()
	if err != nil || mapErr != nil {
		// Even on error, try to return a structured result if we have one
		if result != nil {
			mapResult, structErr := probe.StructToMapByTags(result)
			if structErr == nil {
				// Return the original error (either mapErr or err)
				if mapErr != nil {
					return mapResult, mapErr
				}
				return mapResult, err
			}
		}
		// If we can't create a structured result, return the original error
		if mapErr != nil {
			return map[string]any{}, mapErr
		}
		return map[string]any{}, err
	}

	mapResult, err := probe.StructToMapByTags(result)
	if err != nil {
		return map[string]any{}, err
	}

	// Return the result directly without flattening
	return mapResult, nil
}

func WithBefore(f func(cmd string, shell string, workdir string)) Option {
	return func(c *Callback) {
		c.before = f
	}
}

func WithAfter(f func(result *Result)) Option {
	return func(c *Callback) {
		c.after = f
	}
}
