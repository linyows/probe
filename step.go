package probe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/linyows/probe/actionrpc"
	"github.com/linyows/probe/expr"
	"github.com/linyows/probe/jsonutil"
	"github.com/linyows/probe/mask"
	"github.com/linyows/probe/procgroup"
)

const (
	// DefaultStepTimeout is the default timeout for action execution
	DefaultStepTimeout = 5 * time.Minute
)

type Step struct {
	Name         string            `yaml:"name"`
	ID           string            `yaml:"id,omitempty"`
	Uses         string            `yaml:"uses" validate:"required"`
	With         map[string]any    `yaml:"with"`
	Test         string            `yaml:"test"`
	Echo         string            `yaml:"echo"`
	Vars         map[string]any    `yaml:"vars"`
	Iteration    []map[string]any  `yaml:"iteration"`
	Wait         string            `yaml:"wait,omitempty"`
	SkipIf       string            `yaml:"skipif,omitempty"`
	Outputs      map[string]string `yaml:"outputs,omitempty"`
	Retry        *StepRetry        `yaml:"retry,omitempty"`
	Timeout      Interval          `yaml:"timeout,omitempty"`
	err          error
	ctx          StepContext
	retryAttempt int
	// attempt is the attempt of the action running now, from 1, which the
	// action is told about.
	attempt int
	// expandedName is the name with its templates evaluated.
	expandedName string
	startedAt    time.Time
	failure      *StepFailure
	// templateErr holds the templates of the step's name and vars that
	// could not be evaluated, so that the step fails rather than run with
	// them.
	templateErr  error
	Idx          int          `yaml:"-"`
	Expr         *expr.Expr   `yaml:"-"`
	actionRunner ActionRunner `yaml:"-"`
}

func (st *Step) Do(jCtx *JobContext) {
	// An iteration runs the same Step again, so what the last run left must
	// not be reported as this one's, above all when it fails before its
	// action runs.
	st.startedAt = time.Time{}
	st.retryAttempt = 0
	st.err = nil

	// 1. Preparation phase: validation, wait, skip check
	name, shouldContinue := st.prepare(jCtx)
	if !shouldContinue {
		return
	}

	if st.templateErr != nil {
		st.handleActionError(st.templateErr, name, jCtx)
		return
	}

	// 2. Action execution phase
	st.startedAt = time.Now()
	actionResult, err := st.executeAction(name, jCtx)
	if err != nil {
		st.handleActionError(err, name, jCtx)
		return
	}

	// 3. Result processing phase
	st.processActionResult(actionResult, jCtx)

	// 4. Finalization phase: test, echo, output save, result creation
	st.finalize(name, actionResult, jCtx)
}

// prepare handles step preparation: validation, skip check, and wait
// Returns (stepName, shouldContinue)
func (st *Step) prepare(jCtx *JobContext) (string, bool) {
	// Set default name if empty
	if st.Name == "" {
		st.Name = "Unknown Step"
	}

	// Evaluate step name. A name that cannot be evaluated is shown as
	// written, and fails the step unless it is skipped.
	name, err := st.Expr.EvalTemplate(st.Name, st.ctx)
	if err != nil {
		name = st.Name
		st.templateErr = errors.Join(&expr.FieldError{Path: "name", Err: err}, st.templateErr)
	}

	st.expandedName = name
	jCtx.Printer.StepStart(jCtx.CurrentJobID, name)

	// Check if step should be skipped BEFORE waiting
	if st.shouldSkip(jCtx) {
		st.handleSkip(name, jCtx)
		return name, false
	}

	// Handle wait only if step is not skipped
	st.handleWait(jCtx)

	return name, true
}

// executeAction executes the step action and returns the result
// If retry is configured, it will retry until status == 0 or max attempts reached
func (st *Step) executeAction(name string, jCtx *JobContext) (map[string]any, error) {
	expW, err := st.Expr.EvalTemplateMap(st.With, st.ctx)
	if err != nil {
		// The credentials that could be evaluated are learned, so that an
		// error about another value cannot show them, and the details of a
		// credential's own error are left out.
		jCtx.Printer.Masker().Learn(expW)
		return nil, redactCredentialErrors(prefixFieldErrors(err, "with."))
	}

	runner := st.actionRunner
	if runner == nil {
		runner = &PluginActionRunner{} // Default to plugin execution
	}
	st.attempt = 1

	// If no retry configuration, execute once
	if st.Retry == nil {
		return st.executeSingleAction(runner, expW, jCtx, false)
	}

	// Execute with retry logic
	return st.executeActionWithRetry(runner, expW, jCtx, name)
}

