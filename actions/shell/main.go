package shell

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/linyows/probe"
)

type Action struct {
	log hclog.Logger
}

func (a *Action) Run(args []string, with map[string]string) (map[string]string, error) {
	truncateLength := probe.MaxLogStringLength
	truncatedParams := probe.TruncateMapStringString(with, truncateLength)
	a.log.Debug("received shell request parameters", "params", truncatedParams)

	// Validate required parameters
	cmd, exists := with["cmd"]
	if !exists || cmd == "" {
		return map[string]string{}, fmt.Errorf("cmd parameter is required")
	}

	// Parse parameters
	params, err := parseParams(with)
	if err != nil {
		a.log.Error("failed to parse parameters", "error", err)
		return map[string]string{}, err
	}

	// Execute command
	result, err := executeShellCommand(params, a.log)
	if err != nil {
		a.log.Error("shell command execution failed", "error", err)
		// Return result even on error for debugging
		return result, err
	}

	truncatedResult := probe.TruncateMapStringString(result, truncateLength)
	a.log.Debug("shell command completed", "result", truncatedResult)

	return result, nil
}

type shellParams struct {
	cmd     string
	workdir string
	shell   string
	timeout time.Duration
	env     map[string]string
}

func parseParams(with map[string]string) (*shellParams, error) {
	params := &shellParams{
		cmd:     with["cmd"],
		workdir: with["workdir"],
		shell:   with["shell"],
		env:     make(map[string]string),
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
	if timeoutStr := with["timeout"]; timeoutStr != "" {
		timeout, err := parseTimeout(timeoutStr)
		if err != nil {
			return nil, fmt.Errorf("invalid timeout format: %s", timeoutStr)
		}
		params.timeout = timeout
	} else {
		params.timeout = 30 * time.Second // Default timeout
	}

	// Parse environment variables
	for key, value := range with {
		if strings.HasPrefix(key, "env__") {
			envKey := strings.TrimPrefix(key, "env__")
			params.env[envKey] = value
		}
	}

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
	// Check if path is absolute
	if !filepath.IsAbs(workdir) {
		return fmt.Errorf("workdir must be an absolute path: %s", workdir)
	}

	// Check if directory exists
	if _, err := os.Stat(workdir); os.IsNotExist(err) {
		return fmt.Errorf("workdir does not exist: %s", workdir)
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

type ShellReq struct {
	Cmd     string            `map:"cmd"`
	Shell   string            `map:"shell"`
	Workdir string            `map:"workdir"`
	Timeout string            `map:"timeout"`
	Env     map[string]string `map:"env"`
}

type ShellRes struct {
	Code   int    `map:"code"`
	Stdout string `map:"stdout"`
	Stderr string `map:"stderr"`
}

type ShellResult struct {
	Req ShellReq      `map:"req"`
	Res ShellRes      `map:"res"`
	RT  time.Duration `map:"rt"`
}

func executeShellCommand(params *shellParams, log hclog.Logger) (map[string]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), params.timeout)
	defer cancel()

	// Create command
	cmd := exec.CommandContext(ctx, params.shell, "-c", params.cmd)

	// Set working directory
	if params.workdir != "" {
		cmd.Dir = params.workdir
	}

	// Set environment variables
	cmd.Env = os.Environ()
	for key, value := range params.env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	log.Debug("executing shell command", "cmd", params.cmd, "shell", params.shell, "workdir", params.workdir)

	start := time.Now()

	// Capture stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return map[string]string{}, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return map[string]string{}, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return map[string]string{}, fmt.Errorf("failed to start command: %w", err)
	}

	// Read output
	stdoutBytes := make([]byte, 0)
	stderrBytes := make([]byte, 0)

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

	// Wait for command completion
	cmdErr := cmd.Wait()
	duration := time.Since(start)

	// Get output
	stdoutBytes = <-stdoutChan
	stderrBytes = <-stderrChan

	// Get exit code
	exitCode := 0
	if cmdErr != nil {
		if exitError, ok := cmdErr.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			// Non-exit error (e.g., timeout, process killed)
			return map[string]string{}, fmt.Errorf("command execution failed: %w", cmdErr)
		}
	}

	// Create result structure similar to HTTP action
	result := &ShellResult{
		Req: ShellReq{
			Cmd:     params.cmd,
			Shell:   params.shell,
			Workdir: params.workdir,
			Timeout: params.timeout.String(),
			Env:     params.env,
		},
		Res: ShellRes{
			Code:   exitCode,
			Stdout: string(stdoutBytes),
			Stderr: string(stderrBytes),
		},
		RT: duration,
	}

	// Convert to map[string]string using probe's mapping function
	mapResult, err := probe.StructToMapByTags(result)
	if err != nil {
		return map[string]string{}, fmt.Errorf("failed to convert result to map: %w", err)
	}

	// Flatten the result like HTTP action does
	return probe.FlattenInterface(mapResult), nil
}

func Serve() {
	log := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Debug,
		Output:     os.Stderr,
		JSONFormat: true,
	})

	pl := &probe.ActionsPlugin{
		Impl: &Action{log: log},
	}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: probe.Handshake,
		Plugins:         map[string]plugin.Plugin{"actions": pl},
		GRPCServer:      plugin.DefaultGRPCServer,
	})
}
