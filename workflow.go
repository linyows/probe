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

	ctx := w.newContext(c, vars)
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

func (w *Workflow) newContext(c Config, vars map[string]any) JobContext {
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

type TestContext struct {
	Vars map[string]any   `expr:"vars"`
	Logs []map[string]any `expr:"steps"`
	Res  map[string]any   `expr:"res"`
	Req  map[string]any   `expr:"req"`
}

type Repeat struct {
	Count    int `yaml:"count",validate:"required,gte=0,lt=100"`
	Interval int `yaml:"interval,validate:"gte=0,lt=600"`
}

type Step struct {
	Name string         `yaml:"name"`
	Uses string         `yaml:"uses" validate:"required"`
	With map[string]any `yaml:"with"`
	Test string         `yaml:"test"`
	Echo string         `yaml:"echo"`
	Vars map[string]any `yaml:"vars"`
	log  map[string]any
	err  error
	ctx  TestContext
}

type Job struct {
	Name     string  `yaml:"name",validate:"required"`
	Steps    []Step  `yaml:"steps",validate:"required"`
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

	for i, st := range j.Steps {
		if st.Name == "" {
			st.Name = "Unknown Step"
		}

		expW := expr.EvalTemplateMap(st.With, ctx)
		ret, err := RunActions(st.Uses, []string{}, expW, j.ctx.Config.Verbose)
		if err != nil {
			st.err = err
			continue
		}

		// parse json and sets
		req, okreq := ret["req"].(map[string]any)
		res, okres := ret["res"].(map[string]any)
		if okres {
			body, okbody := res["body"].(string)
			if okbody && isJSON(body) {
				res["rawbody"] = body
				res["body"] = mustMarshalJSON(body)
			}
		}

		// set log and logs
		st.log = ret
		ctx.Logs = append(ctx.Logs, st.log)

		output := ""
		st.ctx = st.SetCtx(ctx, req, res)

		if j.ctx.Config.Verbose && okreq && okres {
			showVerbose(i, st.Name, req, res)
			if st.Test == "" {
				continue
			}

			input := st.Test

			exprOut, err := expr.Eval(input, st.ctx)
			if err != nil {
				fmt.Printf("%s: %s\nInput: %s\n", color.RedString("Test Error"), err, input)
				j.ctx.SetFailed()
			} else {
				boolOutput, boolOk := exprOut.(bool)
				if boolOk {
					boolResultStr := color.GreenString("Success")
					if !boolOutput {
						boolResultStr = color.RedString("Failure")
						j.ctx.SetFailed()
					}
					fmt.Printf("Test: %s (input: %s, env: %#v)\n", boolResultStr, input, st.ctx)
				} else {
					fmt.Printf("Test: `%s` = %s\n", st.Test, exprOut)
					j.ctx.SetFailed()
				}
			}

			// Echo
			if st.Echo != "" {
				exprOut, err := expr.Eval(st.Echo, st.ctx)
				if err != nil {
					fmt.Printf("%s: %#v (input: %s)\n", color.RedString("Echo Error"), err, st.Echo)
				} else {
					fmt.Printf("Echo: %s\n", exprOut)
				}
			}

			fmt.Println("- - -")
			continue

		} else if j.ctx.Config.Verbose {
			fmt.Print("sorry, request or response is nil")
		}

		// Output format here:
		//   1. ✔︎ Step name
		num := color.HiBlackString(fmt.Sprintf("%2d.", i))
		output = fmt.Sprintf("%s %%s %s", num, st.Name)

		if st.Test != "" {
			exprOut, err := expr.Eval(st.Test, st.ctx)
			if err != nil {
				output = fmt.Sprintf(output+"\n", "-")
				output += fmt.Sprintf("Test\nerror: %#v\n", err)
				j.ctx.SetFailed()
			} else {
				boolOutput, boolOk := exprOut.(bool)
				if boolOk {
					boolResultStr := color.GreenString("✔︎ ")
					if !boolOutput {
						boolResultStr = color.RedString("✘ ")
						j.ctx.SetFailed()
					}
					output = fmt.Sprintf(output+"\n", boolResultStr)
					if !boolOutput {
						// 7 spaces
						output += fmt.Sprintf("       request: %#v\n", req)
						output += fmt.Sprintf("       response: %#v\n", res)
					}
				} else {
					output = fmt.Sprintf(output+"\n", "-")
					output += fmt.Sprintf("Test: `%s` = %s\n", st.Test, exprOut)
					j.ctx.SetFailed()
				}
			}
		} else {
			output = fmt.Sprintf(output+"\n", color.BlueString("▲ "))
		}

		fmt.Print(output)

		// Echo
		if st.Echo != "" {
			exprOut, err := expr.Eval(st.Echo, st.ctx)
			if err != nil {
				fmt.Printf("Echo\nerror: %#v\n", err)
			} else {
				// 7 spaces
				fmt.Printf("       %s\n", exprOut)
			}
		}
	}

	return j.ctx.Failed
}

func (s Step) SetCtx(j JobContext, req, res map[string]any) TestContext {
	return TestContext{
		Vars: MergeMaps(j.Vars, s.Vars),
		Logs: j.Logs,
		Req:  req,
		Res:  res,
	}
}

func showVerbose(i int, name string, req, res map[string]any) {
	fmt.Printf("--- Step %d: %s\nRequest:\n", i, name)

	for k, v := range req {
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

	for k, v := range res {
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
}
