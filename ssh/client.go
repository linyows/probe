package ssh

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"

	"github.com/linyows/probe"
)

// NOTE: SSH config file support is intentionally not implemented to maintain
// portability and reproducibility. All SSH connection parameters must be
// explicitly defined in the workflow configuration to ensure that workflows
// are self-contained and can be reliably reproduced across different environments.

type Req struct {
	Host            string            `map:"host" validate:"required"`
	Port            int               `map:"port"`
	User            string            `map:"user" validate:"required"`
	Cmd             string            `map:"cmd" validate:"required"`
	Password        string            `map:"password"`
	KeyFile         string            `map:"key_file"`
	KeyPassphrase   string            `map:"key_passphrase"`
	Timeout         string            `map:"timeout"`
	Workdir         string            `map:"workdir"`
	Env             map[string]string `map:"env"`
	StrictHostCheck bool              `map:"strict_host_check"`
	KnownHosts      string            `map:"known_hosts"`
	cb              *Callback
}

type Res struct {
	Code   int    `map:"code"`
	Stdout string `map:"stdout"`
	Stderr string `map:"stderr"`
}

type Result struct {
	Req    Req           `map:"req"`
	Res    Res           `map:"res"`
	RT     time.Duration `map:"rt"`
	Status int           `map:"status"`
}

type sshParams struct {
	host            string
	port            int
	user            string
	cmd             string
	password        string
	keyFile         string
	keyPassphrase   string
	timeout         time.Duration
	workdir         string
	env             map[string]string
	strictHostCheck bool
	knownHosts      string
}

type Option func(*Callback)

type Callback struct {
	before func(host string, port int, user string, cmd string)
	after  func(result *Result)
}

func NewReq() *Req {
	return &Req{
		Port:            22,
		Timeout:         "30s",
		Env:             make(map[string]string),
		StrictHostCheck: true,
	}
}

func parseParams(req *Req) (*sshParams, error) {
	params := &sshParams{
		host:            req.Host,
		port:            req.Port,
		user:            req.User,
		cmd:             req.Cmd,
		password:        req.Password,
		keyFile:         req.KeyFile,
		keyPassphrase:   req.KeyPassphrase,
		workdir:         req.Workdir,
		env:             req.Env,
		strictHostCheck: req.StrictHostCheck,
		knownHosts:      req.KnownHosts,
	}

	// Validate required parameters
	if params.host == "" {
		return nil, fmt.Errorf("host parameter is required")
	}
	if params.user == "" {
		return nil, fmt.Errorf("user parameter is required")
	}
	if params.cmd == "" {
		return nil, fmt.Errorf("cmd parameter is required")
	}

	// Set default port
	if params.port <= 0 {
		params.port = 22
	}

	// Validate port range
	if params.port < 1 || params.port > 65535 {
		return nil, fmt.Errorf("invalid port number: %d", params.port)
	}

	// Parse timeout
	timeoutStr := req.Timeout
	if timeoutStr == "" {
		timeoutStr = "30s"
	}
	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		return nil, fmt.Errorf("invalid timeout format: %s", timeoutStr)
	}
	params.timeout = timeout

	// Validate authentication method
	if params.password == "" && params.keyFile == "" {
		return nil, fmt.Errorf("either password or key_file must be provided for authentication")
	}

	// Validate key file if provided
	if params.keyFile != "" {
		if err := validateKeyFile(params.keyFile); err != nil {
			return nil, err
		}
	}

	// Validate known hosts file if provided
	if params.knownHosts != "" {
		if err := validateKnownHostsFile(params.knownHosts); err != nil {
			return nil, err
		}
	}

	return params, nil
}

func validateKeyFile(keyFile string) error {
	// Expand tilde if present
	if strings.HasPrefix(keyFile, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		keyFile = filepath.Join(homeDir, keyFile[2:])
	}

	// Check if key file exists and is readable
	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		return fmt.Errorf("key file does not exist: %s", keyFile)
	}

	return nil
}

