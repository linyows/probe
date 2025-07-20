package probe

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
)

// colorSuccess returns a *color.Color for success (RGB 0,175,0)
func colorSuccess() *color.Color {
	return color.RGB(0, 175, 0)
}

// colorError returns a *color.Color for errors (red)
func colorError() *color.Color {
	return color.New(color.FgRed)
}

// colorWarning returns a *color.Color for warnings (blue)
func colorWarning() *color.Color {
	return color.New(color.FgBlue)
}

type Workflow struct {
	Name       string         `yaml:"name",validate:"required"`
	Jobs       []Job          `yaml:"jobs",validate:"required"`
	Vars       map[string]any `yaml:"vars"`
	exitStatus int
	env        map[string]string
}

// JobOutput stores buffered output for a job
type JobOutput struct {
	JobName   string
	JobID     string
	Buffer    strings.Builder
	Status    string
	StartTime time.Time
	EndTime   time.Time
	Success   bool
	mutex     sync.Mutex
}

// WorkflowOutput manages output for multiple jobs
type WorkflowOutput struct {
	Jobs        map[string]*JobOutput
	mutex       sync.RWMutex
	outputMutex sync.Mutex // Protects stdout redirection
}

func NewWorkflowOutput() *WorkflowOutput {
	return &WorkflowOutput{
		Jobs: make(map[string]*JobOutput),
	}
}

func (w *Workflow) SetExitStatus(isErr bool) {
	if isErr {
		w.exitStatus = 1
	}
}

func (w *Workflow) Start(c Config) error {
	// Print workflow name at the beginning
	if w.Name != "" {
		fmt.Printf("%s\n", color.New(color.Bold).Sprint(w.Name))
	}

	vars, err := w.evalVars()
	if err != nil {
		return err
	}

	ctx := w.newJobContext(c, vars)

	// Check if any job has dependencies
	hasDependencies := false
	for _, job := range w.Jobs {
		if len(job.Needs) > 0 {
			hasDependencies = true
			break
		}
	}

	// If no dependencies, use the old parallel execution
	if !hasDependencies {
		return w.startParallel(ctx)
	}

	// Use dependency-aware execution
	return w.startWithDependencies(ctx)
}

func (w *Workflow) startParallel(ctx JobContext) error {
	var wg sync.WaitGroup

	for _, job := range w.Jobs {
		// No repeat
		if job.Repeat == nil {
			wg.Add(1)
			go func(j Job) {
				defer wg.Done()
				w.SetExitStatus(j.Start(ctx))
			}(job)
			continue
		}

		// Repeat
		for i := 0; i < job.Repeat.Count; i++ {
			wg.Add(1)
			go func(j Job) {
				defer wg.Done()
				w.SetExitStatus(j.Start(ctx))
			}(job)
			time.Sleep(time.Duration(job.Repeat.Interval) * time.Second)
		}
	}

	wg.Wait()
	return nil
}

func (w *Workflow) startWithDependencies(ctx JobContext) error {
	scheduler := NewJobScheduler()

	// Add all jobs to scheduler
	for i := range w.Jobs {
		if err := scheduler.AddJob(&w.Jobs[i]); err != nil {
			return err
		}
	}

	// Validate dependencies
	if err := scheduler.ValidateDependencies(); err != nil {
		return err
	}

	// Use buffered output if multiple jobs
	useBuffering := len(w.Jobs) > 1
	var workflowOutput *WorkflowOutput
	if useBuffering {
		workflowOutput = NewWorkflowOutput()
		// Initialize job outputs
		for _, job := range w.Jobs {
			jobID := job.ID
			if jobID == "" {
				jobID = job.Name
			}
			jo := &JobOutput{
				JobName:   job.Name,
				JobID:     jobID,
				StartTime: time.Now(),
			}
			workflowOutput.Jobs[jobID] = jo
		}
	}

	// Execute jobs with dependency control and repeat support
	for !scheduler.AllJobsCompleted() {
		runnableJobs := scheduler.GetRunnableJobs()

		if len(runnableJobs) == 0 {
			// Check if there are pending jobs but none can run
			// This might indicate failed dependencies
			skippedJobs := scheduler.MarkJobsWithFailedDependencies()
			// Update skipped jobs in workflow output
			if useBuffering {
				for _, jobID := range skippedJobs {
					if jo, exists := workflowOutput.Jobs[jobID]; exists {
						jo.mutex.Lock()
						jo.EndTime = jo.StartTime // Set end time same as start time (0 duration)
						jo.Status = "Skipped"
						jo.Success = false
						jo.mutex.Unlock()
					}
				}
			}
			if len(skippedJobs) == 0 {
				// If no jobs were skipped, we might have a deadlock
				time.Sleep(100 * time.Millisecond)
				continue
			}
		}

		// Start all runnable jobs
		for _, jobID := range runnableJobs {
			job := scheduler.jobs[jobID]
			scheduler.SetJobStatus(jobID, JobRunning, false)
			scheduler.wg.Add(1)

			go func(j *Job, id string) {
				defer scheduler.wg.Done()
				if useBuffering {
					w.executeJobWithBuffering(scheduler, j, id, ctx, workflowOutput)
				} else {
					w.executeJobWithRepeat(scheduler, j, id, ctx)
				}
			}(job, jobID)
		}

		// Wait for current batch to complete
		scheduler.wg.Wait()
	}

	// Print detailed results if buffering was used
	if useBuffering {
		w.printDetailedResults(workflowOutput)
	}

	return nil
}