// executeSingleAction executes action once without retry
// quiet marks an attempt whose failure will be retried, so that the action's
// own error records stay out of the log.
func (st *Step) executeSingleAction(runner ActionRunner, expW map[string]any, jCtx *JobContext, quiet bool) (map[string]any, error) {
	// Determine timeout duration
	timeout := DefaultStepTimeout
	if st.Timeout.Duration > 0 {
		timeout = st.Timeout.Duration
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Execute action in goroutine
	type result struct {
		ret   map[string]any
		state map[string]any
		err   error
	}
	resultCh := make(chan result, 1)

	// Learn the credentials the action is about to send before it runs, so
	// that its own log records already have them hidden.
	masker := jCtx.Printer.Masker()
	masker.Learn(expW)

	// A process the action starts in the background is tracked here rather
	// than after the select, so that one started after the step timed out is
	// still stopped with the others.
	done := beginBackground(jCtx.background, st.Uses)
	// The call is made up here: an attempt that timed out goes on in the
	// background while the next one changes the step.
	opts := RunOptions{Verbose: jCtx.Verbose, Quiet: quiet, Masker: masker, BaseDir: jCtx.baseDir}
	call := actionrpc.Call{With: expW, State: jCtx.states.get(st.Uses), Step: st.stepInfo(jCtx)}
	go func() {
		defer done()
		ret, state, err := runAction(runner, st.Uses, call, opts)
		if err == nil {
			trackBackground(jCtx.background, st.Uses, ret)
		}
		resultCh <- result{ret: ret, state: state, err: err}
	}()

	// Wait for either completion or timeout. The state of an action that
	// timed out is not kept, as its result is not.
	select {
	case res := <-resultCh:
		if res.err == nil {
			if err := jCtx.states.set(st.Uses, res.state); err != nil {
				return nil, err
			}
		}
		return res.ret, res.err
	case <-ctx.Done():
		return nil, errors.New("action execution timed out after " + timeout.String())
	}
}

// stepInfo tells an action about the step it runs for.
func (st *Step) stepInfo(jCtx *JobContext) actionrpc.Step {
	attempt := st.attempt
	if attempt < 1 {
		attempt = 1
	}
	return actionrpc.Step{
		RunID:   jCtx.runID,
		JobID:   jCtx.CurrentJobID,
		JobName: jCtx.jobName,
		Index:   st.Idx,
		ID:      st.ID,
		Name:    st.expandedName,
		Repeat:  jCtx.RepeatCurrent,
		Attempt: attempt,
	}
}

// beginBackground tells the tracker that an action that can start a
// background process is about to run. Only the shell action starts one.
func beginBackground(t *procgroup.Tracker, uses string) func() {
	if uses != "shell" {
		return func() {}
	}
	return t.Begin()
}

// trackBackground records the process an action left running, if it did.
func trackBackground(t *procgroup.Tracker, uses string, ret map[string]any) {
	if pid, log, ok := backgroundProcess(uses, ret); ok {
		t.Track(pid, log)
	}
}

// backgroundProcess returns the process group a step's action started in the
// background, and the file its output goes to. Only a shell step asked to run
// in the background starts one.
func backgroundProcess(uses string, ret map[string]any) (pid int, log string, ok bool) {
	if uses != "shell" {
		return 0, "", false
	}
	req, _ := ret["req"].(map[string]any)
	if bg, _ := req["background"].(bool); !bg {
		return 0, "", false
	}
	res, _ := ret["res"].(map[string]any)
	pid = toInt(res["pid"])
	if pid <= 0 {
		return 0, "", false
	}
	log, _ = res["log"].(string)
	return pid, log, true
}

func toInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}