func validateKnownHostsFile(knownHostsFile string) error {
	// Expand tilde if present
	if strings.HasPrefix(knownHostsFile, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		knownHostsFile = filepath.Join(homeDir, knownHostsFile[2:])
	}

	// Check if known hosts file exists and is readable
	if _, err := os.Stat(knownHostsFile); os.IsNotExist(err) {
		return fmt.Errorf("known hosts file does not exist: %s", knownHostsFile)
	}

	return nil
}

func createSSHConfig(params *sshParams) (*ssh.ClientConfig, error) {
	config := &ssh.ClientConfig{
		User:    params.user,
		Timeout: params.timeout,
	}

	// Configure authentication
	var authMethods []ssh.AuthMethod

	// Password authentication
	if params.password != "" {
		authMethods = append(authMethods, ssh.Password(params.password))
	}

	// Key-based authentication
	if params.keyFile != "" {
		// Expand tilde if present
		keyFile := params.keyFile
		if strings.HasPrefix(keyFile, "~/") {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("failed to get home directory: %w", err)
			}
			keyFile = filepath.Join(homeDir, keyFile[2:])
		}

		key, err := os.ReadFile(keyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read private key file: %w", err)
		}

		var signer ssh.Signer
		if params.keyPassphrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase(key, []byte(params.keyPassphrase))
		} else {
			signer, err = ssh.ParsePrivateKey(key)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}

		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	config.Auth = authMethods

	// Configure host key verification
	if params.strictHostCheck {
		if params.knownHosts != "" {
			// Use custom known hosts file
			knownHostsFile := params.knownHosts
			if strings.HasPrefix(knownHostsFile, "~/") {
				homeDir, err := os.UserHomeDir()
				if err != nil {
					return nil, fmt.Errorf("failed to get home directory: %w", err)
				}
				knownHostsFile = filepath.Join(homeDir, knownHostsFile[2:])
			}

			hostKeyCallback, err := knownhosts.New(knownHostsFile)
			if err != nil {
				return nil, fmt.Errorf("failed to create known hosts callback: %w", err)
			}
			config.HostKeyCallback = hostKeyCallback
		} else {
			// Use default known hosts files
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("failed to get home directory: %w", err)
			}

			hostKeyCallback, err := knownhosts.New(
				filepath.Join(homeDir, ".ssh", "known_hosts"),
				"/etc/ssh/ssh_known_hosts",
			)
			if err != nil {
				return nil, fmt.Errorf("failed to create known hosts callback: %w", err)
			}
			config.HostKeyCallback = hostKeyCallback
		}
	} else {
		// Skip host key verification (not recommended for production)
		config.HostKeyCallback = ssh.InsecureIgnoreHostKey()
	}

	return config, nil
}