// executeJobWithRepeat handles the execution of a job with repeat support
func (w *Workflow) executeJobWithRepeat(scheduler *JobScheduler, job *Job, jobID string, ctx JobContext) {
	overallSuccess := true

	// Execute job with repeat logic
	for scheduler.ShouldRepeatJob(jobID) {
		// Execute single run
		success := !job.Start(ctx)

		if !success {
			overallSuccess = false
			w.SetExitStatus(true)
		}

		// Increment counter
		scheduler.IncrementRepeatCounter(jobID)

		// Sleep between repeats (except for the last one)
		current, target := scheduler.GetRepeatInfo(jobID)
		if current < target && job.Repeat != nil {
			time.Sleep(time.Duration(job.Repeat.Interval) * time.Second)
		}
	}

	// Mark job as completed
	scheduler.SetJobStatus(jobID, JobCompleted, overallSuccess)
}

// executeJobWithBuffering handles job execution with buffered output
func (w *Workflow) executeJobWithBuffering(scheduler *JobScheduler, job *Job, jobID string, ctx JobContext, wo *WorkflowOutput) {
	jo := wo.Jobs[jobID]
	jo.mutex.Lock()
	jo.StartTime = time.Now()
	jo.Status = "Running"
	jo.mutex.Unlock()

	// Store status update (don't print immediately)
	// Status updates will be shown in detailed results

	overallSuccess := true

	// Execute job with repeat logic
	for scheduler.ShouldRepeatJob(jobID) {
		// Serialize stdout redirection to prevent race conditions
		wo.outputMutex.Lock()

		// Capture output by redirecting stdout temporarily
		originalStdout := os.Stdout
		r, wr, _ := os.Pipe()
		os.Stdout = wr

		// Execute single run
		success := !job.Start(ctx)

		// Restore stdout and capture output
		wr.Close()
		os.Stdout = originalStdout

		wo.outputMutex.Unlock()

		capturedOutput, _ := io.ReadAll(r)

		// Add captured output to buffer
		jo.mutex.Lock()
		jo.Buffer.Write(capturedOutput)
		jo.mutex.Unlock()

		if !success {
			overallSuccess = false
			w.SetExitStatus(true)
		}

		// Increment counter
		scheduler.IncrementRepeatCounter(jobID)

		// Sleep between repeats (except for the last one)
		current, target := scheduler.GetRepeatInfo(jobID)
		if current < target && job.Repeat != nil {
			time.Sleep(time.Duration(job.Repeat.Interval) * time.Second)
		}
	}

	// Update final status (don't print immediately)
	jo.mutex.Lock()
	jo.EndTime = time.Now()
	jo.Success = overallSuccess
	if overallSuccess {
		jo.Status = "Completed"
	} else {
		jo.Status = "Failed"
	}
	jo.mutex.Unlock()

	// Mark job as completed
	scheduler.SetJobStatus(jobID, JobCompleted, overallSuccess)
}

// printDetailedResults prints the final detailed results
func (w *Workflow) printDetailedResults(wo *WorkflowOutput) {
	fmt.Println()

	totalTime := time.Duration(0)
	successCount := 0

	// Process jobs in original order
	for _, job := range w.Jobs {
		jobID := job.ID
		if jobID == "" {
			jobID = job.Name
		}
		jo, exists := wo.Jobs[jobID]
		if !exists {
			continue
		}

		jo.mutex.Lock()
		duration := jo.EndTime.Sub(jo.StartTime)
		totalTime += duration

		statusColor := colorSuccess().SprintFunc()
		statusIcon := "⏺ "
		if jo.Status == "Skipped" {
			statusColor = colorWarning().SprintFunc()
		} else if !jo.Success {
			statusColor = colorError().SprintFunc()
		} else {
			successCount++
		}

		fmt.Printf("%s%s (%s in %.2fs)\n",
			statusColor(statusIcon),
			jo.JobName,
			jo.Status,
			duration.Seconds())

		// Print buffered output with proper indentation
		output := strings.TrimSpace(jo.Buffer.String())
		if output != "" {
			lines := strings.Split(output, "\n")
			for i, line := range lines {
				if strings.TrimSpace(line) != "" {
					if i == 0 {
						fmt.Printf("  ⎿ %s\n", line)
					} else {
						fmt.Printf("    %s\n", line)
					}
				}
			}
		}
		fmt.Println()

		jo.mutex.Unlock()
	}

	// Print summary
	totalJobs := len(wo.Jobs)
	if successCount == totalJobs {
		fmt.Printf("Total workflow time: %.2fs %s\n",
			totalTime.Seconds(),
			colorSuccess().Sprintf("✔︎ All jobs succeeded"))
	} else {
		failedCount := totalJobs - successCount
		fmt.Printf("Total workflow time: %.2fs %s\n",
			totalTime.Seconds(),
			colorError().Sprintf("✘ %d job(s) failed", failedCount))
	}
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
	Name     string   `yaml:"name",validate:"required"`
	ID       string   `yaml:"id,omitempty"`
	Needs    []string `yaml:"needs,omitempty"`
	Steps    []*Step  `yaml:"steps",validate:"required"`
	Repeat   *Repeat  `yaml:"repeat"`
	Defaults any      `yaml:"defaults"`
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
		j.Name = name
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
		fmt.Printf("%s \"%s\" in %s-action -- %s\n", colorError().Sprintf("Error"), name, st.Uses, err)
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
			output = fmt.Sprintf(output+"\n", colorSuccess().Sprintf("✔︎ "))
		} else {
			output = fmt.Sprintf(output+"\n"+str+"\n", colorError().Sprintf("✘ "))
			jCtx.SetFailed()
		}
	} else {
		output = fmt.Sprintf(output+"\n", colorWarning().Sprintf("▲ "))
	}
	fmt.Print(output)

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

	return boolOk
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
