package probe

import (
	"fmt"
	"sync"
	"time"

	"github.com/fatih/color"
)

type Workflow struct {
	Name string `yaml:"name",validate:"required"`
	Jobs []Job  `yaml:"jobs",validate:"required"`
}

func (w *Workflow) Start(c Config) {
	ctx := w.createContext(c)
	var wg sync.WaitGroup

	for _, job := range w.Jobs {
		// No repeat
		if job.Repeat == nil {
			wg.Add(1)
			go func() {
				defer wg.Done()
				job.Start(ctx)
			}()
			continue
		}

		// Repeat
		for i := 0; i < job.Repeat.Count; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				job.Start(ctx)
			}()
			time.Sleep(time.Duration(job.Repeat.Interval) * time.Second)
		}
	}

	wg.Wait()
}

func (w *Workflow) createContext(c Config) JobContext {
	return JobContext{
		Envs:   getEnvMap(),
		Logs:   []map[string]any{},
		Config: c,
	}
}

type JobContext struct {
	Envs map[string]string `expr:"env"`
	Logs []map[string]any  `expr:"steps"`
	Config
}

type TestContext struct {
	Envs map[string]string `expr:"env"`
	Logs []map[string]any  `expr:"steps"`
	Res  map[string]any    `expr:"res"`
	Req  map[string]any    `expr:"req"`
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
	log  map[string]any
	err  error
}

type Job struct {
	Name     string  `yaml:"name",validate:"required"`
	Steps    []Step  `yaml:"steps",validate:"required"`
	Repeat   *Repeat `yaml:"repeat"`
	Defaults any     `yaml:"defaults"`
	ctx      *JobContext
}

func (j *Job) Start(ctx JobContext) {
	j.ctx = &ctx
	if j.Name == "" {
		j.Name = "Unknown"
	}
	fmt.Printf("Job: %s\n", j.Name)

	for i, st := range j.Steps {
		if st.Name == "" {
			st.Name = "Unknown"
		}

		expW := EvaluateExprs(st.With, ctx)
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

		if j.ctx.Config.Verbose && okreq && okres {
			showVerbose(i, st.Name, req, res)
			if st.Test == "" {
				continue
			}
			exprOut, err := EvalExpr(st.Test, NewTestContext(ctx, req, res))
			if err != nil {
				fmt.Printf("%s: %#v\n", color.RedString("Test Error"), err)
			} else {
				boolOutput, boolOk := exprOut.(bool)
				if boolOk {
					boolResultStr := color.RedString("Failure")
					if boolOutput {
						boolResultStr = color.GreenString("Success")
					}
					fmt.Printf("Test: %s\n", boolResultStr)
				} else {
					fmt.Printf("Test: `%s` = %s\n", st.Test, exprOut)
				}
			}
			fmt.Println("- - -")
			continue

		} else if j.ctx.Config.Verbose {
			fmt.Print("sorry, request or response is nil")
		}

		// 1. Step name
		num := color.HiBlackString(fmt.Sprintf("%2d.", i))
		output = fmt.Sprintf("%s %%s %s", num, st.Name)

		if st.Test == "" {
			fmt.Printf(output+"\n", "-")
			continue
		}

		exprOut, err := EvalExpr(st.Test, NewTestContext(ctx, req, res))
		if err != nil {
			output = fmt.Sprintf(output+"\n", "-")
			output += fmt.Sprintf("Test\nerror: %#v\n", err)
		} else {
			boolOutput, boolOk := exprOut.(bool)
			if boolOk {
				boolResultStr := color.RedString("✖️")
				if boolOutput {
					boolResultStr = color.GreenString("✔︎ ")
				}
				output = fmt.Sprintf(output+"\n", boolResultStr)
				if !boolOutput {
					output += fmt.Sprintf("       request: %#v\n", req)
					output += fmt.Sprintf("       response: %#v\n", res)
				}
			} else {
				output = fmt.Sprintf(output+"\n", "-")
				output += fmt.Sprintf("Test: `%s` = %s\n", st.Test, exprOut)
			}
		}

		fmt.Print(output)
	}
}

func NewTestContext(j JobContext, req, res map[string]any) TestContext {
	return TestContext{
		Envs: j.Envs,
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
