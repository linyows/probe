package probe

import (
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/linyows/probe/actionref"
	"github.com/linyows/probe/actionrpc"
	"github.com/linyows/probe/mask"
)

// RunOptions controls how a single action invocation is run and logged.
type RunOptions struct {
	// Verbose raises the action's log level to debug.
	Verbose bool
	// Quiet silences the action's own log records. It is set for a retry
	// attempt that still has another attempt left: actions have no notion of
	// being retried, so a failure there would otherwise be reported at error
	// level even though the step goes on to succeed.
	Quiet bool
	// Masker hides the workflow's secrets in the action's log records.
	Masker *mask.Masker
	// BaseDir is the directory a local action's path is relative to: the
	// directory of the workflow file. The working directory is used when it
	// is empty.
	BaseDir string
}

// logLevel returns the level the action's log records are filtered at.
func (o RunOptions) logLevel() hclog.Level {
	switch {
	case o.Verbose:
		return hclog.Debug
	case o.Quiet:
		return hclog.Off
	default:
		return hclog.Warn
	}
}

// ActionRunner defines the interface for running actions
type ActionRunner interface {
	RunActions(name string, with map[string]any, opts RunOptions) (map[string]any, error)
}

// PluginActionRunner implements ActionRunner using the plugin system
type PluginActionRunner struct{}

// RunActions executes an action using the plugin system
func (p *PluginActionRunner) RunActions(name string, with map[string]any, opts RunOptions) (map[string]any, error) {
	// Actions are separate processes: they log to stderr as JSON and the
	// records are re-filtered here, so this level decides what the user sees.
	log := hclog.New(&hclog.LoggerOptions{
		Name:   "actions",
		Output: opts.Masker.Writer(os.Stderr),
		Level:  opts.logLevel(),
	})
	if !actionref.IsExternal(name) {
		return actionrpc.Run(name, with, log)
	}
	exe, err := actionref.Resolve(name, opts.BaseDir)
	if err != nil {
		return nil, err
	}
	return actionrpc.RunExecutable(exe.Path, exe.SHA256, with, log)
}

// MockActionRunner implements ActionRunner for testing
type MockActionRunner struct {
	Results map[string]map[string]any
	Errors  map[string]error
}

// NewMockActionRunner creates a new mock action runner
func NewMockActionRunner() *MockActionRunner {
	return &MockActionRunner{
		Results: make(map[string]map[string]any),
		Errors:  make(map[string]error),
	}
}

// SetResult sets the expected result for an action
func (m *MockActionRunner) SetResult(actionName string, result map[string]any) {
	m.Results[actionName] = result
}

// SetError sets the expected error for an action
func (m *MockActionRunner) SetError(actionName string, err error) {
	m.Errors[actionName] = err
}

// RunActions returns the mocked result or error for the given action
func (m *MockActionRunner) RunActions(name string, with map[string]any, opts RunOptions) (map[string]any, error) {
	if err, exists := m.Errors[name]; exists {
		return nil, err
	}

	if result, exists := m.Results[name]; exists {
		return result, nil
	}

	// Default mock response
	return map[string]any{
		"code":    0,
		"mock":    true,
		"action":  name,
		"with":    with,
		"results": map[string]any{},
	}, nil
}