// executeActionWithRetry executes action with retry logic based on test assertion results.
// If no test is configured, retry is not performed and the action is executed once.
func (st *Step) executeActionWithRetry(runner ActionRunner, expW map[string]any, jCtx *JobContext, name string) (map[string]any, error) {
	retry := st.Retry
	st.retryAttempt = 0

	// If no test is configured, don't retry - execute once
	if st.Test == "" {
		return st.executeSingleAction(runner, expW, jCtx, false)
	}

	// Initial delay if specified
	if retry.InitialDelay.Duration > 0 {
		if jCtx.Verbose {
			jCtx.Printer.LogDebug("Initial delay before first attempt: %v", retry.InitialDelay.Duration)
		}
		time.Sleep(retry.InitialDelay.Duration)
	}

	var lastResult map[string]any
	var lastErr error

	// Retry loop
	for attempt := 1; attempt <= retry.MaxAttempts; attempt++ {
		if jCtx.Verbose {
			jCtx.Printer.LogDebug("Executing action with retry: attempt %d/%d", attempt, retry.MaxAttempts)
		}

		// Only the final attempt reports a failure as such; before that the
		// step still has a chance to succeed.
		st.attempt = attempt
		result, err := st.executeSingleAction(runner, expW, jCtx, attempt < retry.MaxAttempts)
		lastResult = result
		lastErr = err

		if err != nil {
			// Action execution failed, retry
			if attempt < retry.MaxAttempts {
				if jCtx.Verbose {
					jCtx.Printer.LogDebug("Action execution failed (attempt %d), retrying after %v", attempt, retry.Interval.Duration)
				}
				time.Sleep(retry.Interval.Duration)
			}
			continue
		}

		// Process action result to set context for test evaluation
		st.processActionResult(result, jCtx)

		// Evaluate test expression to determine success
		exprOut, err := st.evalTest()
		testOk := err == nil && exprOut == true
		if testOk {
			if jCtx.Verbose {
				jCtx.Printer.LogDebug("Action succeeded on attempt %d", attempt)
			}
			st.retryAttempt = attempt
			return result, nil
		}

		// Test failed, retry if not the last attempt
		if attempt < retry.MaxAttempts {
			if jCtx.Verbose {
				jCtx.Printer.LogDebug("Test failed (attempt %d), retrying after %v", attempt, retry.Interval.Duration)
			}
			time.Sleep(retry.Interval.Duration)
		}
	}

	st.retryAttempt = retry.MaxAttempts

	// All attempts failed
	if jCtx.Verbose {
		jCtx.Printer.LogDebug("All retry attempts failed (%d attempts)", retry.MaxAttempts)
	}
	return lastResult, lastErr
}

// handleActionError handles a step that failed before its test could run:
// the action returned an error, or a template the action needed could not be
// evaluated, in which case the action did not run.
func (st *Step) handleActionError(err error, name string, jCtx *JobContext) {
	kind := failureKindOf(err)
	if kind == FailureTemplate {
		st.err = err
		jCtx.Printer.PrintError("Template evaluation failed: %v", err)
	} else {
		actionErr := NewActionError("step_execute", "action execution failed", err).
			WithContext("step_name", name).
			WithContext("action_type", st.Uses)
		st.err = actionErr
		jCtx.Printer.PrintError("Action execution failed: %v", actionErr)
	}
	jCtx.SetFailed()
	jCtx.Result.recordFailure(kind)

	// Create and add step result for failed action execution
	if jCtx.Verbose {
		jCtx.Printer.PrintRequestResponse(st.Idx, name, st.ctx.Req, st.ctx.Res, st.ctx.RT.Duration)
	}

	// Handle repeat execution
	if jCtx.IsRepeating {
		st.handleRepeatExecution(jCtx, name, true) // true = hasError
		return
	}

	// Standard execution: create result for failed step
	stepResult := st.createFailedStepResult(name, jCtx)

	// Add step result to workflow buffer
	if jCtx.Result != nil {
		jCtx.Result.AddStepResult(jCtx.CurrentJobID, stepResult)
	}

	if jCtx.Verbose {
		jCtx.Printer.PrintSeparator()
	}
}

