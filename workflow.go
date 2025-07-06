package probe

import (
	"fmt"
	"sync"
	"time"

	"github.com/fatih/color"
)

type Workflow struct {
	Name       string         `yaml:"name",validate:"required"`
	Jobs       []Job          `yaml:"jobs",validate:"required"`
	Vars       map[string]any `yaml:"vars"`
	exitStatus int
	env        map[string]string
}

func (w *Workflow) SetExitStatus(isErr bool) {
	if isErr {
		w.exitStatus = 1
	}
}

func (w *Workflow) Start(c Config) error {
	vars, err := w.evalVars()
	if err != nil {
		return err
	}

	ctx := w.newJobContext(c, vars)
	var wg sync.WaitGroup

	for _, job := range w.Jobs {
		// No repeat
		if job.Repeat == nil {
			wg.Add(1)
			go func() {
				defer wg.Done()
				w.SetExitStatus(job.Start(ctx))
			}()
			continue
		}

		// Repeat
		for i := 0; i < job.Repeat.Count; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				w.SetExitStatus(job.Start(ctx))
			}()
			time.Sleep(time.Duration(job.Repeat.Interval) * time.Second)
		}
	}

	wg.Wait()

	return nil
}

func (w *Workflow) Env() map[string]string {
	if len(w.env) == 0 {
		w.env = EnvMap()
	}
	return w.env
}

func (w *Workflow) evalVars() (map[string]any, error) {
	env := StrmapToAnymap(w.Env())
	vars := make(map[string]any)

	expr := &Expr{}
	for k, v := range w.Vars {
		if mapV, ok := v.(map[string]any); ok {
			vars[k] = expr.EvalTemplateMap(mapV, env)
		} else if strV, ok2 := v.(string); ok2 {
			output, err := expr.EvalTemplate(strV, env)
			if err != nil {
				return vars, err
			}
			vars[k] = output
		}
	}

	return vars, nil
}

func (w *Workflow) newJobContext(c Config, vars map[string]any) JobContext {
	return JobContext{
		Vars:   vars,
		Logs:   []map[string]any{},
		Config: c,
	}
}

type JobContext struct {
	Vars map[string]any   `expr:"vars"`
	Logs []map[string]any `expr:"steps"`
	Config
	Failed bool
}

func (j *JobContext) SetFailed() {
	j.Failed = true
}

type StepContext struct {
	Vars map[string]any   `expr:"vars"`
	Logs []map[string]any `expr:"steps"`
	Res  map[string]any   `expr:"res"`
	Req  map[string]any   `expr:"req"`
	RT   string           `expr:"rt"`
}

type Repeat struct {
	Count    int `yaml:"count",validate:"required,gte=0,lt=100"`
	Interval int `yaml:"interval,validate:"gte=0,lt=600"`
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

type Job struct {
	Name     string  `yaml:"name",validate:"required"`
	Steps    []*Step `yaml:"steps",validate:"required"`
	Repeat   *Repeat `yaml:"repeat"`
	Defaults any     `yaml:"defaults"`
	ctx      *JobContext
}

func (j *Job) Start(ctx JobContext) bool {
	j.ctx = &ctx
	expr := &Expr{}

	if j.Name == "" {
		j.Name = "Unknown Job"
	}
	name, err := expr.EvalTemplate(j.Name, ctx)
	if err != nil {
		fmt.Printf("Expr error(job name): %#v\n", err)
	} else {
		fmt.Printf("%s\n", name)
	}

	var idx = 0
	for _, st := range j.Steps {
		st.expr = expr
		if len(st.Iter) == 0 {
			st.idx = idx
			idx += 1
			st.SetCtx(ctx, nil)
			st.Do(&ctx)
			continue
		}
		// NOTE: Split JobContext to ExprEnv
		for _, vars := range st.Iter {
			st.idx = idx
			idx += 1
			st.SetCtx(ctx, vars)
			st.Do(&ctx)
		}
	}

	return j.ctx.Failed
}

func (st *Step) Do(jCtx *JobContext) {
	if st.Name == "" {
		st.Name = "Unknown Step"
	}
	name, err := st.expr.EvalTemplate(st.Name, st.ctx)
	if err != nil {
		fmt.Printf("Expr error(step name): %#v\n", err)
	}

	expW := st.expr.EvalTemplateMap(st.With, st.ctx)
	ret, err := RunActions(st.Uses, []string{}, expW, jCtx.Config.Verbose)
	if err != nil {
		st.err = err
		fmt.Printf("%s \"%s\" in %s-action -- %s\n", color.RedString("Error"), name, st.Uses, err)
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
			fmt.Print("sorry, request or response is nil")
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
		fmt.Println("- - -")
		return
	}

	// Output format here:
	//   1. ✔︎ Step name
	num := color.HiBlackString(fmt.Sprintf("%2d.", st.idx))
	ps := ""
	if jCtx.Config.RT && okrt && st.ctx.RT != "" {
		ps = color.HiBlackString(fmt.Sprintf(" (%s)", st.ctx.RT))
	}
	output := fmt.Sprintf("%s %%s %s%s", num, name, ps)
	if st.Test != "" {
		str, ok := st.DoTest()
		if ok {
			output = fmt.Sprintf(output+"\n", color.GreenString("✔︎ "))
		} else {
			output = fmt.Sprintf(output+"\n"+str+"\n", color.RedString("✘ "))
			jCtx.SetFailed()
		}
	} else {
		output = fmt.Sprintf(output+"\n", color.BlueString("▲ "))
	}
	fmt.Print(output)

	if st.Echo != "" {
		st.DoEcho()
	}
}

func (st *Step) DoTestWithSequentialPrint() bool {
	exprOut, err := st.expr.Eval(st.Test, st.ctx)
	if err != nil {
		fmt.Printf("%s: %s\nInput: %s\n", color.RedString("Test Error"), err, st.Test)
		return false
	}

	boolOutput, boolOk := exprOut.(bool)
	if !boolOk {
		fmt.Printf("Test: `%s` = %s\n", st.Test, exprOut)
		return false
	}

	boolResultStr := color.GreenString("Success")
	if !boolOutput {
		boolResultStr = color.RedString("Failure")
	}
	fmt.Printf("Test: %s (input: %s, env: %#v)\n", boolResultStr, st.Test, st.ctx)

	return boolOk
}

func (st *Step) DoEchoWithSequentialPrint() {
	exprOut, err := st.expr.Eval(st.Echo, st.ctx)
	if err != nil {
		fmt.Printf("%s: %#v (input: %s)\n", color.RedString("Echo Error"), err, st.Echo)
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

	fmt.Printf("RT: %s\n", color.BlueString(st.ctx.RT))
}
