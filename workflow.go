package probe

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

type Workflow struct {
	Name        string         `yaml:"name",validate:"required"`
	Description string         `yaml:"description,omitempty"`
	Jobs        []Job          `yaml:"jobs",validate:"required"`
	Vars        map[string]any `yaml:"vars"`
	exitStatus  int
	env         map[string]string
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
	output := NewOutput(c.Verbose)

	// Print workflow header at the beginning
	output.PrintWorkflowHeader(w.Name, w.Description)

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
			// Set IsRepeating to false for jobs without repeat
			jobCtx := ctx
			jobCtx.IsRepeating = false
			jobCtx.RepeatTotal = 1
			jobCtx.RepeatCurrent = 1
			jobCtx.StepCounters = make(map[int]StepRepeatCounter)
			
			wg.Add(1)
			go func(j Job, jCtx JobContext) {
				defer wg.Done()
				w.SetExitStatus(j.Start(jCtx))
			}(job, jobCtx)
			continue
		}

		// Repeat
		for i := 0; i < job.Repeat.Count; i++ {
			// Set IsRepeating to true for jobs with repeat
			jobCtx := ctx
			jobCtx.IsRepeating = true
			jobCtx.RepeatTotal = job.Repeat.Count
			jobCtx.RepeatCurrent = i + 1
			jobCtx.StepCounters = make(map[int]StepRepeatCounter)
			
			wg.Add(1)
			go func(j Job, jCtx JobContext) {
				defer wg.Done()
				w.SetExitStatus(j.Start(jCtx))
			}(job, jobCtx)
			time.Sleep(job.Repeat.Interval.Duration)
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
		output := NewOutput(ctx.Verbose)
		w.printDetailedResults(workflowOutput, output)
	}

	return nil
}

// executeJobWithRepeat handles the execution of a job with repeat support
func (w *Workflow) executeJobWithRepeat(scheduler *JobScheduler, job *Job, jobID string, ctx JobContext) {
	overallSuccess := true

	// Initialize repeat tracking
	_, total := scheduler.GetRepeatInfo(jobID)
	ctx.IsRepeating = total > 1
	ctx.RepeatTotal = total
	ctx.StepCounters = make(map[int]StepRepeatCounter)

	// Execute job with repeat logic
	for scheduler.ShouldRepeatJob(jobID) {
		current, _ := scheduler.GetRepeatInfo(jobID)
		ctx.RepeatCurrent = current

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
			time.Sleep(job.Repeat.Interval.Duration)
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

	// Initialize repeat tracking and buffering flag
	_, total := scheduler.GetRepeatInfo(jobID)
	ctx.IsRepeating = total > 1
	ctx.RepeatTotal = total
	ctx.StepCounters = make(map[int]StepRepeatCounter)
	ctx.UseBuffering = true // Set buffering flag to prevent duplicate job names

	// Store status update (don't print immediately)
	// Status updates will be shown in detailed results

	overallSuccess := true

	// Execute job with repeat logic
	for scheduler.ShouldRepeatJob(jobID) {
		current, _ := scheduler.GetRepeatInfo(jobID)
		ctx.RepeatCurrent = current
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
			time.Sleep(job.Repeat.Interval.Duration)
		}
	}

	// Print repeat step results after all executions complete
	if ctx.IsRepeating {
		w.printRepeatStepResults(&ctx, job, jo)
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
func (w *Workflow) printDetailedResults(wo *WorkflowOutput, output OutputWriter) {
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

		status := StatusSuccess
		if jo.Status == "Skipped" {
			status = StatusWarning
		} else if !jo.Success {
			status = StatusError
		} else {
			successCount++
		}

		output.PrintJobResult(jo.JobName, status, duration.Seconds())
		output.PrintJobOutput(jo.Buffer.String())

		jo.mutex.Unlock()
	}

	// Print summary
	totalJobs := len(wo.Jobs)
	output.PrintWorkflowSummary(totalTime.Seconds(), successCount, totalJobs)
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
		Output: NewOutput(c.Verbose),
	}
}


type StepRepeatCounter struct {
	SuccessCount int
	FailureCount int
	Name         string
	LastResult   bool
	Output       strings.Builder
}

// printRepeatStepResults prints the final results of repeat step executions to buffer
func (w *Workflow) printRepeatStepResults(ctx *JobContext, job *Job, jo *JobOutput) {
	// Capture the step results output to buffer instead of printing directly
	originalStdout := os.Stdout
	r, wr, _ := os.Pipe()
	os.Stdout = wr
	
	output := NewOutput(ctx.Config.Verbose)
	
	for i, step := range job.Steps {
		if counter, exists := ctx.StepCounters[i]; exists {
			hasTest := step.Test != ""
			output.PrintStepRepeatResult(i, counter, hasTest)
		}
	}
	
	// Restore stdout and capture the step results
	wr.Close()
	os.Stdout = originalStdout
	
	capturedOutput, _ := io.ReadAll(r)
	
	// Add captured step results to job buffer
	jo.mutex.Lock()
	jo.Buffer.Write(capturedOutput)
	jo.mutex.Unlock()
}
