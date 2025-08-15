package probe

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
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
	err          error
	ctx          StepContext
	Idx          int          `yaml:"-"`
	Expr         *Expr        `yaml:"-"`
	actionRunner ActionRunner `yaml:"-"`
}

func (st *Step) Do(jCtx *JobContext) {

	// 1. Preparation phase: validation, wait, skip check
	name, shouldContinue := st.prepare(jCtx)
	if !shouldContinue {
		return
	}

	// 2. Action execution phase
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

	jCtx.Printer.AddSpinnerSuffix(st.Name)

	// Evaluate step name
	name, err := st.Expr.EvalTemplate(st.Name, st.ctx)
	if err != nil {
		jCtx.Printer.PrintError("step name evaluation error: %v", err)
		return "", false
	}

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
	expW := st.Expr.EvalTemplateMap(st.With, st.ctx)

	runner := st.actionRunner
	if runner == nil {
		runner = &PluginActionRunner{} // Default to plugin execution
	}

	// If no retry configuration, execute once
	if st.Retry == nil {
		return st.executeSingleAction(runner, expW, jCtx)
	}

	// Execute with retry logic
	return st.executeActionWithRetry(runner, expW, jCtx, name)
}

// executeSingleAction executes action once without retry
func (st *Step) executeSingleAction(runner ActionRunner, expW map[string]any, jCtx *JobContext) (map[string]any, error) {
	ret, err := runner.RunActions(st.Uses, []string{}, expW, jCtx.Verbose)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// executeActionWithRetry executes action with retry logic until status == 0
func (st *Step) executeActionWithRetry(runner ActionRunner, expW map[string]any, jCtx *JobContext, name string) (map[string]any, error) {
	retry := st.Retry

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

		result, err := st.executeSingleAction(runner, expW, jCtx)
		lastResult = result
		lastErr = err

		// Check for success (status == 0)
		if err == nil {
			if status, ok := result["status"]; ok {
				if statusInt, isInt := status.(int); isInt && statusInt == int(ExitStatusSuccess) {
					if jCtx.Verbose {
						jCtx.Printer.LogDebug("Action succeeded on attempt %d", attempt)
					}
					return result, nil
				}
			}
		}

		// If not the last attempt, wait for interval
		if attempt < retry.MaxAttempts {
			if jCtx.Verbose {
				jCtx.Printer.LogDebug("Action failed (attempt %d), retrying after %v", attempt, retry.Interval.Duration)
			}
			time.Sleep(retry.Interval.Duration)
		}
	}

	// All attempts failed
	if jCtx.Verbose {
		jCtx.Printer.LogDebug("All retry attempts failed (%d attempts)", retry.MaxAttempts)
	}
	return lastResult, lastErr
}

// handleActionError handles action execution errors
func (st *Step) handleActionError(err error, name string, jCtx *JobContext) {
	actionErr := NewActionError("step_execute", "action execution failed", err).
		WithContext("step_name", name).
		WithContext("action_type", st.Uses)
	st.err = actionErr
	jCtx.Printer.PrintError("Action execution failed: %v", actionErr)
	jCtx.SetFailed()

	// Create and add step result for failed action execution
	if jCtx.Verbose {
		jCtx.Printer.PrintRequestResponse(st.Idx, name, st.ctx.Req, st.ctx.Res, st.ctx.RT.Duration)
	}

	// Handle repeat execution
	if jCtx.IsRepeating {
		st.handleRepeatExecution(jCtx, name)
		return
	}

	// Standard execution: create result for failed step
	stepResult := st.createFailedStepResult(name, jCtx, nil)

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
		if okbody && isJSON(body) {
			res["rawbody"] = body
			res["body"] = mustMarshalJSON(body)
		}
	}

	// Update context with status
	st.updateCtx(nil, req, res, rt, status)
}

