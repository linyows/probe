package probe

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/linyows/probe/expr"
	"github.com/linyows/probe/mask"
	"github.com/linyows/probe/procgroup"
	"github.com/linyows/probe/report"
)

type Workflow struct {
	Name        string         `yaml:"name" validate:"required"`
	Description string         `yaml:"description,omitempty"`
	Jobs        []Job          `yaml:"jobs" validate:"required"`
	Vars        map[string]any `yaml:"vars"`
	// Secrets names environment variables whose values must not appear in
	// anything Probe prints or writes.
	Secrets    []string `yaml:"secrets,omitempty"`
	exitStatus int
	// failed is set to 1 by any job that does not succeed. Jobs run
	// concurrently, so it is only accessed atomically; exitStatus is derived
	// from it once every job is done. It is an int32 rather than an
	// atomic.Bool because a Workflow is copied by value when it is decoded.
	failed int32
	env    map[string]string
	// basePath is the directory containing the workflow file (used for resolving relative paths)
	basePath string
	// Shared outputs across all jobs
	outputs *Outputs
	printer *Printer
}

// Start executes the workflow with the given configuration
func (w *Workflow) Start(c Config) error {
	// Build the scheduler first: it assigns an ID to every job that omits one,
	// and the result, the report order and the scheduler all index jobs by ID,
	// so they have to agree before anything keys off them.
	scheduler, err := w.initJobScheduler()
	if err != nil {
		return err
	}

	// Collect all job IDs for buffer initialization
	jobIDs := make([]string, len(w.Jobs))
	for i, job := range w.Jobs {
		jobIDs[i] = job.ID
	}

	if w.printer == nil {
		w.printer = NewPrinter(c.Verbose, jobIDs)
	} else {
		// A caller cannot know a generated ID in advance, so Start owns the
		// order the report is rendered in.
		w.printer.SetBufferIDs(jobIDs)
	}

	// Install the masker before anything is printed, the header included.
	w.printer.SetMasker(mask.New(w.Secrets, w.Env()))

	// A reporter tracks how far the report has been emitted, which is state
	// for this run alone, so every run gets a fresh one.
	reporter := newReporter(c.Output, w.printer)
	w.printer.SetReporter(reporter)

	startedAt := time.Now()
	reporter.Start(w.Name, w.Description)

	// Initialize shared outputs
	if w.outputs == nil {
		w.outputs = NewOutputs()
	}

	vars, err := w.evalVars()
	if err != nil {
		return err
	}

	ctx := w.newJobContext(c, vars, scheduler)
	// A process a step started in the background lives on after its step,
	// for later steps and jobs to use, but not after the workflow, and not
	// after probe is interrupted. The watch for signals is deferred first so
	// that it ends last, still on while the processes are being stopped.
	endWatch := ctx.background.StopOnSignal()
	defer endWatch()
	defer ctx.background.Stop()

	if err := w.startJobsWithDependencies(ctx); err != nil {
		return err
	}

	reporter.Finish(ctx.Result)
	w.exitStatus = ctx.Result.exitCode(atomic.LoadInt32(&w.failed) == 1)

	return w.writeReports(c.Reports, ctx.Result, jobIDs, startedAt, time.Now())
}