func (r *Req) Do() (re *Result, er error) {
	params, err := parseParams(r)
	if err != nil {
		return nil, err
	}

	result := &Result{Req: *r}

	// callback before
	if r.cb != nil && r.cb.before != nil {
		r.cb.before(params.host, params.port, params.user, params.cmd)
	}

	start := time.Now()

	// Create SSH configuration
	config, err := createSSHConfig(params)
	if err != nil {
		return result, fmt.Errorf("failed to create SSH config: %w", err)
	}

	// Connect to SSH server
	address := net.JoinHostPort(params.host, strconv.Itoa(params.port))
	client, err := ssh.Dial("tcp", address, config)
	if err != nil {
		return result, fmt.Errorf("failed to connect to SSH server: %w", err)
	}
	defer func() {
		err := client.Close()
		if er == nil {
			er = err
		}
	}()

	// Create SSH session
	session, err := client.NewSession()
	if err != nil {
		return result, fmt.Errorf("failed to create SSH session: %w", err)
	}
	// Note: We will explicitly close the session after command completion
	// instead of using defer to ensure proper cleanup timing

	// Set environment variables
	for key, value := range params.env {
		if err := session.Setenv(key, value); err != nil {
			// Some SSH servers don't allow setting environment variables
			// Log the error but continue execution
			continue
		}
	}

	// Prepare command with working directory if specified
	cmd := params.cmd
	if params.workdir != "" {
		cmd = fmt.Sprintf("cd %s && %s", params.workdir, params.cmd)
	}

	// Setup pipes for stdout and stderr to capture output separately
	stdoutPipe, err := session.StdoutPipe()
	if err != nil {
		return result, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderrPipe, err := session.StderrPipe()
	if err != nil {
		return result, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := session.Start(cmd); err != nil {
		return result, fmt.Errorf("failed to start command: %w", err)
	}

	// Create context with timeout for command execution
	ctx, cancel := context.WithTimeout(context.Background(), params.timeout)
	defer cancel()

	// Read stdout and stderr concurrently
	stdoutChan := make(chan []byte, 1)
	stderrChan := make(chan []byte, 1)
	waitChan := make(chan error, 1)

	go func() {
		defer close(stdoutChan)
		output, _ := io.ReadAll(stdoutPipe)
		stdoutChan <- output
	}()

	go func() {
		defer close(stderrChan)
		output, _ := io.ReadAll(stderrPipe)
		stderrChan <- output
	}()

	// Wait for command completion in a goroutine
	go func() {
		defer close(waitChan)
		waitChan <- session.Wait()
	}()

	// Wait for either command completion or timeout
	var cmdErr error
	select {
	case cmdErr = <-waitChan:
		// Command completed normally
	case <-ctx.Done():
		// Timeout or context cancellation
		// Try to signal the session to stop
		if signalErr := session.Signal(ssh.SIGTERM); signalErr != nil {
			// If SIGTERM fails, try SIGKILL
			_ = session.Signal(ssh.SIGKILL)
		}
		// Wait a bit for graceful termination, then proceed
		select {
		case cmdErr = <-waitChan:
			// Command terminated after signal
		case <-time.After(5 * time.Second):
			// Force termination
			cmdErr = fmt.Errorf("command execution timed out after %v", params.timeout)
		}
	}

	result.RT = time.Since(start)

	// Collect output (may still be available even if command failed/timed out)
	var stdoutBytes, stderrBytes []byte
	select {
	case stdoutBytes = <-stdoutChan:
	case <-time.After(1 * time.Second):
		stdoutBytes = []byte{}
	}

	select {
	case stderrBytes = <-stderrChan:
	case <-time.After(1 * time.Second):
		stderrBytes = []byte{}
	}

	stdout := string(stdoutBytes)
	stderr := string(stderrBytes)

	// Get exit code
	exitCode := 0
	if cmdErr != nil {
		if exitError, ok := cmdErr.(*ssh.ExitError); ok {
			exitCode = exitError.ExitStatus()
		} else {
			// Connection or other error
			return result, fmt.Errorf("SSH command execution failed: %w", cmdErr)
		}
	}

	// Determine status based on exit code (0 = success, 1 = failure)
	status := 1 // default to failure
	if exitCode == 0 {
		status = 0 // success
	}

	result.Res = Res{
		Code:   exitCode,
		Stdout: stdout,
		Stderr: stderr,
	}
	result.Status = status

	_ = session.Close()

	// callback after
	if r.cb != nil && r.cb.after != nil {
		r.cb.after(result)
	}

	return result, nil
}

// PrepareRequestData prepares SSH request data by extracting environment variables
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

func Execute(data map[string]string, opts ...Option) (map[string]string, error) {
	// Create a copy to avoid modifying the original data
	dataCopy := make(map[string]string)
	for k, v := range data {
		dataCopy[k] = v
	}

	// Prepare request data
	if err := PrepareRequestData(dataCopy); err != nil {
		return map[string]string{}, err
	}

	m := probe.HeaderToStringValue(probe.StructFlatToMap(dataCopy))

	r := NewReq()

	cb := &Callback{}
	for _, opt := range opts {
		opt(cb)
	}
	r.cb = cb

	if err := probe.MapToStructByTags(m, r); err != nil {
		return map[string]string{}, err
	}

	result, err := r.Do()
	if err != nil {
		return map[string]string{}, err
	}

	mapResult, err := probe.StructToMapByTags(result)
	if err != nil {
		return map[string]string{}, err
	}

	return probe.MapToStructFlat(mapResult)
}

func WithBefore(f func(host string, port int, user string, cmd string)) Option {
	return func(c *Callback) {
		c.before = f
	}
}

func WithAfter(f func(result *Result)) Option {
	return func(c *Callback) {
		c.after = f
	}
}