// finalize handles the final phase: test, echo, output save, and result creation
func (st *Step) finalize(name string, actionResult map[string]any, jCtx *JobContext) {
	if jCtx.Config.Verbose {
		jCtx.Printer.PrintRequestResponse(st.Idx, name, st.ctx.Req, st.ctx.Res, st.ctx.RT.Duration)
	}

	// Handle repeat execution
	if jCtx.IsRepeating {
		st.handleRepeatExecution(jCtx, name)
		return
	}

	// Standard execution: save outputs and create result
	st.saveOutputs(jCtx)
	stepResult := st.createStepResult(name, jCtx, nil)

	// Add step result to workflow buffer
	if jCtx.Result != nil {
		jCtx.Result.AddStepResult(jCtx.CurrentJobID, stepResult)
	}

	if jCtx.Verbose {
		jCtx.Printer.PrintSeparator()
	}
}

// createStepResult creates a StepResult from step execution
func (st *Step) createStepResult(name string, jCtx *JobContext, repeatCounter *StepRepeatCounter) StepResult {
	result := StepResult{
		Index:         st.Idx,
		Name:          name,
		HasTest:       st.Test != "",
		RT:            "",
		WaitTime:      st.getWaitTimeForDisplay(),
		RepeatCounter: repeatCounter,
	}

	if jCtx.RT && st.ctx.RT.Duration != "" {
		result.RT = st.ctx.RT.Duration
		result.RTSec = st.ctx.RT.Sec
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
			jCtx.SetFailed()
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

func (st *Step) handleRepeatExecution(jCtx *JobContext, name string) {

	// Initialize counter if first execution
	counter, exists := jCtx.StepCounters[st.Idx]
	if !exists {
		counter = StepRepeatCounter{
			Name: name,
		}
	}

	// Execute test and update counter
	hasTest := st.Test != ""
	testResult := true
	if hasTest {
		_, testResult = st.DoTest(jCtx.Printer)
		if !testResult {
			jCtx.SetFailed()
		}

		if testResult {
			counter.SuccessCount++
		} else {
			counter.FailureCount++
		}
	} else {
		// No test - just count as executed
		counter.SuccessCount++
	}
	counter.LastResult = testResult

	// Store updated counter
	jCtx.StepCounters[st.Idx] = counter

	// Display on first execution and final execution only
	isFinalExecution := jCtx.RepeatCurrent == jCtx.RepeatTotal

	if isFinalExecution {
		// Create StepResult with repeat counter for final execution
		stepResult := st.createStepResult(name, jCtx, &counter)

		// Add step result to workflow buffer
		if jCtx.Result != nil {
			jCtx.Result.AddStepResult(jCtx.CurrentJobID, stepResult)
		}
	}

	// Handle echo output
	if st.Echo != "" {
		st.DoEcho(jCtx)
	}
}

func (st *Step) DoTest(printer *Printer) (string, bool) {
	exprOut, err := st.Expr.Eval(st.Test, st.ctx)
	if err != nil {
		return printer.generateTestError(st.Test, err), false
	}

	boolOutput, boolOk := exprOut.(bool)
	if !boolOk {
		return printer.generateTestTypeMismatch(st.Test, exprOut), false
	}

	if printer.verbose {
		printer.PrintTestResult(boolOutput, st.Test, st.ctx)
	}

	if !boolOutput {
		return printer.generateTestFailure(st.Test, exprOut, st.ctx.Req, st.ctx.Res), false
	}

	return "", true
}

func (st *Step) DoEcho(jCtx *JobContext) {
	exprOut, err := st.Expr.EvalTemplate(st.Echo, st.ctx)
	if err != nil {
		jCtx.Printer.LogError("Echo evaluation failed: %#v (input: %s)", err, st.Echo)
	} else {
		jCtx.Printer.PrintEchoContent(exprOut)
	}
}

func (st *Step) SetCtx(j JobContext, override map[string]any) {
	// Use outputs from the unified Outputs structure
	var outputs map[string]any
	if j.Outputs != nil {
		outputs = j.Outputs.GetAll()
	}

	// Create context for step vars evaluation
	evalCtx := StepContext{
		Vars:    j.Vars,
		Outputs: outputs,
	}

	// Evaluate step-level vars with access to outputs
	evaluatedStepVars := make(map[string]any)
	if len(st.Vars) > 0 {
		expr := &Expr{}
		for k, v := range st.Vars {
			if mapV, ok := v.(map[string]any); ok {
				evaluatedStepVars[k] = expr.EvalTemplateMap(mapV, evalCtx)
			} else if strV, ok2 := v.(string); ok2 {
				output, err := expr.EvalTemplate(strV, evalCtx)
				if err != nil {
					// If evaluation fails, keep original value
					evaluatedStepVars[k] = v
				} else {
					evaluatedStepVars[k] = output
				}
			} else {
				evaluatedStepVars[k] = v
			}
		}
	}

	// Merge workflow vars with evaluated step vars
	vers := MergeMaps(j.Vars, evaluatedStepVars)
	if override != nil {
		vers = MergeMaps(vers, override)
	}

	st.ctx = StepContext{
		Vars:    vers,
		Outputs: outputs,
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

	duration, err := st.parseWaitDuration(st.Wait)
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

	duration, err := st.parseWaitDuration(st.Wait)
	if err != nil {
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
	if jCtx.Config.Verbose {
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
	stepResult := st.createSkippedStepResult(name, jCtx, nil)

	// Add step result to workflow buffer
	if jCtx.Result != nil {
		jCtx.Result.AddStepResult(jCtx.CurrentJobID, stepResult)
	}
}

// handleSkipRepeatExecution handles skipped step in repeat mode
func (st *Step) handleSkipRepeatExecution(jCtx *JobContext, name string) {
	// Initialize counter if first execution
	counter, exists := jCtx.StepCounters[st.Idx]
	if !exists {
		counter = StepRepeatCounter{
			Name: name,
		}
	}

	// Count as successful (skipped is not a failure)
	counter.SuccessCount++
	counter.LastResult = true

	// Store updated counter
	jCtx.StepCounters[st.Idx] = counter

	// Display on first execution and final execution only
	isFinalExecution := jCtx.RepeatCurrent == jCtx.RepeatTotal

	if isFinalExecution {
		stepResult := st.createSkippedStepResult(name, jCtx, &counter)

		// Add step result to workflow buffer
		if jCtx.Result != nil {
			jCtx.Result.AddStepResult(jCtx.CurrentJobID, stepResult)
		}
	}
}

// createSkippedStepResult creates a StepResult for a skipped step
func (st *Step) createSkippedStepResult(name string, jCtx *JobContext, repeatCounter *StepRepeatCounter) StepResult {
	return StepResult{
		Index:         st.Idx,
		Name:          name + " (SKIPPED)",
		Status:        StatusSkipped,
		RT:            "",
		WaitTime:      st.getWaitTimeForDisplay(),
		HasTest:       false,
		RepeatCounter: repeatCounter,
	}
}

// createFailedStepResult creates a StepResult for a failed step
func (st *Step) createFailedStepResult(name string, jCtx *JobContext, repeatCounter *StepRepeatCounter) StepResult {
	result := StepResult{
		Index:         st.Idx,
		Name:          name,
		Status:        StatusError,
		RT:            "",
		WaitTime:      st.getWaitTimeForDisplay(),
		HasTest:       st.Test != "",
		RepeatCounter: repeatCounter,
	}

	if jCtx.RT && st.ctx.RT.Duration != "" {
		result.RT = st.ctx.RT.Duration
		result.RTSec = st.ctx.RT.Sec
	}
	if v, ok := st.ctx.Res["report"]; ok {
		if report, sok := v.(string); sok {
			result.Report = report
		}
	}

	// Include error information if available
	if st.err != nil {
		result.TestOutput = st.err.Error()
	}

	return result
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
		if err := jCtx.Outputs.Set(st.ID, outputs); err != nil {
			jCtx.Printer.PrintError("Output conflict warning: %v", err)
		}
	}

	if jCtx.Config.Verbose {
		jCtx.Printer.LogDebug("Step '%s' outputs saved: %v", st.ID, outputs)
	}
}