// writeReports writes every requested report file. A file that cannot be
// written does not stop the others; the errors are returned together.
func (w *Workflow) writeReports(targets []report.Target, rs *Result, order []string, startedAt, finishedAt time.Time) error {
	if len(targets) == 0 {
		return nil
	}

	r := BuildReport(w.Name, w.Description, rs, order, startedAt, finishedAt)
	r.Mask(w.printer.Masker())

	var errs []error
	for _, t := range targets {
		err := r.Write(t)
		if errors.Is(err, report.ErrNoStepSummary) || errors.Is(err, report.ErrStepSummaryFull) {
			// Outside GitHub Actions there is no summary to write to, and a
			// full one has no room; say so and carry on, so one command line
			// serves CI and a laptop and a full summary fails no run.
			w.printer.LogWarn("%v", err)
			continue
		}
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// initJobScheduler creates and sets up the job scheduler with dependencies
func (w *Workflow) initJobScheduler() (*JobScheduler, error) {
	scheduler := NewJobScheduler()

	// Add all jobs to scheduler
	for i := range w.Jobs {
		if err := scheduler.AddJob(&w.Jobs[i]); err != nil {
			return nil, err
		}
	}

	// Validate dependencies
	if err := scheduler.ValidateDependencies(); err != nil {
		return nil, err
	}

	return scheduler, nil
}

// setupResult creates result for managing execution results
func (w *Workflow) setupResult() *Result {
	rs := NewResult()
	// Initialize job outputs
	for _, job := range w.Jobs {
		jobID := job.ID
		if jobID == "" {
			jobID = job.Name
		}
		jr := &JobResult{
			JobName:   job.Name,
			JobID:     jobID,
			StartTime: time.Now(),
		}
		rs.Jobs[jobID] = jr
	}

	return rs
}

// startJobsWithDependencies runs the main job execution loop with dependency management
func (w *Workflow) startJobsWithDependencies(ctx JobContext) error {

	for !ctx.JobScheduler.AllJobsCompleted() {
		runnableJobs := ctx.JobScheduler.GetRunnableJobs()

		if len(runnableJobs) == 0 {
			if err := w.handleNoRunnableJobs(ctx); err != nil {
				return err
			}
			continue
		}

		w.processRunnableJobs(runnableJobs, ctx)
		ctx.JobScheduler.wg.Wait()
	}

	return nil
}

// handleNoRunnableJobs handles the case when no jobs can be run (failed dependencies or deadlock)
func (w *Workflow) handleNoRunnableJobs(ctx JobContext) error {
	skippedJobs := ctx.JobScheduler.MarkJobsWithFailedDependencies()

	// Update skipped jobs in workflow printer
	if ctx.Result != nil {
		w.updateSkippedJobsOutput(skippedJobs, ctx.Result)
	}

	if len(skippedJobs) == 0 {
		// If no jobs were skipped, we might have a deadlock
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

// updateSkippedJobsOutput updates the output for jobs that were skipped due to failed dependencies
func (w *Workflow) updateSkippedJobsOutput(skippedJobs []string, rs *Result) {
	for _, jobID := range skippedJobs {
		if jr, exists := rs.Jobs[jobID]; exists {
			jr.mutex.Lock()
			jr.EndTime = jr.StartTime // Set end time same as start time (0 duration)
			jr.Status = "skipped"
			jr.Success = true // Skipped jobs are considered successful (same as skipif)
			jr.mutex.Unlock()

			// These jobs never reach Executor.finalize, so announce them here
			// or the streaming report would stall on them.
			rs.notifyJobDone(jobID)
		}
	}
}

// processRunnableJobs starts execution of all currently runnable jobs
func (w *Workflow) processRunnableJobs(runnableJobs []string, ctx JobContext) {
	for _, jobID := range runnableJobs {
		job := ctx.JobScheduler.jobs[jobID]
		ctx.JobScheduler.SetJobStatus(jobID, JobRunning, false)
		ctx.JobScheduler.wg.Add(1)

		go func(j *Job, id string) {
			defer ctx.JobScheduler.wg.Done()

			executor := NewExecutor(w, j)
			success := executor.Execute(ctx)
			if !success {
				w.SetExitStatus(true)
			}
		}(job, jobID)
	}
}

func (w *Workflow) SetExitStatus(isErr bool) {
	if isErr {
		atomic.StoreInt32(&w.failed, 1)
	}
}

func (w *Workflow) Env() map[string]string {
	if len(w.env) == 0 {
		w.env = envMap()
	}
	return w.env
}

// evalVars evaluates template variables in workflow vars using environment variables
func (w *Workflow) evalVars() (map[string]any, error) {
	env := strmapToAnymap(w.Env())
	vars := make(map[string]any)

	ev := &expr.Expr{}
	for k, v := range w.Vars {
		if mapV, ok := v.(map[string]any); ok {
			vars[k] = ev.EvalTemplateMap(mapV, env)
		} else if strV, ok2 := v.(string); ok2 {
			output, err := ev.EvalTemplate(strV, env)
			if err != nil {
				return vars, err
			}
			vars[k] = output
		} else {
			// Handle other types directly (bool, int, float, etc.)
			vars[k] = v
		}
	}

	return vars, nil
}

func (w *Workflow) newJobContext(c Config, vars map[string]any, scheduler *JobScheduler) JobContext {
	rs := w.setupResult()
	rs.SetReporter(w.printer.Reporter())

	return JobContext{
		Vars:         vars,
		Config:       c,
		Printer:      w.printer,
		Result:       rs,
		JobScheduler: scheduler,
		Outputs:      w.outputs,
		countersMu:   &sync.Mutex{},
		background:   procgroup.NewTracker(),
	}
}