// processActionResult processes the action result and updates context
func (st *Step) processActionResult(actionResult map[string]any, jCtx *JobContext) {
	// Parse and process JSON response
	req, _ := actionResult["req"].(map[string]any)
	res, okres := actionResult["res"].(map[string]any)
	rt, _ := actionResult["rt"].(string)

	status := parseExitStatus(actionResult["status"])

	if okres {
		body, okbody := res["body"].(string)
		if okbody && jsonutil.LooksLikeJSON(body) {
			res["rawbody"] = body
			res["body"] = jsonutil.Decode(body)
		}

		// A command that prints a JSON object or array, such as a CLI asked
		// for JSON output, is read as res.json. stdout is kept as it is, for
		// tests that read the text.
		if _, taken := res["json"]; !taken {
			if v, ok := decodeStdoutJSON(res["stdout"]); ok {
				res["json"] = v
			}
		}
	}

	// A response can hand out a credential too, such as a session cookie.
	jCtx.Printer.Masker().Learn(res)

	// Update context with status
	st.updateCtx(nil, req, res, rt, status)
}

// decodeStdoutJSON decodes stdout when the whole of it is one JSON object or
// array, surrounding whitespace aside. Anything else, such as plain text,
// JSON Lines or a scalar, is not decoded.
func decodeStdoutJSON(stdout any) (any, bool) {
	s, ok := stdout.(string)
	if !ok || !jsonutil.LooksLikeJSON(s) || !json.Valid([]byte(s)) {
		return nil, false
	}
	return jsonutil.Decode(s), true
}

// finalize handles the final phase: test, echo, output save, and result creation
func (st *Step) finalize(name string, actionResult map[string]any, jCtx *JobContext) {
	if jCtx.Verbose {
		jCtx.Printer.PrintRequestResponse(st.Idx, name, st.ctx.Req, st.ctx.Res, st.ctx.RT.Duration)
	}

	// Outputs are saved before the test and echo, as always; a repeated job
	// used to skip this, so its later steps could not read its earlier ones.
	st.saveOutputs(jCtx)

	// Handle repeat execution
	if jCtx.IsRepeating {
		st.handleRepeatExecution(jCtx, name, false) // false = no error
		return
	}

	// Standard execution: create result
	stepResult := st.createStepResult(name, jCtx)

	// Add step result to workflow buffer
	if jCtx.Result != nil {
		jCtx.Result.AddStepResult(jCtx.CurrentJobID, stepResult)
	}

	if jCtx.Verbose {
		jCtx.Printer.PrintSeparator()
	}
}

// createStepResult creates a StepResult from step execution
func (st *Step) createStepResult(name string, jCtx *JobContext) StepResult {
	result := StepResult{
		Index:    st.Idx,
		Name:     name,
		HasTest:  st.Test != "",
		RT:       "",
		WaitTime: st.getWaitTimeForDisplay(),
		Test:     st.Test,
		Elapsed:  st.elapsed(),
	}

	if st.Retry != nil && st.retryAttempt > 0 {
		result.RetryAttempt = st.retryAttempt
		result.RetryMax = st.Retry.MaxAttempts
	}

	if jCtx.Timing {
		if !st.startedAt.IsZero() {
			result.StartedAt = st.startedAt
		}
		if st.ctx.RT.Duration != "" {
			result.RT = st.ctx.RT.Duration
			result.RTSec = st.ctx.RT.Sec
		}
	}
	if v, ok := st.ctx.Res["report"]; ok {
		if report, sok := v.(string); sok {
			result.Report = report
		}
	}

	if st.Test != "" {
		testOutput, ok := st.DoTest(jCtx.Printer)
		if ok {
			result.Status = StatusSuccess
		} else {
			result.Status = StatusError
			result.TestOutput = testOutput
			result.Failure = st.failure
			jCtx.SetFailed()
			jCtx.Result.recordFailure(st.failure.Kind)
		}
	} else {
		result.Status = StatusWarning
	}

	if st.Echo != "" {
		result.EchoOutput = st.getEchoOutput(jCtx.Printer)
	}

	return result
}

// getEchoOutput returns the echo output as string
func (st *Step) getEchoOutput(printer *Printer) string {
	exprOut, err := st.Expr.EvalTemplate(st.Echo, st.ctx)
	return printer.generateEchoOutput(exprOut, err)
}

