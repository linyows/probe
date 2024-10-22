package probe

import (
	"fmt"
	"sync"
	"time"
)

type Workflow struct {
	Name string `yaml:"name",validate:"required"`
	Jobs []Job  `yaml:"jobs",validate:"required"`
}

func (w *Workflow) Start() {
	ctx := w.createContext()
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

func (w *Workflow) createContext() JobContext {
	return JobContext{
		Envs: getEnvMap(),
		Logs: []map[string]any{},
	}
}

type JobContext struct {
	Envs map[string]string `expr:"env"`
	Logs []map[string]any  `expr:"steps"`
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
	fmt.Printf("=== Job: %s\n", j.Name)

	for i, st := range j.Steps {
		if st.Name == "" {
			st.Name = "Unknown"
		}
		expW := EvaluateExprs(st.With, ctx)
		ret, err := RunActions(st.Uses, []string{}, expW)
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
		if okreq && okres {
			ShowVerbose(i, st.Name, req, res)
		} else {
			fmt.Printf("--- Step %d: %s\n%#v\n", i, st.Name, ret)
		}

		// set log and logs
		st.log = ret
		ctx.Logs = append(ctx.Logs, st.log)

		if st.Test != "" {
			output, err := EvalExpr(st.Test, NewTestContext(ctx, req, res))
			if err != nil {
				fmt.Printf("Test\nerror: %#v\n", err)
			} else {
				boolOutput, boolOk := output.(bool)
				if boolOk {
					boolResultStr := "Failure"
					if boolOutput {
						boolResultStr = "Success"
					}
					fmt.Printf("Test: %s\n", boolResultStr)
				} else {
					fmt.Printf("Test: `%s` = %s\n", st.Test, output)
				}
			}
		}
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

func ShowVerbose(i int, name string, req, res map[string]any) {
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
