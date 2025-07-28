package probe

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

type StepContext struct {
	Vars    map[string]any            `expr:"vars"`
	Logs    []map[string]any          `expr:"steps"`
	Res     map[string]any            `expr:"res"`
	Req     map[string]any            `expr:"req"`
	RT      string                    `expr:"rt"`
	Outputs map[string]map[string]any `expr:"outputs"`
}

// StepRepeatCounter tracks the execution results of repeated steps
type StepRepeatCounter struct {
	SuccessCount int
	FailureCount int
	Name         string
	LastResult   bool
}

type Step struct {
	Name    string            `yaml:"name"`
	ID      string            `yaml:"id,omitempty"`
	Uses    string            `yaml:"uses" validate:"required"`
	With    map[string]any    `yaml:"with"`
	Test    string            `yaml:"test"`
	Echo    string            `yaml:"echo"`
	Vars    map[string]any    `yaml:"vars"`
	Iter    []map[string]any  `yaml:"iter"`
	Wait    string            `yaml:"wait,omitempty"`
	SkipIf  string            `yaml:"skipif,omitempty"`
	Outputs map[string]string `yaml:"outputs,omitempty"`
	err     error
	ctx     StepContext
	idx     int
	expr    *Expr
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

// prepare handles step preparation: validation, wait, and skip check
// Returns (stepName, shouldContinue)
func (st *Step) prepare(jCtx *JobContext) (string, bool) {
	// Set default name if empty
	if st.Name == "" {
		st.Name = "Unknown Step"
	}

	jCtx.Printer.AddSpinnerSuffix(st.Name)

	// Handle wait before step execution
	st.handleWait(jCtx)

	// Evaluate step name
	name, err := st.expr.EvalTemplate(st.Name, st.ctx)
	if err != nil {
		jCtx.Printer.PrintError("step name evaluation error: %v", err)
		return "", false
	}

	// Check if step should be skipped
	if st.shouldSkip(jCtx) {
		st.handleSkip(name, jCtx)
		return name, false
	}

	return name, true
}

// executeAction executes the step action and returns the result
func (st *Step) executeAction(name string, jCtx *JobContext) (map[string]any, error) {
	expW := st.expr.EvalTemplateMap(st.With, st.ctx)
	ret, err := RunActions(st.Uses, []string{}, expW, jCtx.Config.Verbose)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// handleActionError handles action execution errors
func (st *Step) handleActionError(err error, name string, jCtx *JobContext) {
	actionErr := NewActionError("step_execute", "action execution failed", err).
		WithContext("step_name", name).
		WithContext("action_type", st.Uses)
	st.err = actionErr
	jCtx.Printer.PrintError("Action execution failed: %v", actionErr)
	jCtx.SetFailed()
}

// processActionResult processes the action result and updates context
func (st *Step) processActionResult(actionResult map[string]any, jCtx *JobContext) {
	// Parse and process JSON response
	req, _ := actionResult["req"].(map[string]any)
	res, okres := actionResult["res"].(map[string]any)
	rt, _ := actionResult["rt"].(string)

	if okres {
		body, okbody := res["body"].(string)
		if okbody && isJSON(body) {
			res["rawbody"] = body
			res["body"] = mustMarshalJSON(body)
		}
	}

	// Update logs and context
	jCtx.Logs = append(jCtx.Logs, actionResult)
	st.updateCtx(jCtx.Logs, req, res, rt)
}

// finalize handles the final phase: test, echo, output save, and result creation
func (st *Step) finalize(name string, actionResult map[string]any, jCtx *JobContext) {

	// Extract commonly used values
	req, okreq := actionResult["req"].(map[string]any)
	res, okres := actionResult["res"].(map[string]any)
	rt, okrt := actionResult["rt"].(string)

	// Handle verbose mode
	if jCtx.Config.Verbose {
		st.handleVerboseMode(name, req, res, okreq, okres, jCtx)
		return
	}

	// Handle repeat execution
	if jCtx.IsRepeating {
		st.handleRepeatExecution(jCtx, name, rt, okrt)
		return
	}

	// Standard execution: save outputs and create result
	st.saveOutputs(jCtx)
	stepResult := st.createStepResult(name, rt, okrt, jCtx, nil)

	// Add step result to workflow buffer
	if jCtx.WorkflowBuffer != nil {
		jCtx.WorkflowBuffer.AddStepResult(jCtx.CurrentJobID, stepResult)
	}
}

// handleVerboseMode handles verbose execution mode
func (st *Step) handleVerboseMode(name string, req, res map[string]any, okreq, okres bool, jCtx *JobContext) {
	if !okreq || !okres {
		jCtx.Printer.PrintVerbose("sorry, request or response is nil")
		jCtx.SetFailed()
		return
	}

	st.ShowRequestResponse(name, jCtx)

	if st.Test != "" {
		if ok := st.DoTestWithSequentialPrint(jCtx); !ok {
			jCtx.SetFailed()
		}
	}

	if st.Echo != "" {
		st.DoEchoWithSequentialPrint(jCtx)
	}

	// Save step outputs even in verbose mode
	st.saveOutputs(jCtx)

	jCtx.Printer.PrintSeparator()
}

// createStepResult creates a StepResult from step execution
func (st *Step) createStepResult(name, rt string, okrt bool, jCtx *JobContext, repeatCounter *StepRepeatCounter) StepResult {
	result := StepResult{
		Index:         st.idx,
		Name:          name,
		HasTest:       st.Test != "",
		RT:            "",
		WaitTime:      st.getWaitTimeForDisplay(),
		RepeatCounter: repeatCounter,
	}

	if jCtx.Config.RT && okrt && rt != "" {
		result.RT = rt
	}

	if st.Test != "" {
		testOutput, ok := st.DoTest()
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
		result.EchoOutput = st.getEchoOutput()
	}

	return result
}

// getEchoOutput returns the echo output as string
func (st *Step) getEchoOutput() string {
	exprOut, err := st.expr.Eval(st.Echo, st.ctx)
	if err != nil {
		return fmt.Sprintf("Echo\nerror: %#v\n", err)
	}
	return fmt.Sprintf("       %s\n", exprOut)
}

func (st *Step) handleRepeatExecution(jCtx *JobContext, name, rt string, okrt bool) {

	// Initialize counter if first execution
	counter, exists := jCtx.StepCounters[st.idx]
	if !exists {
		counter = StepRepeatCounter{
			Name: name,
		}
	}

	// Execute test and update counter
	hasTest := st.Test != ""
	testResult := true
	if hasTest {
		_, testResult = st.DoTest()
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
	jCtx.StepCounters[st.idx] = counter

	// Display on first execution and final execution only
	isFinalExecution := jCtx.RepeatCurrent == jCtx.RepeatTotal

	if isFinalExecution {
		// Create StepResult with repeat counter for final execution
		stepResult := st.createStepResult(name, rt, okrt, jCtx, &counter)

		// Add step result to workflow buffer
		if jCtx.WorkflowBuffer != nil {
			jCtx.WorkflowBuffer.AddStepResult(jCtx.CurrentJobID, stepResult)
		}
	}

	// Handle echo output
	if st.Echo != "" {
		st.DoEcho(jCtx)
	}
}

func (st *Step) DoTestWithSequentialPrint(jCtx *JobContext) bool {
	exprOut, err := st.expr.Eval(st.Test, st.ctx)
	if err != nil {
		jCtx.Printer.LogError("Test Error: %s", err)
		jCtx.Printer.LogError("Input: %s", st.Test)
		return false
	}

	boolOutput, boolOk := exprOut.(bool)
	if !boolOk {
		jCtx.Printer.LogDebug("Test: `%s` = %s", st.Test, exprOut)
		return false
	}

	boolResultStr := colorSuccess().Sprintf("Success")
	if !boolOutput {
		boolResultStr = colorError().Sprintf("Failure")
	}
	jCtx.Printer.LogDebug("Test: %s (input: %s, env: %#v)", boolResultStr, st.Test, st.ctx)

	return boolOutput
}

func (st *Step) DoEchoWithSequentialPrint(jCtx *JobContext) {
	exprOut, err := st.expr.Eval(st.Echo, st.ctx)
	if err != nil {
		jCtx.Printer.LogError("Echo Error: %#v (input: %s)", err, st.Echo)
	} else {
		jCtx.Printer.LogDebug("Echo: %s", exprOut)
	}
}

func (st *Step) DoTest() (string, bool) {
	exprOut, err := st.expr.Eval(st.Test, st.ctx)
	if err != nil {
		return fmt.Sprintf("Test\nerror: %#v\n", err), false
	}

	boolOutput, boolOk := exprOut.(bool)
	if !boolOk {
		return fmt.Sprintf("Test: `%s` = %s\n", st.Test, exprOut), false
	}

	if !boolOutput {
		// 7 spaces
		output := fmt.Sprintf("       request: %#v\n", st.ctx.Req)
		output += fmt.Sprintf("       response: %#v\n", st.ctx.Res)
		return output, false
	}

	return "", true
}

func (st *Step) DoEcho(jCtx *JobContext) {
	exprOut, err := st.expr.Eval(st.Echo, st.ctx)
	if err != nil {
		jCtx.Printer.LogError("Echo evaluation failed: %#v", err)
	} else {
		// 7 spaces
		jCtx.Printer.LogDebug("       %s", exprOut)
	}
}

func (st *Step) SetCtx(j JobContext, override map[string]any) {
	vers := MergeMaps(j.Vars, st.Vars)
	if override != nil {
		vers = MergeMaps(vers, override)
	}

	// Use outputs from the unified Outputs structure
	var outputs map[string]map[string]any
	if j.Outputs != nil {
		outputs = j.Outputs.GetAll()
	}

	st.ctx = StepContext{
		Vars:    vers,
		Logs:    j.Logs,
		Outputs: outputs,
	}
}

func (st *Step) updateCtx(logs []map[string]any, req, res map[string]any, rt string) {
	st.ctx.Logs = logs
	st.ctx.Req = req
	st.ctx.Res = res
	st.ctx.RT = rt
}

func (st *Step) ShowRequestResponse(name string, jCtx *JobContext) {
	jCtx.Printer.LogDebug("--- Step %d: %s", st.idx, name)
	jCtx.Printer.LogDebug("Request:")
	st.printMapData(st.ctx.Req, jCtx)

	jCtx.Printer.LogDebug("Response:")
	st.printMapData(st.ctx.Res, jCtx)

	jCtx.Printer.LogDebug("RT: %s", colorInfo().Sprintf("%s", st.ctx.RT))
}

// printMapData prints map data with proper formatting for nested structures
func (st *Step) printMapData(data map[string]any, jCtx *JobContext) {
	for k, v := range data {
		if nested, ok := v.(map[string]any); ok {
			st.printNestedMap(k, nested, jCtx)
		} else {
			jCtx.Printer.LogDebug("  %s: %#v", k, v)
		}
	}
}

// printNestedMap prints nested map data with indentation
func (st *Step) printNestedMap(key string, nested map[string]any, jCtx *JobContext) {
	jCtx.Printer.LogDebug("  %s:", key)
	for kk, vv := range nested {
		jCtx.Printer.LogDebug("    %s: %#v", kk, vv)
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

	return
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

	result, err := st.expr.Eval(st.SkipIf, st.ctx)
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
		jCtx.Printer.LogDebug("--- Step %d: %s (SKIPPED)", st.idx, name)
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
	if jCtx.WorkflowBuffer != nil {
		jCtx.WorkflowBuffer.AddStepResult(jCtx.CurrentJobID, stepResult)
	}
}

// handleSkipRepeatExecution handles skipped step in repeat mode
func (st *Step) handleSkipRepeatExecution(jCtx *JobContext, name string) {
	// Initialize counter if first execution
	counter, exists := jCtx.StepCounters[st.idx]
	if !exists {
		counter = StepRepeatCounter{
			Name: name,
		}
	}

	// Count as successful (skipped is not a failure)
	counter.SuccessCount++
	counter.LastResult = true

	// Store updated counter
	jCtx.StepCounters[st.idx] = counter

	// Display on first execution and final execution only
	isFinalExecution := jCtx.RepeatCurrent == jCtx.RepeatTotal

	if isFinalExecution {
		stepResult := st.createSkippedStepResult(name, jCtx, &counter)

		// Add step result to workflow buffer
		if jCtx.WorkflowBuffer != nil {
			jCtx.WorkflowBuffer.AddStepResult(jCtx.CurrentJobID, stepResult)
		}
	}
}

// createSkippedStepResult creates a StepResult for a skipped step
func (st *Step) createSkippedStepResult(name string, jCtx *JobContext, repeatCounter *StepRepeatCounter) StepResult {
	return StepResult{
		Index:         st.idx,
		Name:          name + " (SKIPPED)",
		Status:        StatusSkipped,
		RT:            "",
		WaitTime:      st.getWaitTimeForDisplay(),
		HasTest:       false,
		RepeatCounter: repeatCounter,
	}
}

// saveOutputs evaluates and saves step outputs to JobContext
func (st *Step) saveOutputs(jCtx *JobContext) {
	if len(st.Outputs) == 0 || st.ID == "" {
		return // No outputs to save or no ID (should be caught by validation)
	}

	// Evaluate each output expression
	outputs := make(map[string]any)
	for outputName, outputExpr := range st.Outputs {
		result, err := st.expr.Eval(outputExpr, st.ctx)
		if err != nil {
			jCtx.Printer.PrintError("output '%s' evaluation error: %v", outputName, err)
			continue // Skip this output but continue with others
		}
		outputs[outputName] = result
	}

	// Save outputs to the unified Outputs structure
	if jCtx.Outputs != nil {
		jCtx.Outputs.Set(st.ID, outputs)
	}

	if jCtx.Config.Verbose {
		jCtx.Printer.LogDebug("Step '%s' outputs saved: %v", st.ID, outputs)
	}
}