func (st *Step) handleRepeatExecution(jCtx *JobContext, name string, hasError bool) {
	// Execute test first (outside of lock)
	hasTest := st.Test != ""
	testResult := true

	// If there was an error, always count as failure
	if hasError {
		testResult = false
	} else if hasTest {
		_, testResult = st.DoTest(jCtx.Printer)
		if !testResult {
			jCtx.SetFailed()
			jCtx.Result.recordFailure(st.failure.Kind)
		}
	}

	var failure *StepFailure
	switch {
	case hasError && st.err != nil:
		failure = st.newFailure(failureKindOf(st.err), st.err.Error())
	case hasTest && !testResult:
		failure = st.failure
	}

	// Evaluate echo before taking the lock so we can store the formatted
	// output on the counter alongside the success/failure increment.
	var echoRaw, echoFormatted string
	var echoErr error
	if st.Echo != "" {
		echoRaw, echoErr = st.Expr.EvalTemplate(st.Echo, st.ctx)
		echoFormatted = jCtx.Printer.generateEchoOutput(echoRaw, echoErr)
	}

	// Update counter atomically with lock held throughout
	jCtx.countersMu.Lock()
	counter, exists := jCtx.StepCounters[st.Idx]
	if !exists {
		counter = StepRepeatCounter{
			Name:        name,
			RepeatTotal: jCtx.RepeatTotal,
		}
	}

	// Update counts based on result
	if hasError || (hasTest && !testResult) {
		counter.FailureCount++
	} else {
		counter.SuccessCount++
	}
	counter.LastResult = testResult
	if counter.Failure == nil {
		counter.Failure = failure
	}

	if st.Echo != "" {
		counter.EchoOutputs = append(counter.EchoOutputs, echoFormatted)
	}

	// Store updated counter back to map. The step's result is made from the
	// counter once every run is done (Executor.appendRepeatStepResults).
	jCtx.StepCounters[st.Idx] = counter
	jCtx.countersMu.Unlock()

	if st.Echo != "" {
		if echoErr != nil {
			jCtx.Printer.LogError("Echo evaluation failed: %#v (input: %s)", echoErr, st.Echo)
		} else {
			jCtx.Printer.PrintEchoContent(echoRaw)
		}
	}
}

// evalTest evaluates the test expression and returns the raw result
// without generating any formatted output.
func (st *Step) evalTest() (any, error) {
	return st.Expr.Eval(st.Test, st.ctx)
}

func (st *Step) DoTest(printer *Printer) (string, bool) {
	st.failure = nil

	exprOut, err := st.evalTest()
	if err != nil {
		st.failure = &StepFailure{Kind: FailureTestError, Message: err.Error()}
		return printer.generateTestError(st.Test, err), false
	}

	boolOutput, boolOk := exprOut.(bool)
	if !boolOk {
		st.failure = &StepFailure{
			Kind:    FailureTestType,
			Message: fmt.Sprintf("test evaluated to %v (%T), not a boolean", exprOut, exprOut),
		}
		return printer.generateTestTypeMismatch(st.Test, exprOut), false
	}

	if printer.verbose {
		printer.PrintTestResult(boolOutput, st.Test, st.ctx)
	}

	if !boolOutput {
		st.failure = st.newFailure(FailureAssertion, "test evaluated to false")
		return printer.generateTestFailure(st.Test, exprOut, st.ctx.Req, st.ctx.Res), false
	}

	return "", true
}

// newFailure builds a StepFailure carrying the request and response, unless
// the action opted out of dumping them with res.dump: false.
func (st *Step) newFailure(kind, message string) *StepFailure {
	f := &StepFailure{Kind: kind, Message: message}
	if dump, ok := st.ctx.Res["dump"].(bool); ok && !dump {
		return f
	}
	f.Request = st.ctx.Req
	f.Response = st.ctx.Res
	return f
}

// elapsed is the time since the action started, or zero if it never did.
func (st *Step) elapsed() time.Duration {
	if st.startedAt.IsZero() {
		return 0
	}
	return time.Since(st.startedAt)
}

