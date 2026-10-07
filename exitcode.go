package probe

// Exit codes of a probe run. They tell apart what a caller such as CI or cron
// has to do about a failure: a failed test points at the system under test,
// a configuration error at the workflow or the command line, and an action
// error at the environment, such as a target that cannot be reached.
const (
	// ExitOK means every job succeeded.
	ExitOK = 0
	// ExitTestFailed means a test did not hold: it evaluated to false, could
	// not be evaluated, or did not evaluate to a boolean; or a request or a
	// response broke the contract an action checked it against.
	ExitTestFailed = 1
	// ExitConfigError means the workflow or the command line was wrong, or a
	// report file could not be written, so the run could not do what it was
	// asked to.
	ExitConfigError = 2
	// ExitActionError means an action returned an error, such as a refused
	// connection or a timeout, so a test could not be checked at all.
	ExitActionError = 3
)

// failureConfig is recorded when a job cannot start because its own
// definition is invalid, for example a malformed step ID.
const failureConfig = "config"

// recordFailure notes that a failure of the given kind happened during the run.
func (rs *Result) recordFailure(kind string) {
	if rs == nil || kind == "" {
		return
	}
	rs.failuresMu.Lock()
	defer rs.failuresMu.Unlock()
	if rs.failures == nil {
		rs.failures = make(map[string]bool)
	}
	rs.failures[kind] = true
}

// exitCode maps the failures recorded during a run to an exit code. When
// several kinds occurred, the one a caller has to act on first wins: a broken
// workflow before an unreachable target, and an unreachable target before a
// failed test, since the target being down usually explains the tests.
func (rs *Result) exitCode(failed bool) int {
	if !failed {
		return ExitOK
	}

	rs.failuresMu.Lock()
	defer rs.failuresMu.Unlock()

	switch {
	case rs.failures[failureConfig]:
		return ExitConfigError
	case rs.failures[FailureAction]:
		return ExitActionError
	default:
		return ExitTestFailed
	}
}
