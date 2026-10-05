package probe

import (
	"fmt"
	"sync"
	"time"

	"github.com/linyows/probe/expr"
	"github.com/linyows/probe/mask"
	"github.com/linyows/probe/procgroup"
)

type Job struct {
	Name     string   `yaml:"name" validate:"required"`
	ID       string   `yaml:"id,omitempty"`
	Needs    []string `yaml:"needs,omitempty"`
	Steps    []*Step  `yaml:"steps" validate:"required"`
	Repeat   *Repeat  `yaml:"repeat"`
	Defaults any      `yaml:"defaults"`
	SkipIf   string   `yaml:"skipif,omitempty"`
}

func (j *Job) Start(ctx JobContext) error {
	failed, err := j.run(ctx)
	if err != nil {
		return err
	}
	if failed {
		return errStepsFailed(j.Name)
	}
	return nil
}

// errStepsFailed is the error Start returns when a step of the job failed.
func errStepsFailed(jobName string) error {
	return NewExecutionError("job_start", "job execution failed", nil).
		WithContext("job_name", jobName)
}

// run runs the job. It reports whether a step failed separately from the
// error that kept the job from running at all, such as an invalid step.
func (j *Job) run(ctx JobContext) (failed bool, err error) {
	// Set current job ID in context (already set by Executor.setJobID())
	ctx.CurrentJobID = j.ID

	// Use local pointer instead of j.ctx to avoid race conditions in async mode
	ctxPtr := &ctx
	ev := &expr.Expr{}

	// Validate steps before execution
	if err := j.validateSteps(); err != nil {
		ctx.Result.recordFailure(failureConfig)
		return false, NewExecutionError("job_start", "step validation failed", err)
	}

	if err := j.expandJobName(ev, ctxPtr); err != nil {
		ctx.Result.recordFailure(failureConfig)
		return false, NewExecutionError("job_start", "failed to expand job name", err)
	}

	// Check if job should be skipped
	if j.shouldSkip(ev, *ctxPtr) {
		j.handleSkip(*ctxPtr)
		return false, nil
	}

	j.executeSteps(ev, ctxPtr)
	return ctxPtr.Failed, nil
}

// ApplyDefaults merges the job's defaults into the with of every step that
// uses the action they are keyed by. A value the step sets itself wins, and
// nested objects are merged key by key.
func (j *Job) ApplyDefaults() {
	byAction, ok := j.Defaults.(map[string]any)
	if !ok {
		return
	}
	for uses, values := range byAction {
		defaults, ok := values.(map[string]any)
		if !ok {
			continue
		}
		for _, s := range j.Steps {
			if s.Uses != uses {
				continue
			}
			if s.With == nil {
				s.With = make(map[string]any)
			}
			mergeDefaults(s.With, defaults)
		}
	}
}

// mergeDefaults sets each key of defaults that data lacks, recursing into
// objects that both have.
func mergeDefaults(data, defaults map[string]any) {
	for key, defaultValue := range defaults {
		if _, exists := data[key]; !exists {
			data[key] = defaultValue
			continue
		}
		if nestedDefault, ok := defaultValue.(map[string]any); ok {
			if nestedData, ok := data[key].(map[string]any); ok {
				mergeDefaults(nestedData, nestedDefault)
			}
		}
	}
}

// expandJobName evaluates and sets the job name, printing it if appropriate
func (j *Job) expandJobName(ev *expr.Expr, ctx *JobContext) error {
	if j.Name == "" {
		j.Name = "Unknown Job"
		return nil
	}

	name, err := ev.EvalTemplate(j.Name, *ctx)
	if err != nil {
		return err
	}

	j.Name = name

	// Update the job name in the result as well. Async repeat shares the
	// JobResult across iterations, so writers must take the mutex.
	if ctx.Result != nil {
		if jobResult, exists := ctx.Result.Jobs[j.ID]; exists {
			jobResult.mutex.Lock()
			jobResult.JobName = name
			jobResult.mutex.Unlock()
		}
	}

	return nil
}