func (st *Step) SetCtx(j JobContext, override map[string]any) {
	// Use outputs from the unified Outputs structure
	var outputs map[string]any
	if j.Outputs != nil {
		outputs = j.Outputs.GetAll()
	}

	// Create context for step vars evaluation
	evalCtx := StepContext{
		Vars:        j.Vars,
		Outputs:     outputs,
		RepeatIndex: j.RepeatCurrent,
	}

	// Evaluate step-level vars with access to outputs. A var that cannot be
	// evaluated is left out, and fails the step when it runs.
	st.templateErr = nil
	evaluatedStepVars := make(map[string]any)
	if len(st.Vars) > 0 {
		ev := &expr.Expr{}
		var errs []error
		// Sorted so that the errors come in the same order on every run.
		for _, k := range slices.Sorted(maps.Keys(st.Vars)) {
			out, err := evalVar(ev, k, st.Vars[k], evalCtx)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			evaluatedStepVars[k] = out
		}
		st.templateErr = errors.Join(errs...)
	}

	// Merge workflow vars with evaluated step vars
	vers := mergeMaps(j.Vars, evaluatedStepVars)
	if override != nil {
		vers = mergeMaps(vers, override)
	}

	st.ctx = StepContext{
		Vars:        vers,
		Outputs:     outputs,
		RepeatIndex: j.RepeatCurrent,
	}
}

// parseExitStatus converts various status representations to int
func parseExitStatus(status any) int {
	if status == nil {
		return int(ExitStatusFailure) // default to failure if status is nil
	}

	switch v := status.(type) {
	case int:
		return v
	case int64:
		// Handle integers converted from protobuf float64 by the actionrpc client
		return int(v)
	case float64:
		// Handle JSON numbers which are parsed as float64
		return int(v)
	case string:
		if v == "0" {
			return int(ExitStatusSuccess)
		} else {
			return int(ExitStatusFailure)
		}
	case ExitStatus:
		return int(v)
	default:
		return int(ExitStatusFailure) // default to failure for unknown types
	}
}

func (st *Step) updateCtx(logs []map[string]any, req, res map[string]any, rt string, status int) {
	st.ctx.Req = req
	st.ctx.Res = res
	st.ctx.Status = status

	// Parse RT string to populate RT structure
	if rt != "" {
		if duration, err := time.ParseDuration(rt); err == nil {
			st.ctx.RT = ResponseTime{
				Duration: rt,
				Sec:      duration.Seconds(),
			}
		}
	} else {
		st.ctx.RT = ResponseTime{}
	}
}

// handleWait processes the wait field and sleeps if necessary
func (st *Step) handleWait(jCtx *JobContext) {
	if st.Wait == "" {
		return
	}

	// Evaluate wait expression template first
	waitExpr, err := st.Expr.EvalTemplate(st.Wait, st.ctx)
	if err != nil {
		jCtx.Printer.PrintError("wait expression evaluation error: %v", err)
		return
	}

	duration, err := st.parseWaitDuration(waitExpr)
	if err != nil {
		jCtx.Printer.PrintError("wait duration parsing error: %v", err)
		return
	}

	if duration > 0 {
		msg := colorWarning().Sprintf("(%s wait)", st.formatWaitTime(duration))
		msg = fmt.Sprintf("%s %s", msg, st.Name)
		sleepWithMessage(duration, msg, jCtx.Printer.AddSpinnerSuffix)
	}
}

