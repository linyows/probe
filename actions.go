package probe

import (
	"os"
	"sync"

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

// StatefulActionRunner is an ActionRunner that also carries the state an
// action keeps in a job, as actionrpc.StatefulAction describes. A runner
// that is not one runs every action without state.
type StatefulActionRunner interface {
	ActionRunner
	RunActionsWithState(name string, with, state map[string]any, opts RunOptions) (result, newState map[string]any, err error)
}

// PluginActionRunner implements ActionRunner using the plugin system
type PluginActionRunner struct{}

// RunActions executes an action using the plugin system
func (p *PluginActionRunner) RunActions(name string, with map[string]any, opts RunOptions) (map[string]any, error) {
	result, _, err := p.RunActionsWithState(name, with, nil, opts)
	return result, err
}

// RunActionsWithState executes an action using the plugin system, giving it
// the state it left in the job and returning the state it leaves.
func (p *PluginActionRunner) RunActionsWithState(name string, with, state map[string]any, opts RunOptions) (map[string]any, map[string]any, error) {
	// Actions are separate processes: they log to stderr as JSON and the
	// records are re-filtered here, so this level decides what the user sees.
	log := hclog.New(&hclog.LoggerOptions{
		Name:   "actions",
		Output: opts.Masker.Writer(os.Stderr),
		Level:  opts.logLevel(),
	})
	if !actionref.IsExternal(name) {
		return actionrpc.RunWithState(name, with, state, log)
	}
	exe, err := actionref.Resolve(name, opts.BaseDir)
	if err != nil {
		return nil, nil, err
	}
	return actionrpc.RunExecutableWithState(exe.Path, exe.SHA256, with, state, log)
}

// runAction runs the action named name with the state it left in the job,
// when runner carries state, and returns the state it leaves.
func runAction(runner ActionRunner, name string, with, state map[string]any, opts RunOptions) (map[string]any, map[string]any, error) {
	if sr, ok := runner.(StatefulActionRunner); ok {
		return sr.RunActionsWithState(name, with, state, opts)
	}
	result, err := runner.RunActions(name, with, opts)
	return result, nil, err
}

// actionStates holds the state each action keeps in one run of a job, keyed
// by the action's name as the steps write it in uses.
type actionStates struct {
	mu     sync.Mutex
	states map[string]map[string]any
}

func newActionStates() *actionStates {
	return &actionStates{states: make(map[string]map[string]any)}
}

// get returns the state the action left, or nil when it left none.
func (s *actionStates) get(name string) map[string]any {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.states[name]
}

// set records the state the action leaves. A nil state keeps the one there
// was.
func (s *actionStates) set(name string, state map[string]any) {
	if s == nil || state == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.states[name] = state
}

// MockActionRunner implements ActionRunner for testing
type MockActionRunner struct {
	Results map[string]map[string]any
	Errors  map[string]error
	// States is the state each action leaves in the job, keyed by its name.
	States map[string]map[string]any
	// Received records the state each call to an action was given, in order.
	Received map[string][]map[string]any

	mu sync.Mutex
}

// NewMockActionRunner creates a new mock action runner
func NewMockActionRunner() *MockActionRunner {
	return &MockActionRunner{
		Results:  make(map[string]map[string]any),
		Errors:   make(map[string]error),
		States:   make(map[string]map[string]any),
		Received: make(map[string][]map[string]any),
	}
}

// RunActionsWithState records the state the action is given and returns the
// result and the state set for it.
func (m *MockActionRunner) RunActionsWithState(name string, with, state map[string]any, opts RunOptions) (map[string]any, map[string]any, error) {
	m.mu.Lock()
	if m.Received == nil {
		m.Received = make(map[string][]map[string]any)
	}
	m.Received[name] = append(m.Received[name], state)
	newState := m.States[name]
	m.mu.Unlock()
	result, err := m.RunActions(name, with, opts)
	if err != nil {
		return nil, nil, err
	}
	return result, newState, nil
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
