package probe

import "fmt"

type StepContext struct {
	Vars map[string]any   `expr:"vars"`
	Logs []map[string]any `expr:"steps"`
	Res  map[string]any   `expr:"res"`
	Req  map[string]any   `expr:"req"`
	RT   string           `expr:"rt"`
}

type Step struct {
	Name string           `yaml:"name"`
	Uses string           `yaml:"uses" validate:"required"`
	With map[string]any   `yaml:"with"`
	Test string           `yaml:"test"`
	Echo string           `yaml:"echo"`
	Vars map[string]any   `yaml:"vars"`
	Iter []map[string]any `yaml:"iter"`
	err  error
	ctx  StepContext
	idx  int
	expr *Expr
}

func (st *Step) Do(jCtx *JobContext) {
	if st.Name == "" {
		st.Name = "Unknown Step"
	}
	name, err := st.expr.EvalTemplate(st.Name, st.ctx)
	if err != nil {
		jCtx.Output.PrintError("step name evaluation error: %v", err)
		return
	}

	expW := st.expr.EvalTemplateMap(st.With, st.ctx)
	ret, err := RunActions(st.Uses, []string{}, expW, jCtx.Config.Verbose)
	if err != nil {
		st.err = err
		jCtx.Output.PrintError("\"%s\" in %s-action -- %s", name, st.Uses, err)
		jCtx.SetFailed()
		return
	}

	// parse json and sets
	req, okreq := ret["req"].(map[string]any)
	res, okres := ret["res"].(map[string]any)
	rt, okrt := ret["rt"].(string)
	if okres {
		body, okbody := res["body"].(string)
		if okbody && isJSON(body) {
			res["rawbody"] = body
			res["body"] = mustMarshalJSON(body)
		}
	}

	// set log and logs
	jCtx.Logs = append(jCtx.Logs, ret)
	st.updateCtx(jCtx.Logs, req, res, rt)

	if jCtx.Config.Verbose {
		if !okreq || !okres {
			jCtx.Output.PrintVerbose("sorry, request or response is nil")
			jCtx.SetFailed()
			return
		}
		st.ShowRequestResponse(name)
		if st.Test != "" {
			if ok := st.DoTestWithSequentialPrint(); !ok {
				jCtx.SetFailed()
			}
		}
		if st.Echo != "" {
			st.DoEchoWithSequentialPrint()
		}
		jCtx.Output.PrintSeparator()
		return
	}

	// Handle repeat execution
	if jCtx.IsRepeating {
		st.handleRepeatExecution(jCtx, name, rt, okrt)
		return
	}

	// Create step result for output
	stepResult := st.createStepResult(name, rt, okrt, jCtx)
	jCtx.Output.PrintStepResult(stepResult)
}

// createStepResult creates a StepResult from step execution
func (st *Step) createStepResult(name, rt string, okrt bool, jCtx *JobContext) StepResult {
	result := StepResult{
		Index:   st.idx,
		Name:    name,
		HasTest: st.Test != "",
		RT:      "",
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
	totalCount := counter.SuccessCount + counter.FailureCount
	isFirstExecution := totalCount == 1
	isFinalExecution := jCtx.RepeatCurrent == jCtx.RepeatTotal

	if isFirstExecution {
		jCtx.Output.PrintStepRepeatStart(st.idx, name, jCtx.RepeatTotal)
	}
	
	if isFinalExecution {
		jCtx.Output.PrintStepRepeatResult(st.idx, counter, hasTest)
	}

	// Handle echo output
	if st.Echo != "" {
		st.DoEcho()
	}
}

func (st *Step) DoTestWithSequentialPrint() bool {
	exprOut, err := st.expr.Eval(st.Test, st.ctx)
	if err != nil {
		fmt.Printf("%s: %s\nInput: %s\n", colorError().Sprintf("Test Error"), err, st.Test)
		return false
	}

	boolOutput, boolOk := exprOut.(bool)
	if !boolOk {
		fmt.Printf("Test: `%s` = %s\n", st.Test, exprOut)
		return false
	}

	boolResultStr := colorSuccess().Sprintf("Success")
	if !boolOutput {
		boolResultStr = colorError().Sprintf("Failure")
	}
	fmt.Printf("Test: %s (input: %s, env: %#v)\n", boolResultStr, st.Test, st.ctx)

	return boolOutput
}

func (st *Step) DoEchoWithSequentialPrint() {
	exprOut, err := st.expr.Eval(st.Echo, st.ctx)
	if err != nil {
		fmt.Printf("%s: %#v (input: %s)\n", colorError().Sprintf("Echo Error"), err, st.Echo)
	} else {
		fmt.Printf("Echo: %s\n", exprOut)
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

func (st *Step) DoEcho() {
	exprOut, err := st.expr.Eval(st.Echo, st.ctx)
	if err != nil {
		fmt.Printf("Echo\nerror: %#v\n", err)
	} else {
		// 7 spaces
		fmt.Printf("       %s\n", exprOut)
	}
}

func (st *Step) SetCtx(j JobContext, override map[string]any) {
	vers := MergeMaps(j.Vars, st.Vars)
	if override != nil {
		vers = MergeMaps(vers, override)
	}
	st.ctx = StepContext{
		Vars: vers,
		Logs: j.Logs,
	}
}

func (st *Step) updateCtx(logs []map[string]any, req, res map[string]any, rt string) {
	st.ctx.Logs = logs
	st.ctx.Req = req
	st.ctx.Res = res
	st.ctx.RT = rt
}

func (st *Step) ShowRequestResponse(name string) {
	fmt.Printf("--- Step %d: %s\nRequest:\n", st.idx, name)

	for k, v := range st.ctx.Req {
		nested, ok := v.(map[string]any)
		if ok {
			fmt.Printf("  %s:\n", k)
			for kk, vv := range nested {
				fmt.Printf("    %s: %#v\n", kk, vv)
			}
		} else {
			fmt.Printf("  %s: %#v\n", k, v)
		}
	}
	fmt.Printf("Response:\n")

	for k, v := range st.ctx.Res {
		nested, ok := v.(map[string]any)
		if ok {
			fmt.Printf("  %s:\n", k)
			for kk, vv := range nested {
				fmt.Printf("    %s: %#v\n", kk, vv)
			}
		} else {
			fmt.Printf("  %s: %#v\n", k, v)
		}
	}

	fmt.Printf("RT: %s\n", colorWarning().Sprintf("%s", st.ctx.RT))
}