func sleepWithMessage(d time.Duration, m string, fn func(m string)) {
	if d < time.Second {
		time.Sleep(d)
		return
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	timer := time.NewTimer(d)
	defer timer.Stop()

	for {
		select {
		case <-ticker.C:
			fn(m)
		case <-timer.C:
			return
		}
	}
}

// parseWaitDuration parses wait string to time.Duration
func (st *Step) parseWaitDuration(wait string) (time.Duration, error) {
	// Check if it's a plain number (treat as seconds for backward compatibility)
	if matched, _ := regexp.MatchString(`^\d+$`, wait); matched {
		if seconds, err := strconv.Atoi(wait); err == nil {
			return time.Duration(seconds) * time.Second, nil
		}
		return 0, fmt.Errorf("invalid wait value: %s", wait)
	}

	// Parse as duration string (e.g., "1s", "500ms", "2m")
	duration, err := time.ParseDuration(wait)
	if err != nil {
		return 0, fmt.Errorf("invalid wait format: %s", wait)
	}

	return duration, nil
}

// formatWaitTime formats duration for display
func (st *Step) formatWaitTime(duration time.Duration) string {
	if duration < time.Second {
		return duration.String()
	}
	if duration%time.Second == 0 {
		return fmt.Sprintf("%ds", int(duration/time.Second))
	}
	return duration.String()
}

// getWaitTimeForDisplay returns formatted wait time for display
func (st *Step) getWaitTimeForDisplay() string {
	if st.Wait == "" {
		return ""
	}

	// Note: st.ctx might not be fully initialized when this is called during result creation
	// So we need to handle template evaluation carefully
	waitExpr := st.Wait
	if st.Expr != nil {
		if evaluated, err := st.Expr.EvalTemplate(st.Wait, st.ctx); err == nil {
			waitExpr = evaluated
		}
	}

	duration, err := st.parseWaitDuration(waitExpr)
	if err != nil {
		// If template evaluation failed and we still have template syntax, return empty
		// This prevents display errors during result creation
		if strings.Contains(waitExpr, "{{") {
			return ""
		}
		return ""
	}

	return st.formatWaitTime(duration)
}

// shouldSkip evaluates the skipif expression and returns true if step should be skipped
func (st *Step) shouldSkip(jCtx *JobContext) bool {
	if st.SkipIf == "" {
		return false
	}

	result, err := st.Expr.Eval(st.SkipIf, st.ctx)
	if err != nil {
		jCtx.Printer.PrintError("skipif evaluation error: %v", err)
		return false // Don't skip on evaluation error
	}

	boolResult, ok := result.(bool)
	if !ok {
		jCtx.Printer.PrintError("skipif expression must return boolean, got: %T", result)
		return false // Don't skip on type error
	}

	return boolResult
}

// handleSkip handles the skipped step logic
func (st *Step) handleSkip(name string, jCtx *JobContext) {
	if jCtx.Verbose {
		jCtx.Printer.LogDebug("%s", colorWarning().Sprintf("--- Step %d: %s (SKIPPED)", st.Idx, name))
		jCtx.Printer.LogDebug("Skip condition: %s", st.SkipIf)
		jCtx.Printer.PrintSeparator()
		return
	}

	// Handle repeat execution for skipped steps
	if jCtx.IsRepeating {
		st.handleSkipRepeatExecution(jCtx, name)
		return
	}

	// Create step result for skipped step
	stepResult := st.createSkippedStepResult(name, jCtx)

	// Add step result to workflow buffer
	if jCtx.Result != nil {
		jCtx.Result.AddStepResult(jCtx.CurrentJobID, stepResult)
	}
}

// handleSkipRepeatExecution handles skipped step in repeat mode
func (st *Step) handleSkipRepeatExecution(jCtx *JobContext, name string) {
	// Update counter atomically with lock held throughout
	jCtx.countersMu.Lock()
	counter, exists := jCtx.StepCounters[st.Idx]
	if !exists {
		counter = StepRepeatCounter{
			Name:        name,
			RepeatTotal: jCtx.RepeatTotal,
		}
	}

	// Count as successful (skipped is not a failure)
	counter.SuccessCount++
	counter.LastResult = true

	// Store updated counter back to map. The step's result is made from the
	// counter once every run is done (Executor.appendRepeatStepResults).
	jCtx.StepCounters[st.Idx] = counter
	jCtx.countersMu.Unlock()
}

// createSkippedStepResult creates a StepResult for a skipped step
func (st *Step) createSkippedStepResult(name string, jCtx *JobContext) StepResult {
	return StepResult{
		Index:    st.Idx,
		Name:     name + " (SKIPPED)",
		Status:   StatusSkipped,
		RT:       "",
		WaitTime: st.getWaitTimeForDisplay(),
		HasTest:  false,
	}
}

// createFailedStepResult creates a StepResult for a failed step
func (st *Step) createFailedStepResult(name string, jCtx *JobContext) StepResult {
	result := StepResult{
		Index:    st.Idx,
		Name:     name,
		Status:   StatusError,
		RT:       "",
		WaitTime: st.getWaitTimeForDisplay(),
		HasTest:  st.Test != "",
		Test:     st.Test,
		Elapsed:  st.elapsed(),
	}

	if st.Retry != nil && st.retryAttempt > 0 {
		result.RetryAttempt = st.retryAttempt
		result.RetryMax = st.Retry.MaxAttempts
	}

	if jCtx.Timing {
		if !st.startedAt.IsZero() {
			result.StartedAt = st.startedAt
		}
		if st.ctx.RT.Duration != "" {
			result.RT = st.ctx.RT.Duration
			result.RTSec = st.ctx.RT.Sec
		}
	}
	if v, ok := st.ctx.Res["report"]; ok {
		if report, sok := v.(string); sok {
			result.Report = report
		}
	}

	// Include error information if available
	if st.err != nil {
		result.TestOutput = indentDetail(st.err.Error())
		result.Failure = st.newFailure(failureKindOf(st.err), st.err.Error())
	}

	return result
}

// errCredentialTemplate replaces why the template of a credential could not
// be evaluated: the template, the excerpt of it in the error, and a value in
// the error, such as the input of a failed conversion, may each hold the
// credential.
var errCredentialTemplate = errors.New("the template could not be evaluated; the details are not shown, as the value is a credential")

// redactCredentialErrors replaces the error of each *expr.FieldError joined
// into err whose path goes through a credential, such as
// with.headers.authorization or with.password, keeping the path.
func redactCredentialErrors(err error) error {
	var errs []error
	for _, e := range unwrapJoined(err) {
		var fe *expr.FieldError
		if errors.As(e, &fe) && pathHoldsCredential(fe.Path) {
			e = &expr.FieldError{Path: fe.Path, Err: errCredentialTemplate}
		}
		errs = append(errs, e)
	}
	return errors.Join(errs...)
}

// pathHoldsCredential reports whether any key in path, such as
// headers.cookie[0], names a credential.
func pathHoldsCredential(path string) bool {
	for _, key := range strings.Split(path, ".") {
		if i := strings.IndexByte(key, '['); i >= 0 {
			key = key[:i]
		}
		if mask.HoldsCredential(key) {
			return true
		}
	}
	return false
}

// failureKindOf tells a template that could not be evaluated from an error
// of the action itself.
func failureKindOf(err error) string {
	var tErr *expr.TemplateError
	var fErr *expr.FieldError
	if errors.As(err, &tErr) || errors.As(err, &fErr) {
		return FailureTemplate
	}
	return FailureAction
}

// unwrapJoined returns the errors joined into err, or err alone.
func unwrapJoined(err error) []error {
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		return joined.Unwrap()
	}
	return []error{err}
}