// cloneForAsync returns a copy of the job with its own Steps slice so that
// each iteration of an async repeat can mutate per-execution state
// (Job.Name, Step.Expr, Step.ctx, Step.Idx, Step.startedAt, Step.err,
// Step.retryAttempt) without racing with sibling iterations or the
// originally configured Job. Pointer-indirected configuration fields
// (Repeat, Outputs, Retry, Vars, With, Iteration, Defaults, actionRunner)
// remain shared because they are read-only at execution time or already
// guarded by their own synchronization.
func (j *Job) cloneForAsync() *Job {
	cloned := *j
	cloned.Steps = make([]*Step, len(j.Steps))
	for i, s := range j.Steps {
		stepCopy := *s
		cloned.Steps[i] = &stepCopy
	}
	return &cloned
}

// executeSteps runs all steps in the job, handling iterations appropriately
func (j *Job) executeSteps(ev *expr.Expr, ctx *JobContext) {
	idx := 0
	for _, st := range j.Steps {
		st.Expr = ev

		if len(st.Iteration) == 0 {
			j.executeStep(st, &idx, ctx, nil)
		} else {
			j.executeStepWithIterations(st, &idx, ctx)
		}
	}
}

// executeStep executes a single step without iterations
func (j *Job) executeStep(st *Step, idx *int, ctx *JobContext, vars map[string]any) {
	st.Idx = *idx
	*idx++
	st.SetCtx(*ctx, vars)
	st.Do(ctx)
}

// executeStepWithIterations executes a step multiple times with different variable sets
func (j *Job) executeStepWithIterations(st *Step, idx *int, ctx *JobContext) {
	for _, vars := range st.Iteration {
		j.executeStep(st, idx, ctx, vars)
	}
}

// validateSteps validates step configurations in the job
func (j *Job) validateSteps() error {
	stepIDs := make(map[string]int) // stepID -> stepIndex for duplicate check

	for i, step := range j.Steps {
		// Validate step ID format if provided
		if step.ID != "" {
			if !isValidStepID(step.ID) {
				return fmt.Errorf("step %d: invalid step ID '%s' - only [a-z0-9_-] characters are allowed", i, step.ID)
			}

			// Check for duplicate step IDs within the job
			if existingIndex, exists := stepIDs[step.ID]; exists {
				return fmt.Errorf("step %d: duplicate step ID '%s' (already used in step %d)", i, step.ID, existingIndex)
			}
			stepIDs[step.ID] = i
		}

		// Validate that outputs require an ID
		if len(step.Outputs) > 0 && step.ID == "" {
			stepName := step.Name
			if stepName == "" {
				stepName = fmt.Sprintf("step %d", i)
			}
			return fmt.Errorf("%s: step with outputs must have an 'id' field", stepName)
		}
	}

	return nil
}

// isValidStepID validates step ID format: only [a-z0-9_-] allowed
func isValidStepID(id string) bool {
	if id == "" {
		return false
	}

	// Check each character
	for _, char := range id {
		if (char < 'a' || char > 'z') &&
			(char < '0' || char > '9') &&
			char != '_' && char != '-' {
			return false
		}
	}

	return true
}

// shouldSkip evaluates the skipif expression and returns true if job should be skipped
func (j *Job) shouldSkip(ev *expr.Expr, ctx JobContext) bool {
	if j.SkipIf == "" {
		return false
	}

	// Create a step context for evaluation - same as SetCtx in step.go
	var outputs map[string]any
	if ctx.Outputs != nil {
		outputs = ctx.Outputs.GetAll()
	}

	// Create context for job skipif evaluation
	evalCtx := StepContext{
		Vars:    ctx.Vars,
		Outputs: outputs,
	}

	result, err := ev.Eval(j.SkipIf, evalCtx)
	if err != nil {
		ctx.Printer.PrintError("job skipif evaluation error: %v", err)
		return false // Don't skip on evaluation error
	}

	boolResult, ok := result.(bool)
	if !ok {
		ctx.Printer.PrintError("job skipif expression must return boolean, got: %T", result)
		return false // Don't skip on type error
	}

	return boolResult
}