// saveOutputs evaluates and saves step outputs to JobContext
func (st *Step) saveOutputs(jCtx *JobContext) {
	if len(st.Outputs) == 0 || st.ID == "" {
		return // No outputs to save or no ID (should be caught by validation)
	}

	// Evaluate each output expression
	outputs := make(map[string]any)
	for outputName, outputExpr := range st.Outputs {
		result, err := st.Expr.Eval(outputExpr, st.ctx)
		if err != nil {
			jCtx.Printer.PrintError("output '%s' evaluation error: %v", outputName, err)
			continue // Skip this output but continue with others
		}
		outputs[outputName] = result
	}

	// Save outputs to the unified Outputs structure
	if jCtx.Outputs != nil {
		// A name taken before is a warning; the step's value is still kept.
		// A step id taken by an output name is an error: the step's outputs
		// cannot be read through it.
		if err := jCtx.Outputs.Set(st.ID, outputs); err != nil {
			for _, e := range unwrapJoined(err) {
				var taken *NameTakenError
				if errors.As(e, &taken) {
					jCtx.Printer.LogWarn("%v", taken)
				} else {
					jCtx.Printer.PrintError("Output conflict: %v", e)
				}
			}
		}
	}

	if jCtx.Verbose {
		jCtx.Printer.LogDebug("Step '%s' outputs saved: %v", st.ID, outputs)
	}
}