// handleSkip handles the skipped job logic
func (j *Job) handleSkip(ctx JobContext) {
	if ctx.Verbose {
		ctx.Printer.LogDebug("Job '%s' (SKIPPED)", j.Name)
		ctx.Printer.LogDebug("Skip condition: %s", j.SkipIf)
		ctx.Printer.PrintSeparator()
	}

	// Mark job as skipped in the result. Async repeat shares the
	// JobResult across iterations, so writers must take the mutex
	// to stay race-free with sibling iterations and Executor.finalize.
	if ctx.Result != nil {
		if jobResult, exists := ctx.Result.Jobs[j.ID]; exists {
			jobResult.mutex.Lock()
			jobResult.Status = "skipped"
			jobResult.Success = true // Skipped jobs are considered successful
			jobResult.mutex.Unlock()
		}
	}
}

// JobRun is the outcome of a job run on its own with RunStandalone.
type JobRun struct {
	// Success is true when every step passed or the job was skipped.
	Success bool
	// Outputs are the outputs the job's steps published.
	Outputs map[string]any
	// Report is the report of the job's steps.
	Report string
	// Err is set when the job could not run as it was written, such as when
	// a step is invalid or an action cannot be resolved. A step that fails
	// leaves it nil and Success false.
	Err error
	// Duration is how long the job took.
	Duration time.Duration
}

// RunStandalone runs the job outside a workflow, as an embedded job is run.
// A local action is resolved relative to baseDir, the directory of the file
// the job comes from, or to the working directory when baseDir is empty.
func (j *Job) RunStandalone(vars map[string]any, printer *Printer, jobID, baseDir string) JobRun {
	run, _ := j.runStandalone(vars, printer, jobID, baseDir)
	return run
}

// RunIndependently executes a job independently with its own context and result tracking
// Returns success/failure status, outputs, report, error message, and duration
func (j *Job) RunIndependently(vars map[string]any, printer *Printer, jobID string) (bool, map[string]any, string, string, time.Duration) {
	run, failed := j.runStandalone(vars, printer, jobID, "")
	errorMsg := ""
	switch {
	case run.Err != nil:
		errorMsg = run.Err.Error()
	case failed:
		errorMsg = errStepsFailed(j.Name).Error()
	}
	return run.Success, run.Outputs, run.Report, errorMsg, run.Duration
}

// runStandalone is RunStandalone, also reporting whether a step failed.
func (j *Job) runStandalone(vars map[string]any, printer *Printer, jobID, baseDir string) (JobRun, bool) {
	start := time.Now()
	j.ID = jobID
	// The job runs outside Workflow.Start, which is what installs a masker.
	// One is installed here, so that credentials the job's own steps pass to
	// actions are learned and hidden as they are in a workflow.
	if printer.Masker() == nil {
		printer.SetMasker(mask.New(nil, nil))
	}
	result := NewResult()
	jr := &JobResult{
		JobName:   j.Name,
		JobID:     jobID,
		StartTime: start,
	}
	result.Jobs[jobID] = jr

	ctx := JobContext{
		Vars:    vars,
		Outputs: NewOutputs(),
		Result:  result,
		Config: Config{
			Verbose: printer.verbose,
		},
		Printer:    printer,
		countersMu: &sync.Mutex{},
		background: procgroup.NewTracker(),
		baseDir:    baseDir,
	}
	// This job runs inside the plugin process of the step that embeds it,
	// which exits once the job is done, so its background processes go then.
	// The watch for signals ends after the stop, as in a workflow.
	endWatch := ctx.background.StopOnSignal()
	defer endWatch()
	defer ctx.background.Stop()

	// External actions are resolved before any step runs, as a workflow
	// does, so that a bad reference stops the job before it changes anything.
	failed, err := false, resolveExternalActions([]*Job{j}, baseDir)
	if err == nil {
		failed, err = j.run(ctx)
	}

	run := JobRun{Err: err, Success: err == nil && !failed}
	if run.Success {
		jr.Status = "Completed"
	} else {
		jr.Status = "Failed"
	}
	jr.Success = run.Success

	run.Duration = time.Since(start)
	jr.EndTime = jr.StartTime.Add(run.Duration)
	run.Outputs = ctx.Outputs.GetAll()
	run.Report = ctx.Printer.GenerateReportOnlySteps(result)

	return run, failed
}
